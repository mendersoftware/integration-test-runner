package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

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

const (
	prStatsModeFull = "full"
	prStatsModeTeam = "team"
)

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
	isDefaultPath := false
	if path == "" {
		path = os.Getenv("PR_STATS_CONFIG_PATH")
		if path == "" {
			path = "pr_stats_config.json"
			isDefaultPath = true
			if _, err := os.Stat(path); os.IsNotExist(err) {
				path = "/pr_stats_config.json"
			}
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if isDefaultPath && os.IsNotExist(err) {
			logrus.Infof("Optional config file %s not found, using defaults", path)
			return nil, nil
		}
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
				// We make sure that the current repository is always included
				// regardless of the config and the mode selected.
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
	if !start.Before(end) {
		return 0
	}

	// Normalize to start-of-day boundaries to count whole weekdays,
	// then add back the partial day contributions.
	startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())

	totalDays := int(endDay.Sub(startDay).Hours() / 24)
	completeWeeks := totalDays / 7
	remainderDays := totalDays % 7

	// Count weekend days in the remainder
	weekendDays := 0
	for i := 0; i < remainderDays; i++ {
		d := startDay.AddDate(0, 0, completeWeeks*7+i)
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			weekendDays++
		}
	}

	businessDays := totalDays - completeWeeks*2 - weekendDays

	// Full business days worth of time
	duration := time.Duration(businessDays) * 24 * time.Hour

	// Add partial time from the start day (midnight to end-of-day or end time)
	isStartWeekday := start.Weekday() != time.Saturday && start.Weekday() != time.Sunday
	isEndWeekday := end.Weekday() != time.Saturday && end.Weekday() != time.Sunday

	if start.Day() == end.Day() && start.Month() == end.Month() && start.Year() == end.Year() {
		// Same day: just the difference if it's a weekday
		if isStartWeekday {
			return end.Sub(start)
		}
		return 0
	}

	// Subtract the full start day (we counted it as 24h) and add only the working portion
	if isStartWeekday {
		startOfNextDay := startDay.AddDate(0, 0, 1)
		duration -= 24 * time.Hour
		duration += startOfNextDay.Sub(start)
	}

	// Add the partial end day
	if isEndWeekday {
		duration += end.Sub(endDay) // add partial: midnight -> end
	}

	if duration < 0 {
		return 0
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
	sorted := make([]time.Duration, n)
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	total := time.Duration(0)
	for _, d := range sorted {
		total += d
	}
	avg := time.Duration(int64(total) / int64(n))

	var median time.Duration
	if n%2 == 1 {
		median = sorted[n/2]
	} else {
		median = (sorted[n/2-1] + sorted[n/2]) / 2
	}

	p90Idx := int(float64(n) * 0.9)
	if p90Idx >= n {
		p90Idx = n - 1
	}
	return formatDuration(avg), formatDuration(median), formatDuration(sorted[p90Idx])
}

type repoResult struct {
	openPRs      []PRData
	processedPRs []PRData
	userStats    map[string]*UserStats
}

func mergeUserStats(dst, src map[string]*UserStats) {
	for login, s := range src {
		d := ensureUser(dst, login)
		d.Opened += s.Opened
		d.Reviewed += s.Reviewed
		d.Closed += s.Closed
		d.CurrentOpen += s.CurrentOpen
		d.CurrentAssigned += s.CurrentAssigned
		d.ReviewTimes = append(d.ReviewTimes, s.ReviewTimes...)
		d.CloseTimes = append(d.CloseTimes, s.CloseTimes...)
	}
}

func getPRStats(
	ctx context.Context,
	githubClient clientgithub.Client,
	org string,
	opts PRStatsOptions,
) (string, error) {
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	needReviews := true

	results := make([]repoResult, len(opts.Repos))

	g, gctx := errgroup.WithContext(ctx)
	for i, repo := range opts.Repos {
		g.Go(func() error {
			logrus.Infof("Processing repo: %s", repo)

			openStats := make(map[string]*UserStats)
			closedStats := make(map[string]*UserStats)
			var repoOpenPRs, repoClosedPRs []PRData

			// Fetch open and closed PRs in parallel
			fg, fctx := errgroup.WithContext(gctx)
			fg.Go(func() error {
				var err error
				repoOpenPRs, err = fetchRepoOpenPRs(
					fctx, githubClient, org, repo, opts, openStats, thirtyDaysAgo,
				)
				return err
			})
			fg.Go(func() error {
				var err error
				repoClosedPRs, err = fetchRepoClosedPRs(
					fctx, githubClient, org, repo, opts, closedStats, thirtyDaysAgo,
				)
				return err
			})
			if err := fg.Wait(); err != nil {
				return err
			}

			// Merge open and closed stats into a single repo map
			repoStats := make(map[string]*UserStats)
			mergeUserStats(repoStats, openStats)
			mergeUserStats(repoStats, closedStats)

			// Collect PRs that need processing
			var toReview []*PRData
			for j := range repoOpenPRs {
				if repoOpenPRs[j].CreatedAt.After(thirtyDaysAgo) {
					toReview = append(toReview, &repoOpenPRs[j])
				}
			}
			for j := range repoClosedPRs {
				toReview = append(toReview, &repoClosedPRs[j])
			}

			// Fetch reviews concurrently with bounded parallelism
			if needReviews && len(toReview) > 0 {
				var mu sync.Mutex
				rg, rgctx := errgroup.WithContext(gctx)
				rg.SetLimit(10)
				for _, pr := range toReview {
					rg.Go(func() error {
						localStats := make(map[string]*UserStats)
						fetchReviewsAndTTRv(
							rgctx, githubClient, org, repo,
							pr, localStats, opts.ExcludedUsers,
						)
						mu.Lock()
						mergeUserStats(repoStats, localStats)
						mu.Unlock()
						return nil
					})
				}
				_ = rg.Wait()
			}

			// Build processed slice from reviewed PRs
			processed := make([]PRData, 0, len(toReview))
			for _, pr := range toReview {
				processed = append(processed, *pr)
			}

			results[i] = repoResult{
				openPRs:      repoOpenPRs,
				processedPRs: processed,
				userStats:    repoStats,
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return "", err
	}

	// Merge all per-repo results
	userStatsMap := make(map[string]*UserStats)
	var processedPRs []PRData
	var allOpenPRs []PRData
	for _, r := range results {
		allOpenPRs = append(allOpenPRs, r.openPRs...)
		processedPRs = append(processedPRs, r.processedPRs...)
		mergeUserStats(userStatsMap, r.userStats)
	}

	return generatePRStatsReport(opts, processedPRs, allOpenPRs, userStatsMap, thirtyDaysAgo), nil
}

func fetchRepoOpenPRs(
	ctx context.Context,
	githubClient clientgithub.Client,
	org, repo string,
	opts PRStatsOptions,
	userStatsMap map[string]*UserStats,
	thirtyDaysAgo time.Time,
) ([]PRData, error) {
	var allOpenPRs []PRData
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
			return nil, fmt.Errorf(
				"failed to list open pull requests for %s/%s: %w", org, repo, err,
			)
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
	var recentlyClosed []PRData
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
			return nil, fmt.Errorf(
				"failed to list closed pull requests for %s/%s: %w", org, repo, err,
			)
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
	var allReviews []*github.PullRequestReview
	reviewOpts := &github.ListOptions{PerPage: 100, Page: 1}
	for {
		reviews, err := client.ListReviews(ctx, org, repo, pr.Number, reviewOpts)
		if err != nil {
			logrus.Warnf("Failed to list reviews for %s/%s#%d: %s", org, repo, pr.Number, err)
			return
		}
		allReviews = append(allReviews, reviews...)
		if len(reviews) < 100 {
			break
		}
		reviewOpts.Page++
	}
	if len(allReviews) == 0 {
		return
	}

	sort.Slice(allReviews, func(i, j int) bool {
		return allReviews[i].GetSubmittedAt().Before(allReviews[j].GetSubmittedAt())
	})

	// NOTE: go-github v28's Timeline struct does not expose RequestedReviewer,
	// so we use pr.CreatedAt as the baseline for per-reviewer review times.
	proc := make(map[string]bool)
	for _, r := range allReviews {
		login := r.GetUser().GetLogin()
		if login == pr.Author || excluded[login] {
			continue
		}
		// Use the first qualifying review for TimeToFirstReview
		if pr.TimeToFirstReview == 0 {
			pr.TimeToFirstReview = calculateWorkingTime(
				pr.CreatedAt, r.GetSubmittedAt(),
			)
		}
		if proc[login] {
			continue
		}
		u := ensureUser(stats, login)
		u.Reviewed++
		proc[login] = true
		u.ReviewTimes = append(
			u.ReviewTimes, calculateWorkingTime(pr.CreatedAt, r.GetSubmittedAt()),
		)
	}
}

func generatePRStatsReport(
	opts PRStatsOptions,
	processedPRs []PRData,
	allOpenPRs []PRData,
	userStatsMap map[string]*UserStats,
	thirtyDaysAgo time.Time,
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

	if opts.Mode == prStatsModeFull {
		writeReportSummary(&report, processedPRs, allOpenPRs, thirtyDaysAgo)
	}

	writeReportTeamActivity(&report, userStatsMap, opts.Mode)

	if opts.Mode == prStatsModeFull {
		writeReportAttention(&report, allOpenPRs, opts.SLAHours)
		writeReportFullDetails(&report, processedPRs)
	}

	report.WriteString(fmt.Sprintf(
		"\n---\n_Report generated on %s_",
		time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
	))
	return report.String()
}

func writeReportSummary(
	report *strings.Builder,
	processedPRs []PRData,
	allOpenPRs []PRData,
	thirtyDaysAgo time.Time,
) {
	report.WriteString("\n### Metrics Summary (Last 30 Days)\n")
	report.WriteString("| Metric | Average | Median | 90th percentile |\n")
	report.WriteString("|---|---|---|---|\n")

	var ttrList, ttcList []time.Duration
	closedCount := 0
	createdCount := 0
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
		if pr.CreatedAt.After(thirtyDaysAgo) {
			createdCount++
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
	report.WriteString(fmt.Sprintf("| PRs created (Last 30d) | **%d** |\n", createdCount))
}

func writeReportTeamActivity(
	report *strings.Builder, userStatsMap map[string]*UserStats, mode string,
) {
	report.WriteString("\n### Team Activity\n")
	if mode == prStatsModeTeam {
		report.WriteString("| User | Opened (30d) | Closed (30d) | Reviews (30d) | ")
		report.WriteString("Median TTC | **Open Now** | **Assigned/Reviewing** |\n")
		report.WriteString("|---|---|---|---|---|---|---|\n")
	} else {
		report.WriteString("| User | Opened (30d) | Closed (30d) | Reviews (30d) | ")
		report.WriteString("Median TTC | Median TTRv | **Open Now** | **Assigned/Reviewing** |\n")
		report.WriteString("|---|---|---|---|---|---|---|---|\n")
	}

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
		if mode == prStatsModeTeam {
			report.WriteString(fmt.Sprintf(
				"| %s | %d | %d | %d | %s | **%d** | **%d** |\n",
				s.Login, s.Opened, s.Closed, s.Reviewed, ttcMed,
				s.CurrentOpen, s.CurrentAssigned,
			))
		} else {
			_, ttrvMed, _ := getStats(s.ReviewTimes)
			report.WriteString(fmt.Sprintf(
				"| %s | %d | %d | %d | %s | %s | **%d** | **%d** |\n",
				s.Login, s.Opened, s.Closed, s.Reviewed, ttcMed, ttrvMed,
				s.CurrentOpen, s.CurrentAssigned,
			))
		}
	}
}

type slowPR struct {
	PRData
	Age time.Duration
}

func writeReportAttention(report *strings.Builder, allOpenPRs []PRData, slaHours int) {
	sla, now := time.Duration(slaHours)*time.Hour, time.Now()
	var slowPRs []slowPR
	for _, pr := range allOpenPRs {
		age := calculateWorkingTime(pr.CreatedAt, now)
		if age > sla {
			slowPRs = append(slowPRs, slowPR{PRData: pr, Age: age})
		}
	}
	report.WriteString(fmt.Sprintf("\n### PRs Needing Attention (>%d business hours)\n", slaHours))
	if len(slowPRs) > 0 {
		report.WriteString("| PR | Author | Issue |\n|---|---|---|\n")
		for _, pr := range slowPRs {
			report.WriteString(fmt.Sprintf(
				"| [#%d (%s)](%s) | %s | Open for %s |\n",
				pr.Number, pr.Repo, pr.URL, pr.Author,
				formatDuration(pr.Age),
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
		titleRunes := []rune(title)
		if len(titleRunes) > 40 {
			title = string(titleRunes[:37]) + "..."
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
