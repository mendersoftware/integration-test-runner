package main

import (
	"context"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"

	clientgithub "github.com/mendersoftware/integration-test-runner/client/github"
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
				log.Errorf("failed to start pipeline for PR: %s", err)
				// post a comment only if GitLab is supposed to start a pipeline
				gitlabReplyErrorMsg := "Missing CI config file"
				if !strings.Contains(err.Error(), gitlabReplyErrorMsg) {
					msg := "There was an error running your pipeline, " + msgDetails
					postGitHubMessage(ctx, pr, log, msg)
				}
			}
		}

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

		// make sure we only parse one pr at a time, since we use release_tool
		mutex.Lock()

		// If the pr was merged, suggest cherry-picks
		if err := suggestCherryPicks(log, pr, githubClient, conf); err != nil {
			log.Errorf("Failed to suggest cherry picks for the pr %v. Error: %v", pr, err)
		}

		// release the mutex
		mutex.Unlock()
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

	// make sure we only parse one pr at a time, since we use release_tool
	mutex.Lock()

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

	// release the mutex
	mutex.Unlock()

	// do not start the builds, inform the user about the `start pipeline` command instead
	if len(builds) > 0 {
		// Only comment, if not already commented on a PR
		botCommentString := ", Let me know if you want to start the integration pipeline by " +
			"mentioning me and the command \""
		if !botHasAlreadyCommentedOnPR(log, githubClient, pr, botCommentString, conf) {

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

func botHasAlreadyCommentedOnPR(
	log *logrus.Entry,
	githubClient clientgithub.Client,
	pr *github.PullRequestEvent,
	botComment string,
	conf *config,
) bool {
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
		return false
	}
	for _, comment := range comments {
		if comment.Body != nil && strings.Contains(*comment.Body, botComment) {
			return true
		}
	}
	return false
}
