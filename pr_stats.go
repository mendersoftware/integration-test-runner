package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"

	clientgithub "github.com/mendersoftware/integration-test-runner/client/github"
)

type PRData struct {
	Number            int
	Title             string
	URL               string
	Author            string
	Assignees         []string
	State             string
	Draft             bool
	CreatedAt         time.Time
	TimeToFirstReview time.Duration
	TimeToClose       time.Duration
	Repo              string
}

type UserStats struct {
	Login           string
	Opened          int
	Reviewed        int
	Closed          int
	ReviewTimes     []time.Duration
	CloseTimes      []time.Duration
	CurrentOpen     int
	CurrentAssigned int
}

type PRStatsOptions struct {
	Repos          []string
	RepoLabel      string
	ExcludeDrafts  bool
	ExcludedUsers  map[string]bool
	ExcludedLabels map[string]bool
	SLAHours       int
	Mode           string
}

type GlobalConfig struct {
	ExcludedUsers  []string `json:"excluded_users"`
	ExcludedLabels []string `json:"excluded_labels"`
	ExcludeDrafts  bool     `json:"exclude_drafts"`
	SLAHours       int      `json:"sla_hours"`
}

type TeamConfig struct {
	Name             string   `json:"name"`
	Repositories     []string `json:"repositories"`
	FastRepositories []string `json:"fast_repositories"`
}

type PRStatsConfig struct {
	Global GlobalConfig `json:"global"`
	Teams  []TeamConfig `json:"teams"`
}

func loadPRStatsConfig(path string) (*PRStatsConfig, error) {
	if path == "" {
		path = os.Getenv("PR_STATS_CONFIG_PATH")
		if path == "" {
			path = "pr_stats_config.json"
			if _, err := os.Stat(path); os.IsNotExist(err) {
				path = "/pr_stats_config.json"
			}
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		logrus.Warnf("Could not read config file at %s: %s", path, err)
		return nil, err
	}
	var config PRStatsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	logrus.Infof("Successfully loaded PR Stats config from %s", path)
	return &config, nil
}

func getTeamRepos(currentRepo string, config *PRStatsConfig, slow bool) ([]string, string) {
	if config == nil {
		return []string{currentRepo}, ""
	}
	for _, team := range config.Teams {
		isTeamRepo := false
		for _, repo := range team.Repositories {
			if repo == currentRepo {
				isTeamRepo = true
				break
			}
		}
		if isTeamRepo {
			if slow {
				return team.Repositories, team.Name + " Team (All Repos)"
			}
			repos := team.FastRepositories
			found := false
			for _, r := range repos {
				if r == currentRepo {
					found = true
					break
				}
			}
			if !found {
				repos = append([]string{currentRepo}, repos...)
			}
			return repos, team.Name + " Team (Fast Mode)"
		}
	}
	return []string{currentRepo}, ""
}

func ensureUser(m map[string]*UserStats, login string) *UserStats {
	if _, ok := m[login]; !ok {
		m[login] = &UserStats{Login: login}
	}
	return m[login]
}

func calculateWorkingTime(start, end time.Time) time.Duration {
	if start.After(end) {
		return 0
	}
	duration := time.Duration(0)
	current := start
	for current.Before(end) {
		if current.Weekday() != time.Saturday && current.Weekday() != time.Sunday {
			nextDay := time.Date(
				current.Year(), current.Month(), current.Day()+1, 0, 0, 0, 0, current.Location(),
			)
			chunkEnd := nextDay
			if chunkEnd.After(end) {
				chunkEnd = end
			}
			duration += chunkEnd.Sub(current)
			current = chunkEnd
		} else {
			current = time.Date(
				current.Year(), current.Month(), current.Day()+1, 0, 0, 0, 0, current.Location(),
			)
		}
	}
	return duration
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "None"
	}
	days, hours, minutes := int(d.Hours())/24, int(d.Hours())%24, int(d.Minutes())%60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return "<1m"
}

func getStats(durations []time.Duration) (string, string, string) {
	n := len(durations)
	if n == 0 {
		return "None", "None", "None"
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	total := time.Duration(0)
	for _, d := range durations {
		total += d
	}
	avg := time.Duration(int64(total) / int64(n))

	var median time.Duration
	if n%2 == 1 {
		median = durations[n/2]
	} else {
		median = (durations[n/2-1] + durations[n/2]) / 2
	}

	p90Idx := int(float64(n) * 0.9)
	if p90Idx >= n {
		p90Idx = n - 1
	}
	return formatDuration(avg), formatDuration(median), formatDuration(durations[p90Idx])
}

func getPRStats(
	ctx context.Context,
	githubClient clientgithub.Client,
	org string,
	opts PRStatsOptions,
) (string, error) {
	userStatsMap := make(map[string]*UserStats)
	processedPRs := []PRData{}
	allOpenPRs := []PRData{}
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	for _, repo := range opts.Repos {
		logrus.Infof("Processing repo: %s", repo)

		repoOpenPRs, err := fetchRepoOpenPRs(ctx, githubClient, org, repo, opts, userStatsMap, thirtyDaysAgo)
		if err != nil {
			return "", err
		}
		allOpenPRs = append(allOpenPRs, repoOpenPRs...)

		repoRecentlyClosed, err := fetchRepoClosedPRs(
			ctx, githubClient, org, repo, opts, userStatsMap, thirtyDaysAgo,
		)
		if err != nil {
			return "", err
		}

		for i := range repoOpenPRs {
			if repoOpenPRs[i].CreatedAt.After(thirtyDaysAgo) {
				fetchReviewsAndTTRv(
					ctx, githubClient, org, repo, &repoOpenPRs[i], userStatsMap, opts.ExcludedUsers,
				)
				processedPRs = append(processedPRs, repoOpenPRs[i])
			}
		}
		for i := range repoRecentlyClosed {
			fetchReviewsAndTTRv(
				ctx, githubClient, org, repo, &repoRecentlyClosed[i], userStatsMap, opts.ExcludedUsers,
			)
			processedPRs = append(processedPRs, repoRecentlyClosed[i])
		}
	}
	return generatePRStatsReport(opts, processedPRs, allOpenPRs, userStatsMap), nil
}

func fetchRepoOpenPRs(
	ctx context.Context,
	githubClient clientgithub.Client,
	org, repo string,
	opts PRStatsOptions,
	userStatsMap map[string]*UserStats,
	thirtyDaysAgo time.Time,
) ([]PRData, error) {
	allOpenPRs := []PRData{}
	openOpts := &github.PullRequestListOptions{
		State: "open",
		ListOptions: github.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}
	for {
		prs, err := githubClient.ListPullRequests(ctx, org, repo, openOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to list open pull requests for %s/%s: %w", org, repo, err)
		}
		if len(prs) == 0 {
			break
		}
		for _, pr := range prs {
			if (opts.ExcludeDrafts && pr.GetDraft()) || hasExcludedLabel(pr, opts.ExcludedLabels) {
				continue
			}
			author := pr.GetUser().GetLogin()
			involvedNow := make(map[string]bool)
			for _, a := range pr.Assignees {
				involvedNow[a.GetLogin()] = true
			}
			for _, r := range pr.RequestedReviewers {
				involvedNow[r.GetLogin()] = true
			}
			if !opts.ExcludedUsers[author] {
				ensureUser(userStatsMap, author).CurrentOpen++
				prData := PRData{
					Number:    pr.GetNumber(),
					Title:     pr.GetTitle(),
					URL:       pr.GetHTMLURL(),
					Author:    author,
					State:     "open",
					Draft:     pr.GetDraft(),
					CreatedAt: pr.GetCreatedAt(),
					Repo:      repo,
				}
				for l := range involvedNow {
					prData.Assignees = append(prData.Assignees, l)
				}
				sort.Strings(prData.Assignees)
				allOpenPRs = append(allOpenPRs, prData)
			}
			for login := range involvedNow {
				if !opts.ExcludedUsers[login] {
					ensureUser(userStatsMap, login).CurrentAssigned++
				}
			}
			if pr.GetCreatedAt().After(thirtyDaysAgo) && !opts.ExcludedUsers[author] {
				ensureUser(userStatsMap, author).Opened++
			}
		}
		if len(prs) < 100 {
			break
		}
		openOpts.Page++
	}
	return allOpenPRs, nil
}

func fetchRepoClosedPRs(
	ctx context.Context,
	githubClient clientgithub.Client,
	org, repo string,
	opts PRStatsOptions,
	userStatsMap map[string]*UserStats,
	thirtyDaysAgo time.Time,
) ([]PRData, error) {
	recentlyClosed := []PRData{}
	closedOpts := &github.PullRequestListOptions{
		State:     "closed",
		Sort:      "updated",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}
	for {
		prs, err := githubClient.ListPullRequests(ctx, org, repo, closedOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to list closed pull requests for %s/%s: %w", org, repo, err)
		}
		if len(prs) == 0 {
			break
		}
		for _, pr := range prs {
			if pr.GetClosedAt().Before(thirtyDaysAgo) {
				return recentlyClosed, nil
			}
			if (opts.ExcludeDrafts && pr.GetDraft()) || hasExcludedLabel(pr, opts.ExcludedLabels) {
				continue
			}
			author := pr.GetUser().GetLogin()
			if opts.ExcludedUsers[author] {
				continue
			}
			u := ensureUser(userStatsMap, author)
			if pr.GetCreatedAt().After(thirtyDaysAgo) {
				u.Opened++
			}
			u.Closed++
			ttc := calculateWorkingTime(pr.GetCreatedAt(), pr.GetClosedAt())
			recentlyClosed = append(recentlyClosed, PRData{
				Number:      pr.GetNumber(),
				Title:       pr.GetTitle(),
				URL:         pr.GetHTMLURL(),
				Author:      author,
				State:       "closed",
				Draft:       pr.GetDraft(),
				CreatedAt:   pr.GetCreatedAt(),
				TimeToClose: ttc,
				Repo:        repo,
			})
			u.CloseTimes = append(u.CloseTimes, ttc)
		}
		if len(prs) < 100 {
			break
		}
		closedOpts.Page++
	}
	return recentlyClosed, nil
}

func hasExcludedLabel(pr *github.PullRequest, excluded map[string]bool) bool {
	for _, l := range pr.Labels {
		if excluded[l.GetName()] {
			return true
		}
	}
	return false
}

func fetchReviewsAndTTRv(
	ctx context.Context,
	client clientgithub.Client,
	org, repo string,
	pr *PRData,
	stats map[string]*UserStats,
	excluded map[string]bool,
) {
	reviews, _ := client.ListReviews(ctx, org, repo, pr.Number, nil)
	if len(reviews) == 0 {
		return
	}
	timeline, _ := client.ListTimeline(ctx, org, repo, pr.Number, nil)
	requestTimes := make(map[string]time.Time)
	for _, e := range timeline {
		if e.GetEvent() == "review_requested" {
			login := ""
			if e.Actor != nil {
				login = e.Actor.GetLogin()
			}
			if login != "" {
				if _, ok := requestTimes[login]; !ok {
					requestTimes[login] = e.GetCreatedAt()
				}
			}
		}
	}
	sort.Slice(reviews, func(i, j int) bool {
		return reviews[i].GetSubmittedAt().Before(reviews[j].GetSubmittedAt())
	})
	pr.TimeToFirstReview = calculateWorkingTime(pr.CreatedAt, reviews[0].GetSubmittedAt())
	proc := make(map[string]bool)
	for _, r := range reviews {
		login := r.GetUser().GetLogin()
		if login == pr.Author || excluded[login] || proc[login] {
			continue
		}
		u := ensureUser(stats, login)
		u.Reviewed++
		proc[login] = true
		reqTime, ok := requestTimes[login]
		if !ok {
			reqTime = pr.CreatedAt
		}
		u.ReviewTimes = append(u.ReviewTimes, calculateWorkingTime(reqTime, r.GetSubmittedAt()))
	}
}

func generatePRStatsReport(
	opts PRStatsOptions,
	processedPRs []PRData,
	allOpenPRs []PRData,
	userStatsMap map[string]*UserStats,
) string {
	var report strings.Builder
	repoLabel := opts.RepoLabel
	if repoLabel == "" {
		repoLabel = strings.Join(opts.Repos, ", ")
		if len(opts.Repos) > 3 {
			repoLabel = fmt.Sprintf("%d repositories", len(opts.Repos))
		}
	}
	report.WriteString(fmt.Sprintf("# PR Metrics for `%s` (Last 30 Days)\n", repoLabel))

	if opts.Mode == "full" {
		writeReportSummary(&report, processedPRs, allOpenPRs)
	}

	writeReportTeamActivity(&report, userStatsMap)

	if opts.Mode == "full" {
		writeReportAttention(&report, allOpenPRs, opts.SLAHours)
		writeReportFullDetails(&report, processedPRs)
	}

	report.WriteString(fmt.Sprintf(
		"\n---\n_Report generated on %s_",
		time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
	))
	return report.String()
}

func writeReportSummary(report *strings.Builder, processedPRs []PRData, allOpenPRs []PRData) {
	report.WriteString("\n### Metrics Summary (Last 30 Days)\n")
	report.WriteString("| Metric | Average | Median | 90th percentile |\n")
	report.WriteString("|---|---|---|---|\n")

	ttrList, ttcList := []time.Duration{}, []time.Duration{}
	closedCount := 0
	for _, pr := range processedPRs {
		if pr.TimeToFirstReview > 0 {
			ttrList = append(ttrList, pr.TimeToFirstReview)
		}
		if pr.TimeToClose > 0 {
			ttcList = append(ttcList, pr.TimeToClose)
		}
		if pr.State == "closed" {
			closedCount++
		}
	}
	ttrAvg, ttrMed, ttrP90 := getStats(ttrList)
	ttcAvg, ttcMed, ttcP90 := getStats(ttcList)

	report.WriteString(fmt.Sprintf(
		"| **Time to first response** | %s | %s | %s |\n", ttrAvg, ttrMed, ttrP90,
	))
	report.WriteString(fmt.Sprintf(
		"| **Time to close** | %s | %s | %s |\n", ttcAvg, ttcMed, ttcP90,
	))

	report.WriteString("\n### Activity Counts\n| Metric | Count |\n|---|---|\n")
	report.WriteString(fmt.Sprintf("| PRs currently open (filtered) | **%d** |\n", len(allOpenPRs)))
	report.WriteString(fmt.Sprintf("| PRs closed (Last 30d) | **%d** |\n", closedCount))
	report.WriteString(fmt.Sprintf("| PRs created (Last 30d) | **%d** |\n", len(processedPRs)))
}

func writeReportTeamActivity(report *strings.Builder, userStatsMap map[string]*UserStats) {
	report.WriteString("\n### Team Activity\n")
	report.WriteString("| User | Opened (30d) | Closed (30d) | Reviews (30d) | ")
	report.WriteString("Median TTC | Median TTRv | **Open Now** | **Assigned/Reviewing** |\n")
	report.WriteString("|---|---|---|---|---|---|---|---|\n")

	users := make([]*UserStats, 0, len(userStatsMap))
	for _, s := range userStatsMap {
		users = append(users, s)
	}
	sort.Slice(users, func(i, j int) bool {
		activityI := users[i].Opened + users[i].Reviewed + users[i].CurrentAssigned
		activityJ := users[j].Opened + users[j].Reviewed + users[j].CurrentAssigned
		return activityI > activityJ
	})

	for _, s := range users {
		_, ttcMed, _ := getStats(s.CloseTimes)
		_, ttrvMed, _ := getStats(s.ReviewTimes)
		report.WriteString(fmt.Sprintf(
			"| %s | %d | %d | %d | %s | %s | **%d** | **%d** |\n",
			s.Login, s.Opened, s.Closed, s.Reviewed, ttcMed, ttrvMed,
			s.CurrentOpen, s.CurrentAssigned,
		))
	}
}

func writeReportAttention(report *strings.Builder, allOpenPRs []PRData, slaHours int) {
	sla, now := time.Duration(slaHours)*time.Hour, time.Now()
	slowPRs := []PRData{}
	for _, pr := range allOpenPRs {
		if calculateWorkingTime(pr.CreatedAt, now) > sla {
			slowPRs = append(slowPRs, pr)
		}
	}
	report.WriteString(fmt.Sprintf("\n### PRs Needing Attention (>%d business hours)\n", slaHours))
	if len(slowPRs) > 0 {
		report.WriteString("| PR | Author | Issue |\n|---|---|---|\n")
		for _, pr := range slowPRs {
			report.WriteString(fmt.Sprintf(
				"| [#%d (%s)](%s) | %s | Open for %s |\n",
				pr.Number, pr.Repo, pr.URL, pr.Author,
				formatDuration(calculateWorkingTime(pr.CreatedAt, now)),
			))
		}
	} else {
		report.WriteString("_None!_\n")
	}
}

func writeReportFullDetails(report *strings.Builder, processedPRs []PRData) {
	if len(processedPRs) == 0 {
		return
	}
	report.WriteString("\n<details>\n<summary><b>View All Processed PRs (30d)</b></summary>\n\n")
	report.WriteString("| Title | PR | Author | Involved | Review Time | Close Time | Status |\n")
	report.WriteString("|---|---|---|---|---|---|---|\n")
	for _, pr := range processedPRs {
		title := pr.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}
		involved := strings.Join(pr.Assignees, ", ")
		if involved == "" {
			involved = "None"
		}
		status := pr.State
		if pr.Draft {
			status += " (draft)"
		}
		report.WriteString(fmt.Sprintf(
			"| %s | [#%d (%s)](%s) | %s | %s | %s | %s | %s |\n",
			title, pr.Number, pr.Repo, pr.URL, pr.Author, involved,
			formatDuration(pr.TimeToFirstReview), formatDuration(pr.TimeToClose), status,
		))
	}
	report.WriteString("\n</details>\n")
}
