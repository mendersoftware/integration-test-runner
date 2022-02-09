package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/mendersoftware/integration-test-runner/git"
)

func shouldStartPipeline(branchName string) bool {
	startByName := []string{
		"master",
		"staging",
		"production",
		"hosted",
	}
	for _, n := range startByName {
		if branchName == n {
			return true
		}
	}

	startByRegEx := []*regexp.Regexp{
		regexp.MustCompile(`^[0-9]+\.[0-9]+\.x`),
		regexp.MustCompile(`^pr_[0-9]+$`),
	}
	for _, n := range startByRegEx {
		if n.MatchString(branchName) {
			return true
		}
	}

	return false
}

func syncRemoteRef(log *logrus.Entry, org, repo, ref string, conf *config) error {

	remoteURLGitLab, err := getRemoteURLGitLab(org, repo)
	if err != nil {
		return fmt.Errorf("getRemoteURLGitLab returned error: %s", err.Error())
	}

	state, err := git.Commands(
		git.Command("init", "."),
		git.Command("remote", "add", "github",
			getRemoteURLGitHub(conf.githubProtocol, conf.githubOrganization, repo)),
		git.Command("remote", "add", "gitlab", remoteURLGitLab),
	)
	defer state.Cleanup()
	if err != nil {
		return err
	}

	if strings.Contains(ref, "tags") {
		tagName := strings.TrimPrefix(ref, "refs/tags/")

		err := git.Command("fetch", "--tags", "github").With(state).Run()
		if err != nil {
			return err
		}
		err = git.Command("push", "-f", "gitlab", tagName).With(state).Run()
		if err != nil {
			return err
		}
	} else if strings.Contains(ref, "heads") {
		branchName := strings.TrimPrefix(ref, "refs/heads/")

		err := git.Command("fetch", "github").With(state).Run()
		if err != nil {
			return err
		}
		err = git.Command("checkout", "-b", branchName, "github/"+branchName).With(state).Run()
		if err != nil {
			return err
		}
		// For the push, add option ci.skip for mender-qa
		cmdArgs := []string{"push", "-f"}
		if repo == "mender-qa" {
			cmdArgs = append(cmdArgs, "-o", "ci.skip")
		}
		if !shouldStartPipeline(branchName) {
			cmdArgs = append(cmdArgs, "-o", "ci.skip")
		}
		cmdArgs = append(cmdArgs, "gitlab", branchName)
		err = git.Command(cmdArgs...).With(state).Run()
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Unrecognized ref %s", ref)
	}

	log.Infof("Pushed ref to GitLab: %s:%s", repo, ref)
	return nil
}
