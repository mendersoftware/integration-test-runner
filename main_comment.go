package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

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

func syncProtectedBranch(
	log *logrus.Entry,
	pr *github.PullRequestEvent,
	conf *config,
	pipelinePath string,
) (string, error) {
	prBranchName := "pr_" + strconv.Itoa(pr.GetNumber()) + "_protected"
	if err := syncBranch(prBranchName, log, pr, conf); err != nil {
		mainErrMsg := "There was an error syncing branches"
		return "", fmt.Errorf("%v returned error: %s: %s", err, mainErrMsg, err.Error())
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
						revision = "pull/" + userInputParts[1] + "/head"
					} else {
						// feature branch
						revision = userInputParts[1]
					}
				case 3: // deviceconnect/pull/12 syntax
					revision = strings.Join(userInputParts[1:], "/") + "/head"
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
		}
	}

	return buildOptions, err
}
