package main

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v28/github"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	clientgithub "github.com/mendersoftware/integration-test-runner/client/github"
)

var (
	changelogPrefix = "Merging these commits will result in the following changelog entries:\n\n"
	warningHeader   = "\n\n## Warning\n\nGenerating changelogs also resulted in these warnings:\n\n"

	msgDetailsKubernetesLog = "see <a href=\"https://console.cloud.google.com/kubernetes/" +
		"deployment/us-east1/company-websites/default/test-runner-mender-io/logs?" +
		"project=gp-kubernetes-269000\">logs</a> for details."
)

type retryParams struct {
	retryFunc func() error
	compFunc  func(error) bool
}

const (
	doRetry bool = true
	noRetry      = false
)

func retryOnError(args retryParams) error {
	var maxBackoff int = 8 * 8
	err := args.retryFunc()
	i := 1
	for i <= maxBackoff && args.compFunc(err) {
		err = args.retryFunc()
		i = i * 2
		time.Sleep(time.Duration(i) * time.Second)
	}
	return err
}

type TitleOptions struct {
	SkipCI bool
}

const (
	titleOptionSkipCI = "noci"
)

func getTitleOptions(title string) (titleOptions TitleOptions) {
	start, end := strings.Index(title, "["), strings.Index(title, "]")
	// First character must be '['
	if start != 0 || end < start {
		return
	}
	for _, option := range strings.Fields(title[start+1 : end]) {
		switch strings.ToLower(option) {
		case titleOptionSkipCI:
			titleOptions.SkipCI = true
		}
	}
	return
}

func processGitHubPullRequest(
	ctx *gin.Context,
	pr *github.PullRequestEvent,
	githubClient clientgithub.Client,
	conf *config,
) error {

	var (
		prRef  string
		err    error
		action = pr.GetAction()
	)
	log := getCustomLoggerFromContext(ctx).
		WithField("pull", pr.GetNumber()).
		WithField("action", action)
	req := pr.GetPullRequest()

	// Do not run if the PR is a draft
	if req.GetDraft() {
		log.Infof(
			"The PR: %s/%d is a draft. Do not run tests",
			pr.GetRepo().GetName(),
			pr.GetNumber(),
		)
		return nil
	}
	title := strings.TrimSpace(req.GetTitle())
	options := getTitleOptions(title)

	log.Debugf("Processing pull request action %s", action)
	switch action {
	case "opened", "reopened", "synchronize", "ready_for_review":
		// We always create a pr_* branch
		if prRef, err = syncPullRequestBranch(log, pr, conf); err != nil {
			log.Errorf("Could not create PR branch: %s", err.Error())
			msg := "There was an error syncing branches, " + msgDetailsKubernetesLog
			postGitHubMessage(ctx, pr, log, msg)
		}
		//and we run a pipeline only for the pr_* branch
		if prRef != "" {
			prNum := strconv.Itoa(pr.GetNumber())
			prBranchName := "pr_" + prNum
			isOrgMember := func() bool {
				return githubClient.IsOrganizationMember(
					ctx,
					conf.githubOrganization,
					pr.Sender.GetLogin(),
				)
			}
			if !options.SkipCI {
				err = retryOnError(retryParams{
					retryFunc: func() error {
						return startPRPipeline(log, prBranchName, pr, conf, isOrgMember)
					},
					compFunc: func(compareError error) bool {
						re := regexp.MustCompile("Missing CI config file|" +
							"No stages / jobs for this pipeline")
						switch {
						case compareError == nil:
							return noRetry
						case re.MatchString(compareError.Error()):
							log.Infof("start client pipeline for PR '%d' is skipped", pr.Number)
							return noRetry
						default:
							log.Errorf("failed to start client pipeline for PR: %s", compareError)
							return doRetry
						}
					},
				})
			}
			if err != nil {
				msg := "There was an error running your pipeline, " + msgDetailsKubernetesLog
				postGitHubMessage(ctx, pr, log, msg)
			}
		}

		handleChangelogComments(log, ctx, githubClient, pr, conf)

	case "closed":
		// Delete merged pr branches in GitLab
		if err := deleteStaleGitlabPRBranch(log, pr, conf); err != nil {
			log.Errorf(
				"Failed to delete the stale PR branch after the PR: %v was merged or closed. "+
					"Error: %v",
				pr,
				err,
			)
		}

		// If the pr was merged, suggest cherry-picks
		if err := suggestCherryPicks(log, pr, githubClient, conf); err != nil {
			log.Errorf("Failed to suggest cherry picks for the pr %v. Error: %v", pr, err)
		}
	}

	// Continue to the integration Pipeline only for organization members
	if member := githubClient.IsOrganizationMember(
		ctx,
		conf.githubOrganization,
		pr.Sender.GetLogin(),
	); !member {
		log.Warnf(
			"%s is making a pullrequest, but he/she is not a member of our organization, ignoring",
			pr.Sender.GetLogin(),
		)
		return nil
	}

	// First check if the PR has been merged. If so, stop
	// the pipeline, and do nothing else.
	if err := stopBuildsOfStaleClientPRs(log, pr, conf); err != nil {
		log.Errorf(
			"Failed to stop a stale build after the PR: %v was merged or closed. Error: %v",
			pr,
			err,
		)
	}

	// Keep the OS and Enterprise repos in sync
	if err := syncIfOSHasEnterpriseRepo(log, conf, pr); err != nil {
		log.Errorf("Failed to sync the OS and Enterprise repos: %s", err.Error())
	}

	// get the list of builds
	builds := parseClientPullRequest(log, conf, action, pr)
	log.Infof("%s:%d would trigger %d builds", pr.GetRepo().GetName(), pr.GetNumber(), len(builds))

	// do not start the builds, inform the user about the `start client pipeline` command instead
	if len(builds) > 0 {
		// Two possible pipelines: client or integration
		var botCommentString string
		if pr.GetRepo().GetName() == "integration" {
			botCommentString = `, start a full integration test pipeline with:
   - mentioning me and ` + "`" + commandStartIntegrationPipeline + "`"
		} else {
			botCommentString = `, start a full client pipeline with:
   - mentioning me and ` + "`" + commandStartClientPipeline + "`"
		}

		if getFirstMatchingBotCommentInPR(log, githubClient, pr, botCommentString, conf) == nil {

			msg := "@" + pr.GetSender().GetLogin() + botCommentString +
				commandStartClientPipeline + "\"."
			// nolint:lll
			msg += `

   ---

   <details>
   <summary>my commands and options</summary>
   <br />

   You can prevent me from automatically starting CI pipelines:
   - if your pull request title starts with "[NoCI] ..."

   You can trigger a client pipeline on multiple prs with:
   - mentioning me and ` + "`" + `start client pipeline --pr mender/127 --pr mender-connect/255` + "`" + `

   You can trigger GitHub->GitLab branch sync with:
   - mentioning me and ` + "`" + `sync` + "`" + `

   You can cherry pick to a given branch or branches with:
   - mentioning me and:
   ` + "```" + `
    cherry-pick to:
    * 1.0.x
    * 2.0.x
   ` + "```" + `
   </details>
   `
			postGitHubMessage(ctx, pr, log, msg)
		} else {
			log.Infof(
				"I have already commented on the pr: %s/%d, no need to keep on nagging",
				pr.GetRepo().GetName(), pr.GetNumber())
		}
	}

	return nil
}

func postGitHubMessage(
	ctx *gin.Context,
	pr *github.PullRequestEvent,
	log *logrus.Entry,
	msg string,
) {
	if err := githubClient.CreateComment(
		ctx,
		pr.GetOrganization().GetLogin(),
		pr.GetRepo().GetName(),
		pr.GetNumber(),
		&github.IssueComment{Body: github.String(msg)},
	); err != nil {
		log.Infof("Failed to comment on the pr: %v, Error: %s", pr, err.Error())
	}
}

func getFirstMatchingBotCommentInPR(
	log *logrus.Entry,
	githubClient clientgithub.Client,
	pr *github.PullRequestEvent,
	botComment string,
	conf *config,
) *github.IssueComment {

	comments, err := githubClient.ListComments(
		context.Background(),
		conf.githubOrganization,
		pr.GetRepo().GetName(),
		pr.GetNumber(),
		&github.IssueListCommentsOptions{
			Sort:      "created",
			Direction: "asc",
		})
	if err != nil {
		log.Errorf("Failed to list the comments on PR: %s/%d, err: '%s'",
			pr.GetRepo().GetName(), pr.GetNumber(), err)
		return nil
	}
	for _, comment := range comments {
		if comment.Body != nil &&
			strings.Contains(*comment.Body, botComment) &&
			comment.User != nil &&
			comment.User.Login != nil &&
			*comment.User.Login == githubBotName {
			return comment
		}
	}
	return nil
}

func handleChangelogComments(
	log *logrus.Entry,
	ctx *gin.Context,
	githubClient clientgithub.Client,
	pr *github.PullRequestEvent,
	conf *config,
) {
	// It would be semantically correct to update the integration repo
	// here. However, this step is carried out on every PR update, causing a
	// big amount of "git fetch" requests, which both reduces performance,
	// and could result in rate limiting. Instead, we assume that the
	// integration repo is recent enough, since it is still updated when
	// doing mender-qa builds.
	//
	// // First update integration repo.
	// err := updateIntegrationRepo(conf)
	// if err != nil {
	// 	log.Errorf("Could not update integration repo: %s", err.Error())
	// 	// Should still be safe to continue though.
	// }

	// Only do changelog commenting for mendersoftware repositories.
	if pr.GetPullRequest().GetBase().GetRepo().GetOwner().GetLogin() != "mendersoftware" {
		log.Info("Not a mendersoftware repository. Ignoring.")
		return
	}

	changelogText, warningText, err := fetchChangelogTextForPR(log, pr, conf)
	if err != nil {
		log.Errorf("Error while fetching changelog text: %s", err.Error())
		return
	}

	updatePullRequestChangelogComments(log, ctx, githubClient, pr, conf,
		changelogText, warningText)
}

func fetchChangelogTextForPR(
	log *logrus.Entry,
	pr *github.PullRequestEvent,
	conf *config,
) (string, string, error) {

	repo := pr.GetPullRequest().GetBase().GetRepo().GetName()
	baseSHA := pr.GetPullRequest().GetBase().GetSHA()
	headSHA := pr.GetPullRequest().GetHead().GetSHA()
	baseRef := pr.GetPullRequest().GetBase().GetRef()
	headRef := pr.GetPullRequest().GetHead().GetRef()
	versionRange := fmt.Sprintf(
		"%s..%s",
		baseSHA,
		headSHA,
	)

	log.Debugf("Getting changelog for repo (%s) and range (%s)", repo, versionRange)

	// Generate the changelog text for this PR.
	changelogText, warningText, err := getChangelogText(
		repo, versionRange, conf)
	if err != nil {
		err = errors.Wrap(err, "Not able to get changelog text")
	}

	// Replace SHAs with the original ref names, so that the changelog text
	// does not change on every commit amend. The reason we did not use ref
	// names to begin with is that they may live in personal forks, so it
	// complicates the fetching mechanism. SHAs however, are always present
	// in the repository you are merging into.
	//
	// Fetching changelogs online from personal forks is pretty unlikely to
	// be useful outside of the integration-test-runner niche (better to use
	// the local version), therefore we do this replacement instead of
	// making the changelog-generator "fork aware".
	changelogText = strings.ReplaceAll(changelogText, baseSHA, baseRef)
	changelogText = strings.ReplaceAll(changelogText, headSHA, headRef)

	log.Debugf("Prepared changelog text: %s", changelogText)
	log.Debugf("Got warning text: %s", warningText)

	return changelogText, warningText, err
}

func assembleCommentText(changelogText, warningText string) string {
	commentText := changelogPrefix + changelogText
	if warningText != "" {
		commentText += warningHeader + warningText
	}
	return commentText
}

func updatePullRequestChangelogComments(
	log *logrus.Entry,
	ctx *gin.Context,
	githubClient clientgithub.Client,
	pr *github.PullRequestEvent,
	conf *config,
	changelogText string,
	warningText string,
) {
	var err error

	commentText := assembleCommentText(changelogText, warningText)
	emptyChangelog := (changelogText == "" ||
		strings.HasSuffix(changelogText, "### Changelogs\n\n"))

	comment := getFirstMatchingBotCommentInPR(log, githubClient, pr, changelogPrefix, conf)
	if comment != nil {
		// There is a previous comment about changelog.
		if *comment.Body == commentText {
			log.Debugf("The changelog hasn't changed (comment ID: %d). Leave it alone.",
				comment.ID)
			return
		} else {
			log.Debugf("Deleting old changelog comment (comment ID: %d).",
				comment.ID)
			err = githubClient.DeleteComment(
				ctx,
				conf.githubOrganization,
				pr.GetRepo().GetName(),
				*comment.ID,
			)
			if err != nil {
				log.Errorf("Could not delete changelog comment: %s",
					err.Error())
			}
		}
	} else if emptyChangelog {
		log.Info("Changelog is empty, and there is no previous changelog comment. Stay silent.")
		return
	}

	commentBody := &github.IssueComment{
		Body: &commentText,
	}
	err = githubClient.CreateComment(
		ctx,
		conf.githubOrganization,
		pr.GetRepo().GetName(),
		pr.GetNumber(),
		commentBody,
	)
	if err != nil {
		log.Errorf("Could not post changelog comment: %s. Comment text: %s",
			err.Error(), commentText)
	}
}
