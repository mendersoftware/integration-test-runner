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

func createPullRequestBranch(log *logrus.Entry, pr *github.PullRequestEvent, conf *config) error {

	action := pr.GetAction()
	if action != "opened" && action != "edited" && action != "reopened" &&
		action != "synchronize" && action != "ready_for_review" {
		log.Infof("createPullRequestBranch: Action %s, ignoring", action)
		return nil
	}

	prHeadFork := pr.GetPullRequest().GetHead().GetUser().GetLogin()
	if prHeadFork == "mendersoftware" {
		log.Debug("createPullRequestBranch: PR head is a branch in mendersoftware, ignoring")
		return nil
	}

	repo := pr.GetRepo().GetName()
	org := pr.GetOrganization().GetLogin()
	prNum := strconv.Itoa(pr.GetNumber())

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

	prBranchName := "pr_" + prNum
	gitcmd = git.Command("fetch", "github", "pull/"+prNum+"/head:"+prBranchName)
	gitcmd.Dir = tmpdir
	out, err = gitcmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v returned error: %s: %s", gitcmd.Args, out, err.Error())
	}

	gitcmd = git.Command("push", "-f", "--set-upstream", "gitlab", prBranchName)
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

	gitcmd = git.Command("push", "gitlab", "--delete", fmt.Sprintf("pr_%d", pr.GetNumber()))
	gitcmd.Dir = tmpdir
	out, err = gitcmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v returned error: %s: %s", gitcmd.Args, out, err.Error())
	}

	return nil

}
