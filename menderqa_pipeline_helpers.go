package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"

	clientgithub "github.com/mendersoftware/integration-test-runner/client/github"
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

// Legacy: Mender Client 5.0.x and below. Uses release_tool.py from the
// integration repo. Remove when deprecating the old release process.
func getIntegrationVersionsLegacy(
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
			"getIntegrationVersionsLegacy: Error: %v (%s)",
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

// MenderClientRelease represents a Mender Client release as defined in
// mender-client-subcomponents/subcomponents/releases/*.json
type MenderClientRelease struct {
	Version       string                     `json:"version"`
	Subcomponents []MenderClientSubcomponent `json:"components"`
}

// MenderClientSubcomponent represents a single subcomponent in a release.
type MenderClientSubcomponent struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Source  string `json:"source"`
}

// repoNameFromSource extracts the GitHub repo name from the source field.
// e.g., "github.com/mendersoftware/monitor-client" -> "monitor-client"
func repoNameFromSource(source string) string {
	return path.Base(source)
}

var maintenanceBranchPattern = regexp.MustCompile(`^\d+\.\d+\.x$`)

// fetchMenderClientReleases fetches all maintenance release JSON files from the
// mender-client-subcomponents repo on GitHub (main branch) and returns the
// parsed releases.
func fetchMenderClientReleases(
	ctx context.Context,
	log *logrus.Entry,
	client clientgithub.Client,
) ([]MenderClientRelease, error) {
	_, dirContents, err := client.GetContents(
		ctx,
		"mendersoftware",
		"mender-client-subcomponents",
		"subcomponents/releases",
		&github.RepositoryContentGetOptions{Ref: "main"},
	)
	if err != nil {
		return nil, fmt.Errorf("fetchMenderClientReleases: failed to list releases: %w", err)
	}

	var releases []MenderClientRelease
	for _, entry := range dirContents {
		name := entry.GetName()
		// Only consider maintenance branch files (e.g., 6.0.x.json).
		// Skip tagged releases (6.0.0.json) and next.json.
		if !strings.HasSuffix(name, ".json") ||
			!maintenanceBranchPattern.MatchString(strings.TrimSuffix(name, ".json")) {
			continue
		}

		// Fetch the individual file (directory listings don't include content)
		fileContent, _, err := client.GetContents(
			ctx,
			"mendersoftware",
			"mender-client-subcomponents",
			"subcomponents/releases/"+name,
			&github.RepositoryContentGetOptions{Ref: "main"},
		)
		if err != nil {
			return nil, fmt.Errorf(
				"fetchMenderClientReleases: failed to fetch %s: %w", name, err,
			)
		}

		content, err := fileContent.GetContent()
		if err != nil {
			return nil, fmt.Errorf(
				"fetchMenderClientReleases: failed to decode %s: %w", name, err,
			)
		}

		var release MenderClientRelease
		if err := json.Unmarshal([]byte(content), &release); err != nil {
			return nil, fmt.Errorf(
				"fetchMenderClientReleases: failed to parse %s: %w", name, err,
			)
		}

		log.Infof("fetchMenderClientReleases: loaded release %s", release.Version)
		releases = append(releases, release)
	}

	return releases, nil
}

// releaseVersions returns a list of version strings for logging.
func releaseVersions(releases []MenderClientRelease) []string {
	versions := make([]string, len(releases))
	for i, r := range releases {
		versions[i] = r.Version
	}
	return versions
}

// findMatchingReleases returns all releases that contain the given repo at
// the given branch. For example, findMatchingReleases(releases, "mender-connect", "3.0.x")
// returns all releases where a component sourced from mender-connect has version "3.0.x".
func findMatchingReleases(
	releases []MenderClientRelease,
	repo, branch string,
) []MenderClientRelease {
	var matched []MenderClientRelease
	for _, release := range releases {
		for _, comp := range release.Subcomponents {
			if repoNameFromSource(comp.Source) == repo && comp.Version == branch {
				matched = append(matched, release)
				break
			}
		}
	}
	return matched
}
