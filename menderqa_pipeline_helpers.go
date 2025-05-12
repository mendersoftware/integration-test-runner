package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

func getServiceRevisionFromIntegration(repo, baseBranch string, conf *config) (string, error) {
	c := exec.Command(
		"python3",
		"release_tool.py",
		"--version-of",
		repo,
		"--in-integration-version",
		baseBranch,
	)
	c.Dir = conf.integrationDirectory + "/extra/"
	out, err := c.Output()
	if err != nil {
		err = fmt.Errorf("getServiceRevisionFromIntegration: Error: %v (%s)", err, out)
	}
	version := string(out)

	// remove the remote (ex: "`origin`/1.0.x")
	if strings.Contains(version, "/") {
		version = strings.SplitN(version, "/", 2)[1]
	}
	return strings.TrimSpace(string(version)), err
}

// The parameter that the build system uses for repo specific revisions is <REPO_NAME>_REV
func repoToBuildParameter(repo string) string {
	repoRevision := strings.ToUpper(repo) + "_REV"
	return strings.Replace(repoRevision, "-", "_", -1)
}

// Use python script in order to determine which integration branches to test with
func getIntegrationVersionsUsingMicroservice(
	log *logrus.Entry,
	repo, version string,
	conf *config,
) ([]string, error) {
	cmdArgs := []string{
		"release_tool.py",
		"--integration-versions-including",
		repo,
		"--version",
		version,
	}
	if strings.HasPrefix(version, featureBranchPrefix) {
		cmdArgs = append(cmdArgs, "--feature-branches")
	}
	c := exec.Command("python3", cmdArgs...)
	c.Dir = conf.integrationDirectory + "/extra/"
	integrations, err := c.Output()

	if err != nil {
		return nil, fmt.Errorf(
			"getIntegrationVersionsUsingMicroservice: Error: %v (%s)",
			err,
			integrations,
		)
	}

	branches := strings.Split(strings.TrimSpace(string(integrations)), "\n")

	// remove the remote (ex: "`origin`/1.0.x")
	for idx, branch := range branches {
		if strings.Contains(branch, "/") {
			branches[idx] = strings.Split(branch, "/")[1]
		}
	}

	// filter out "staging" branch if version is "master"
	// Reasoning: integration/staging will have "master" versions for client side
	// unreleased components, making the bot trigger two pipelines, wasting
	// resources and confusing developers...
	i := 0
	for _, branch := range branches {
		if version == "master" && branch == "staging" {
			continue
		}
		branches[i] = branch
		i++

	}
	branches = branches[:i]

	log.Infof("%s/%s is being used in the following integration: %s", repo, version, branches)
	return branches, nil
}

func getListOfVersionedRepositories(inVersion string, conf *config) ([]string, error) {
	c := exec.Command("python3", "release_tool.py", "--list", "--in-integration-version", inVersion)
	c.Dir = conf.integrationDirectory + "/extra/"
	output, err := c.Output()
	if err != nil {
		return nil, fmt.Errorf("getListOfVersionedRepositories: Error: %v (%s)", err, output)
	}

	return strings.Split(strings.TrimSpace(string(output)), "\n"), nil
}
