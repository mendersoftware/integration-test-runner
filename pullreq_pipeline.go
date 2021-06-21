package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/google/go-github/v28/github"
	"github.com/mendersoftware/integration-test-runner/git"
	"github.com/sirupsen/logrus"
)

func getGitlabBranch(pr *github.PullRequestEvent) string {
	prHead := pr.GetPullRequest().GetHead()
	if prHead.GetUser().GetLogin() == "mendersoftware" {
		return prHead.GetRef()
	}
	return "pr_" + strconv.Itoa(pr.GetNumber())
}

func syncPullRequest(log *logrus.Entry, pr *github.PullRequestEvent, conf *config) error {

	action := pr.GetAction()
	if action != "opened" && action != "edited" && action != "reopened" &&
		action != "synchronize" && action != "ready_for_review" {
		log.Infof("syncPullRequest: Action %s, ignoring", action)
		return nil
	}

	repo := pr.GetRepo().GetName()
	org := pr.GetOrganization().GetLogin()
	prNum := strconv.Itoa(pr.GetNumber())
	req := pr.GetPullRequest()
	base := req.GetBase()
	targetRepo := base.GetRepo().GetFullName()
	targetBranch := base.GetRef()
	targetSHA := base.GetSHA()
	head := req.GetHead()
	sourceRepo := head.GetRepo().GetFullName()
	sourceBranch := head.GetRef()
	sourceSHA := head.GetSHA()

	tmpdir, err := ioutil.TempDir("", repo)
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

	repoURL := getRemoteURLGitHub(conf.githubProtocol, githubOrganization, repo)
	gitcmd = git.Command("remote", "add", "github", repoURL)
	gitcmd.Dir = tmpdir
	out, err = gitcmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v returned error: %s: %s", gitcmd.Args, out, err.Error())
	}

	remoteURL, err := getRemoteURLGitLab(org, repo)
	if err != nil {
		return fmt.Errorf("getRemoteURLGitLab returned error: %s", err.Error())
	}

	gitcmd = git.Command("remote", "add", "gitlab", remoteURL)
	gitcmd.Dir = tmpdir
	out, err = gitcmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v returned error: %s: %s", gitcmd.Args, out, err.Error())
	}

	prBranchName := getGitlabBranch(pr)
	gitcmd = git.Command("fetch", "github", "pull/"+prNum+"/head:"+prBranchName)
	gitcmd.Dir = tmpdir
	out, err = gitcmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v returned error: %s: %s", gitcmd.Args, out, err.Error())
	}
	gitcmd = git.Command(
		"push", "-f",
		// The following push options are simulating gitlab repository mirroring
		// and are for resolving coverage reports PRs.
		"-o", `ci.variable="CI_EXTERNAL_PULL_REQUEST_IID=`+prNum+`"`,
		"-o", `ci.variable="CI_EXTERNAL_PULL_REQUEST_SOURCE_REPOSITORY=`+sourceRepo+`"`,
		"-o", `ci.variable="CI_EXTERNAL_PULL_REQUEST_TARGET_REPOSITORY=`+targetRepo+`"`,
		"-o", `ci.variable="CI_EXTERNAL_PULL_REQUEST_SOURCE_BRANCH_NAME=`+sourceBranch+`"`,
		"-o", `ci.variable="CI_EXTERNAL_PULL_REQUEST_TARGET_BRANCH_NAME=`+targetBranch+`"`,
		"-o", `ci.variable="CI_EXTERNAL_PULL_REQUEST_SOURCE_BRANCH_SHA=`+sourceSHA+`"`,
		"-o", `ci.variable="CI_EXTERNAL_PULL_REQUEST_TARGET_BRANCH_SHA=`+targetSHA+`"`,
		"--set-upstream", "gitlab", prBranchName,
	)
	gitcmd.Dir = tmpdir
	out, err = gitcmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v returned error: %s: %s", gitcmd.Args, out, err.Error())
	}

	log.Infof("Created branch: %s:%s", repo, prBranchName)
	log.Info("Pipeline is expected to start automatically")
	return nil
}

func deleteStaleGitlabPRBranch(log *logrus.Entry, pr *github.PullRequestEvent, conf *config) error {

	// If the action is "closed" the pull request was merged or just closed,
	// stop builds in both cases.
	if pr.GetAction() != "closed" {
		log.Debugf("deleteStaleGitlabPRBranch: PR not closed, therefore not stopping it's pipeline")
		return nil
	}
	repoName := pr.GetRepo().GetName()
	repoOrg := pr.GetOrganization().GetLogin()
	prBranchName := getGitlabBranch(pr)

	tmpdir, err := ioutil.TempDir("", repoName)
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

	remoteURL, err := getRemoteURLGitLab(repoOrg, repoName)
	if err != nil {
		return fmt.Errorf("getRemoteURLGitLab returned error: %s", err.Error())
	}

	gitcmd = git.Command("remote", "add", "gitlab", remoteURL)
	gitcmd.Dir = tmpdir
	out, err = gitcmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v returned error: %s: %s", gitcmd.Args, out, err.Error())
	}

	gitcmd = git.Command("fetch", "gitlab")
	gitcmd.Dir = tmpdir
	out, err = gitcmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v returned error: %s: %s", gitcmd.Args, out, err.Error())
	}

	gitcmd = git.Command("push", "gitlab", "--delete", prBranchName)
	gitcmd.Dir = tmpdir
	out, err = gitcmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v returned error: %s: %s", gitcmd.Args, out, err.Error())
	}

	return nil

}
