package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	clientgitlab "github.com/mendersoftware/integration-test-runner/client/gitlab"
	"github.com/mendersoftware/integration-test-runner/git"
)

func startPRPipeline(
	log *logrus.Entry,
	ref string,
	event *github.PullRequestEvent,
	conf *config,
	isOrgMember func() bool,
) error {
	client, err := clientgitlab.NewGitLabClient(
		conf.gitlabToken,
		conf.gitlabBaseURL,
		conf.dryRunMode,
	)
	if err != nil {
		return err
	}
	pr := event.GetPullRequest()
	org := event.GetOrganization().GetLogin()
	head := pr.GetHead()
	base := pr.GetBase()
	repo := event.GetRepo()
	if repo.GetName() == "mender-qa" {
		// Verify that the pipe is started by a member of the organization
		if isOrgMember() {
			log.Warnf(
				"%s is making a pullrequest, but he/she is not a member of our organization, "+
					"ignoring",
				pr.GetUser().GetLogin(),
			)
			return nil
		}
	}
	repoURL, err := getRemoteURLGitLab(org, repo.GetName())
	if err != nil {
		return err
	}
	repoHostURI := strings.SplitN(repoURL, ":", 2)
	if len(repoHostURI) != 2 {
		return fmt.Errorf("invalid GitLab URL '%s': failed to start GitLab pipeline", repoURL)
	}
	gitlabPath := repoHostURI[1]

	ciIIDKey := "CI_EXTERNAL_PULL_REQUEST_IID"
	ciIID := strconv.Itoa(event.GetNumber())
	ciSourceRepoKey := "CI_EXTERNAL_PULL_REQUEST_SOURCE_REPOSITORY"
	ciSourceRepo := head.GetRepo().GetFullName()
	ciTargetRepoKey := "CI_EXTERNAL_PULL_REQUEST_TARGET_REPOSITORY"
	ciTargetRepo := repo.GetFullName()
	ciSourceBranchNameKey := "CI_EXTERNAL_PULL_REQUEST_SOURCE_BRANCH_NAME"
	ciSourceBranchName := head.GetRef()
	ciSourceBranchSHAKey := "CI_EXTERNAL_PULL_REQUEST_SOURCE_BRANCH_SHA"
	ciSourceBranchSHA := head.GetSHA()
	ciTargetBranchNameKey := "CI_EXTERNAL_PULL_REQUEST_TARGET_BRANCH_NAME"
	ciTargetBranchName := base.GetRef()
	ciTargetBranchShaKey := "CI_EXTERNAL_PULL_REQUEST_TARGET_BRANCH_SHA"
	ciTargetBranchSha := base.GetSHA()

	pipeline, err := client.CreatePipeline(gitlabPath, &gitlab.CreatePipelineOptions{
		Ref: &ref,
		Variables: &[]*gitlab.PipelineVariableOptions{
			{Key: &ciIIDKey, Value: &ciIID},
			{Key: &ciSourceRepoKey, Value: &ciSourceRepo},
			{Key: &ciTargetRepoKey, Value: &ciTargetRepo},
			{Key: &ciSourceBranchNameKey, Value: &ciSourceBranchName},
			{Key: &ciSourceBranchSHAKey, Value: &ciSourceBranchSHA},
			{Key: &ciTargetBranchNameKey, Value: &ciTargetBranchName},
			{Key: &ciTargetBranchShaKey, Value: &ciTargetBranchSha},
		},
	})
	if err != nil {
		return err
	} else {
		log.Debugf("started pipeline for PR: %s", pipeline.WebURL)
	}

	return nil
}

func syncPullRequestBranch(
	log *logrus.Entry,
	pr *github.PullRequestEvent,
	conf *config,
) (string, error) {
	prBranchName := "pr_" + strconv.Itoa(pr.GetNumber())
	if err := syncBranch(prBranchName, log, pr, conf); err != nil {
		mainErrMsg := "There was an error syncing branches"
		return "", fmt.Errorf("%v returned error: %s: %s", err, mainErrMsg, err.Error())
	}
	return prBranchName, nil
}

func syncBranch(
	prBranchName string,
	log *logrus.Entry,
	pr *github.PullRequestEvent,
	conf *config,
) error {

	repo := pr.GetRepo().GetName()
	prNum := strconv.Itoa(pr.GetNumber())

	tmpdir, err := os.MkdirTemp("", repo)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)

	gitcmd := git.Command("init", ".")
	gitcmd.Dir = tmpdir
	out, err := gitcmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v returned error: %s: %s", gitcmd.Args, out, err.Error())
	}

	repoURL := getRemoteURLGitHub(conf.githubProtocol, conf.githubOrganization, repo)
	gitcmd = git.Command("remote", "add", "github", repoURL)
	gitcmd.Dir = tmpdir
	out, err = gitcmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v returned error: %s: %s", gitcmd.Args, out, err.Error())
	}

	remoteURL, err := getRemoteURLGitLab(conf.githubOrganization, repo)
	if err != nil {
		return fmt.Errorf("getRemoteURLGitLab returned error: %s", err.Error())
	}

	gitcmd = git.Command("remote", "add", "gitlab", remoteURL)
	gitcmd.Dir = tmpdir
	out, err = gitcmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v returned error: %s: %s", gitcmd.Args, out, err.Error())
	}

	gitcmd = git.Command("fetch", "github", "pull/"+prNum+"/head:"+prBranchName)
	gitcmd.Dir = tmpdir
	out, err = gitcmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v returned error: %s: %s", gitcmd.Args, out, err.Error())
	}

	// Push but not don't trigger CI (yet)
	gitcmd = git.Command("push", "-f", "-o", "ci.skip", "--set-upstream", "gitlab", prBranchName)
	gitcmd.Dir = tmpdir
	out, err = gitcmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v returned error: %s: %s", gitcmd.Args, out, err.Error())
	}

	log.Infof("Created branch: %s:%s", repo, prBranchName)
	return nil
}

func deleteStaleGitlabPRBranch(log *logrus.Entry, pr *github.PullRequestEvent, conf *config) error {
	// If the action is "closed", the pull request was merged or just closed,
	// stop builds in both cases.
	if pr.GetAction() != "closed" {
		log.Debugf("deleteStaleGitlabPRBranch: PR not closed, therefore not stopping it's pipeline")
		return nil
	}

	// Call the DeletePRBranch function to delete the PR branch on GitLab
	response, err := deletePRBranch(pr, conf, fmt.Sprintf("pr_%d", pr.GetNumber()), log)
	if err != nil {
		return fmt.Errorf("Got response: %d. Failed to delete PR branch: %s",
			response.StatusCode,
			err.Error(),
		)
	}

	if pr.GetRepo().GetName() == "integration" {
		// check if we have a protected branch and try to delete it
		response, err := deletePRBranch(pr, conf,
			fmt.Sprintf("pr_%d_protected", pr.GetNumber()), log)
		if err != nil {
			// Don't return error if the branch doesn't exist
			if response.StatusCode != 404 {
				return fmt.Errorf("Got response: %d. Failed to delete PR branch: %s",
					response.StatusCode,
					err.Error(),
				)
			}
		}
	}

	return nil
}

func deletePRBranch(
	pr *github.PullRequestEvent,
	conf *config,
	prBranchName string,
	log *logrus.Entry,
) (*gitlab.Response, error) {

	repoName := pr.GetRepo().GetName()

	group, ok := gitHubOrganizationToGitLabGroup[conf.githubOrganization]
	if !ok {
		return nil, fmt.Errorf("Unrecognized organization %q", conf.githubOrganization)
	}
	path := "Northern.tech/" + group + "/" + repoName

	client, err := clientgitlab.NewGitLabClient(
		conf.gitlabToken,
		conf.gitlabBaseURL,
		conf.dryRunMode,
	)
	if err != nil {
		return nil, err
	}

	response, err := client.DeleteBranch(path, prBranchName, nil)
	if err != nil {
		return response, err
	}

	// Deleting the branch does _not_ delete the protection
	// Attempt to delete the branch protection as well
	response, err = client.UnprotectRepositoryBranches(
		path,
		prBranchName,
		nil,
	)
	if err != nil {
		// Do not return error if the branch protection doesn't exist
		if response.StatusCode != 404 {
			return response, err
		}
	}

	return response, nil
}
