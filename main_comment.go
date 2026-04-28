package main

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	clientgithub "github.com/mendersoftware/integration-test-runner/client/github"
	clientgitlab "github.com/mendersoftware/integration-test-runner/client/gitlab"
)

// nolint: gocyclo
func processGitHubComment(
	ctx *gin.Context,
	comment *github.IssueCommentEvent,
	githubClient clientgithub.Client,
	conf *config,
) error {
	log := getCustomLoggerFromContext(ctx)

	// process created actions only, ignore the others
	action := comment.GetAction()
	if action != "created" {
		log.Infof("Ignoring action %s on comment", action)
		return nil
	}

	// accept commands only from organization members
	if !githubClient.IsOrganizationMember(ctx, conf.githubOrganization, comment.Sender.GetLogin()) {
		log.Warnf(
			"%s commented, but he/she is not a member of our organization, ignoring",
			comment.Sender.GetLogin(),
		)
		return nil
	}

	// but ignore comments from myself
	if comment.Sender.GetLogin() == githubBotName {
		log.Warnf("%s commented, probably giving instructions, ignoring", comment.Sender.GetLogin())
		return nil
	}

	// filter comments mentioning the bot
	commentBody := comment.Comment.GetBody()
	if !strings.Contains(commentBody, "@"+githubBotName) {
		log.Info("ignoring comment not mentioning me")
		return nil
	}

	// retrieve the pull request
	prLink := comment.Issue.GetPullRequestLinks().GetURL()
	if prLink == "" {
		log.Warnf("ignoring comment not on a pull request")
		return nil
	}

	prLinkParts := strings.Split(prLink, "/")
	prNumber, err := strconv.Atoi(prLinkParts[len(prLinkParts)-1])
	if err != nil {
		log.Errorf("Unable to retrieve the pull request: %s", err.Error())
		return err
	}

	pr, err := githubClient.GetPullRequest(
		ctx,
		conf.githubOrganization,
		comment.GetRepo().GetName(),
		prNumber,
	)
	if err != nil {
		log.Errorf("Unable to retrieve the pull request: %s", err.Error())
		return err
	}

	// extract the command and check it is valid
	switch {
	case strings.Contains(commentBody, commandStartIntegrationPipeline):
		prRequest := &github.PullRequestEvent{
			Repo:        comment.GetRepo(),
			Number:      github.Int(pr.GetNumber()),
			PullRequest: pr,
		}
		build := getIntegrationBuild(log, conf, prRequest)

		_, err = syncProtectedBranch(log, prRequest, conf, integrationPipelinePath)
		if err != nil {
			_ = say(ctx, "There was an error while syncing branches: {{.ErrorMessage}}",
				struct {
					ErrorMessage string
				}{
					ErrorMessage: err.Error(),
				},
				log,
				conf,
				prRequest)
			return err

		}

		// start the build
		if err := triggerIntegrationBuild(log, conf, &build, prRequest, nil); err != nil {
			log.Errorf("Could not start build: %s", err.Error())
		}
	case strings.Contains(commentBody, commandStartClientPipeline):
		buildOptions, err := parseBuildOptions(commentBody)
		// get the list of builds
		prRequest := &github.PullRequestEvent{
			Repo:        comment.GetRepo(),
			Number:      github.Int(pr.GetNumber()),
			PullRequest: pr,
		}
		if err != nil {
			_ = say(ctx, "There was an error while parsing arguments: {{.ErrorMessage}}",
				struct {
					ErrorMessage string
				}{
					ErrorMessage: err.Error(),
				},
				log,
				conf,
				prRequest)
			return err
		}
		builds := parseClientPullRequest(log, conf, "opened", prRequest)
		log.Infof(
			"%s:%d will trigger %d builds",
			comment.GetRepo().GetName(),
			pr.GetNumber(),
			len(builds),
		)

		// start the builds
		for idx, build := range builds {
			log.Infof("%d: "+spew.Sdump(build)+"\n", idx+1)
			if build.repo == "meta-mender" && build.baseBranch == "master-next" {
				log.Info("Skipping build targeting meta-mender:master-next")
				continue
			}
			if len(buildOptions.Releases) > 0 && !slices.Contains(
				buildOptions.Releases,
				build.baseBranch,
			) {
				log.Infof("Skipping build for %s (not in --release)", build.baseBranch)
				continue
			}
			if err := triggerClientBuild(log, conf, &build, prRequest, buildOptions); err != nil {
				log.Errorf("Could not start build: %s", err.Error())
			}
		}
	case strings.Contains(commentBody, commandCherryPickBranch):
		log.Infof("Attempting to cherry-pick the changes in PR: %s/%d",
			comment.GetRepo().GetName(),
			pr.GetNumber(),
		)
		err = cherryPickPR(log, comment, pr, conf, commentBody, githubClient)
		if err != nil {
			log.Error(err)
		}
	case strings.Contains(commentBody, commandConventionalCommit) &&
		strings.Contains(pr.GetUser().GetLogin(), "dependabot"):
		log.Infof(
			"Attempting to make the PR: %s/%d and commit: %s a conventional commit",
			comment.GetRepo().GetName(),
			pr.GetNumber(),
			pr.GetHead().GetSHA(),
		)
		err = conventionalComittifyDependabotPr(log, comment, pr, conf, commentBody, githubClient)
		if err != nil {
			log.Error(err)
		}
	case strings.Contains(commentBody, commandSyncRepos):
		syncPRBranch(ctx, comment, pr, log, conf)
	case strings.Contains(commentBody, commandPrintFullPRStats) ||
		strings.Contains(commentBody, commandPrintPRStats):
		handlePRStatsCommand(ctx, comment, pr, githubClient, conf, log, commentBody)
	default:
		log.Warnf("no command found: %s", commentBody)
		return nil
	}

	return nil
}

func protectBranch(conf *config, branchName string, pipelinePath string) error {
	// https://docs.gitlab.com/ee/api/protected_branches.html#protect-repository-branches
	allow_force_push := false
	opt := &gitlab.ProtectRepositoryBranchesOptions{
		Name:           &branchName,
		AllowForcePush: &allow_force_push,
	}

	client, err := clientgitlab.NewGitLabClient(
		conf.gitlabToken,
		conf.gitlabBaseURL,
		conf.dryRunMode,
	)

	if err != nil {
		return err
	}

	_, err = client.ProtectRepositoryBranches(pipelinePath, opt)
	if err != nil {
		return fmt.Errorf("%v returned error: %s", err, err.Error())
	}
	return nil
}

func unprotectBranch(conf *config, branchName string, pipelinePath string) error {
	client, err := clientgitlab.NewGitLabClient(
		conf.gitlabToken,
		conf.gitlabBaseURL,
		conf.dryRunMode,
	)
	if err != nil {
		return err
	}

	response, err := client.UnprotectRepositoryBranches(pipelinePath, branchName, nil)
	if err != nil {
		// Ignore 404 — branch may not be protected (yet)
		if response != nil && response.StatusCode == 404 {
			return nil
		}
		return fmt.Errorf("failed to unprotect branch %s: %s", branchName, err.Error())
	}
	return nil
}

const (
	// maxSyncRetries is the number of times to retry the unprotect+push cycle
	// when force push fails due to branch protection not yet being removed.
	maxSyncRetries = 3
	// syncRetryDelay is the delay between retries to allow GitLab to propagate
	// the branch unprotection.
	syncRetryDelay = 3 * time.Second
)

func isProtectedBranchError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "protected branch")
}

func syncProtectedBranch(
	log *logrus.Entry,
	pr *github.PullRequestEvent,
	conf *config,
	pipelinePath string,
) (string, error) {
	prBranchName := "pr_" + strconv.Itoa(pr.GetNumber()) + "_protected"

	var lastErr error
	for attempt := 1; attempt <= maxSyncRetries; attempt++ {
		// Unprotect the branch so that the force push is allowed.
		// This is idempotent — if the branch is not protected, 404 is ignored.
		if err := unprotectBranch(conf, prBranchName, pipelinePath); err != nil {
			return "", fmt.Errorf("failed to unprotect branch before sync: %s", err.Error())
		}

		lastErr = syncBranch(prBranchName, log, pr, conf)
		if lastErr == nil {
			break
		}

		if !isProtectedBranchError(lastErr) {
			mainErrMsg := "There was an error syncing branches"
			return "", fmt.Errorf("%v returned error: %s: %s", lastErr, mainErrMsg, lastErr.Error())
		}

		if attempt < maxSyncRetries {
			log.Warnf(
				"Force push to %s failed due to branch protection (attempt %d/%d), retrying in %s",
				prBranchName, attempt, maxSyncRetries, syncRetryDelay,
			)
			time.Sleep(syncRetryDelay)
		}
	}

	if lastErr != nil {
		mainErrMsg := "There was an error syncing branches"
		return "", fmt.Errorf("%v returned error: %s: %s", lastErr, mainErrMsg, lastErr.Error())
	}

	if err := protectBranch(conf, prBranchName, pipelinePath); err != nil {
		return "", fmt.Errorf("%v returned error: %s", err, err.Error())
	}
	return prBranchName, nil
}

func syncPRBranch(
	ctx *gin.Context,
	comment *github.IssueCommentEvent,
	pr *github.PullRequest,
	log *logrus.Entry,
	conf *config,
) {
	prEvent := &github.PullRequestEvent{
		Repo:        comment.GetRepo(),
		Number:      github.Int(pr.GetNumber()),
		PullRequest: pr,
	}
	if _, err := syncPullRequestBranch(log, prEvent, conf); err != nil {
		mainErrMsg := "There was an error syncing branches"
		log.Errorf(mainErrMsg+": %s", err.Error())
		msg := mainErrMsg + ", " + msgDetailsKubernetesLog
		postGitHubMessage(ctx, prEvent, log, msg)
	}
}

func handlePRStatsCommand(
	ctx *gin.Context,
	comment *github.IssueCommentEvent,
	pr *github.PullRequest,
	githubClient clientgithub.Client,
	conf *config,
	log *logrus.Entry,
	commentBody string,
) {
	opts := parsePRStatsOptions(commentBody, comment.GetRepo().GetName(), log)

	report, err := getPRStats(ctx, githubClient, conf.githubOrganization, opts)
	if err != nil {
		report = "Failed to generate PR stats: " + err.Error()
	}

	err = githubClient.CreateComment(
		ctx,
		conf.githubOrganization,
		comment.GetRepo().GetName(),
		pr.GetNumber(),
		&github.IssueComment{
			Body: github.String(report),
		},
	)
	if err != nil {
		log.Errorf(
			"Failed to comment on the pr: %v, Error: %s",
			pr, err.Error(),
		)
	}
}

func parsePRStatsOptions(
	commentBody, defaultRepo string, log *logrus.Entry,
) PRStatsOptions {
	// isFull is the single source of truth:
	//   print fast pr stats -> team mode, fast (only fast team repos)
	//   print full pr stats -> full mode, slow (all team repos)
	// --repo / --team opt out of auto team detection and use an explicit selection.
	isFull := strings.Contains(commentBody, commandPrintFullPRStats)

	repos := []string{defaultRepo}
	statsConfig, err := loadPRStatsConfig("")
	if err != nil {
		log.Errorf("failed to load pr stats config: %s", err.Error())
	}

	opts := defaultStatsOptions(statsConfig)
	repoLabel := ""
	repoOverridden := false

	words := strings.Fields(commentBody)
	for i, word := range words {
		switch word {
		case "--repo":
			if i+1 < len(words) {
				repos = []string{words[i+1]}
				repoOverridden = true
			}
		case "--all-repos":
			// no-op: team expansion is now the default
		case "--exclude-drafts":
			opts.ExcludeDrafts = true
		case "--exclude-user":
			if i+1 < len(words) {
				opts.ExcludedUsers[words[i+1]] = true
			}
		case "--team":
			if r, l, ok := applyTeamFlag(words, i, statsConfig, isFull); ok {
				repos, repoLabel = r, l
				repoOverridden = true
			}
		}
	}

	if !repoOverridden && statsConfig != nil {
		repos, repoLabel = getTeamRepos(repos[0], statsConfig, isFull)
	}

	opts.Repos = repos
	opts.RepoLabel = repoLabel
	if isFull {
		opts.Mode = prStatsModeFull
	} else {
		opts.Mode = prStatsModeTeam
	}
	return opts
}

func defaultStatsOptions(cfg *PRStatsConfig) PRStatsOptions {
	opts := PRStatsOptions{
		SLAHours:       48,
		ExcludedUsers:  make(map[string]bool),
		ExcludedLabels: make(map[string]bool),
	}
	if cfg != nil {
		for _, u := range cfg.Global.ExcludedUsers {
			opts.ExcludedUsers[u] = true
		}
		for _, l := range cfg.Global.ExcludedLabels {
			opts.ExcludedLabels[l] = true
		}
		opts.SLAHours = cfg.Global.SLAHours
		opts.ExcludeDrafts = cfg.Global.ExcludeDrafts
	}
	return opts
}

func applyTeamFlag(
	words []string, i int, cfg *PRStatsConfig, slow bool,
) (repos []string, label string, ok bool) {
	if i+1 >= len(words) || cfg == nil {
		return nil, "", false
	}
	target := words[i+1]
	for _, t := range cfg.Teams {
		if strings.EqualFold(t.Name, target) {
			repos = t.Repositories
			label = t.Name + " Team"
			if !slow {
				repos = t.FastRepositories
				label = t.Name + " Team (Fast Mode)"
			}
			return repos, label, true
		}
	}
	return nil, "", false
}

// parsing `start client pipeline --pr mender-connect/pull/88/head --pr deviceconnect/pull/12/head
// --pr mender/3.1.x --fast sugar pretty please`
//
//	BuildOptions {
//		Fast: true,
//		PullRequests: map[string]string{
//			"mender-connect": "pull/88/head",
//			"deviceconnect": "pull/12/head",
//		}
//	}
func parseBuildOptions(commentBody string) (*BuildOptions, error) {
	buildOptions := NewBuildOptions()
	var err error
	words := strings.Fields(commentBody)
	tokensCount := len(words)
	for id, word := range words {
		if word == "--pr" && id < (tokensCount-1) {
			userInput := strings.TrimSpace(words[id+1])
			userInputParts := strings.Split(userInput, "/")

			if len(userInput) > 0 {
				var revision string
				switch len(userInputParts) {
				case 2: // we can have both deviceauth/1 and mender/3.1.x syntax
					// repo/<pr_number> syntax
					if _, err := strconv.Atoi(userInputParts[1]); err == nil {
						revision = "pull/" + userInputParts[1] + "/merge"
					} else {
						// feature branch
						revision = userInputParts[1]
					}
				case 3: // deviceconnect/pull/12 syntax
					revision = strings.Join(userInputParts[1:], "/") + "/merge"
				case 4: // deviceauth/pull/1/head syntax
					revision = strings.Join(userInputParts[1:], "/")
				default:
					err = errors.New(
						"parse error near '" + userInput + "', I need, e.g.: start client" +
							" pipeline --pr somerepo/pull/12/head --pr somerepo/1.0.x ",
					)
				}
				buildOptions.PullRequests[userInputParts[0]] = revision
			}
		} else if word == "--fast" {
			buildOptions.Fast = true
		} else if word == "--release" && id < (tokensCount-1) {
			buildOptions.Releases = append(buildOptions.Releases, strings.TrimSpace(words[id+1]))
		}
	}

	return buildOptions, err
}
