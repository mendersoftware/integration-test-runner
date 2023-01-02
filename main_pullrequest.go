package main

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v28/github"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	clientgithub "github.com/mendersoftware/integration-test-runner/client/github"
)

var (
	changelogPrefix = "Merging these commits will result in the following changelog entries:\n\n"
	warningHeader   = "\n\n## Warning\n\nGenerating changelogs also resulted in these warnings:\n\n"
)

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

	log.Debugf("Processing pull request action %s", action)
	switch action {
	case "opened", "edited", "reopened", "synchronize", "ready_for_review":
		msgDetails := "see <a href=\"https://console.cloud.google.com/kubernetes/" +
			"deployment/us-east1/company-websites/default/test-runner-mender-io/logs?" +
			"project=gp-kubernetes-269000\">logs</a> for details."
		// We always create a pr_* branch
		if prRef, err = syncPullRequestBranch(log, pr, conf); err != nil {
			log.Errorf("Could not create PR branch: %s", err.Error())
			msg := "There was an error syncing branches, " + msgDetails
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
			err = startPRPipeline(log, prBranchName, pr, conf, isOrgMember)
			if err != nil {
				// post a comment only if GitLab is supposed to start a pipeline
				re := regexp.MustCompile("Missing CI config file|" +
					"No stages / jobs for this pipeline")
				if re.MatchString(err.Error()) {
					log.Infof("start pipeline for PR '%d' is skipped", pr.Number)
				} else {
					log.Errorf("failed to start pipeline for PR: %s", err)
					msg := "There was an error running your pipeline, " + msgDetails
					postGitHubMessage(ctx, pr, log, msg)
				}
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
	if err := stopBuildsOfStalePRs(log, pr, conf); err != nil {
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
	builds := parsePullRequest(log, conf, action, pr)
	log.Infof("%s:%d would trigger %d builds", pr.GetRepo().GetName(), pr.GetNumber(), len(builds))

	// do not start the builds, inform the user about the `start pipeline` command instead
	if len(builds) > 0 {
		// Only comment, if not already commented on a PR
		botCommentString := ", Let me know if you want to start the integration pipeline by " +
			"mentioning me and the command \""
		if getFirstMatchingBotCommentInPR(log, githubClient, pr, botCommentString, conf) == nil {

			msg := "@" + pr.GetSender().GetLogin() + botCommentString + commandStartPipeline + "\"."
			msg += `

   ---

   <details>
   <summary>my commands and options</summary>
   <br />
   You can trigger a pipeline on multiple prs with:

   - mentioning me and ` + "`" + `start pipeline --pr mender/127 --pr mender-connect/255` + "`" + `

   You can start a fast pipeline, disabling full integration tests with:

   - mentioning me and ` + "`" + `start pipeline --fast` + "`" + `

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
	// First update integration repo.
	err := updateIntegrationRepo(conf)
	if err != nil {
		log.Errorf("Could not update integration repo: %s", err.Error())
		// Should still be safe to continue though.
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
