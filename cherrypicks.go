package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"

	clientgithub "github.com/mendersoftware/integration-test-runner/client/github"
	"github.com/mendersoftware/integration-test-runner/git"
)

var versionsUrl = "https://docs.mender.io/releases/versions.json"

var errorCherryPickConflict = errors.New("Cherry pick had conflicts")

const apiWarningString = "\nNote: Suggestions could not be fetched from " +
	"[release version endpoint](%s) and are therefore taken from integration."

type versions struct {
	Lts []string `json:",omitempty"`
}

func (v *versions) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, v)
}

func getLatestReleaseFromApi(url string) ([]string, error) {
	client := http.Client{
		Timeout: time.Second * 2,
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	v := versions{}
	err = v.Unmarshal(body)
	if err != nil {
		return nil, err
	}
	if len(v.Lts) == 0 {
		return nil, errors.New("getLatestReleaseFromApi: lts version list is empty")
	}
	for idx, val := range v.Lts {
		v.Lts[idx] = val + ".x"
	}
	return v.Lts, nil
}

func getLatestIntegrationRelease(number int, conf *config) ([]string, error) {
	cmd := fmt.Sprintf(
		"git for-each-ref --sort=-creatordate --format='%%(refname:short)' 'refs/tags' "+
			"| sed -E '/(^[0-9]+\\.[0-9]+)\\.[0-9]+$/!d;s//\\1.x/' |  uniq "+
			"| head -n 4 | sort -V -r | head -n %d",
		number,
	)
	c := exec.Command("sh", "-c", cmd)
	c.Dir = conf.integrationDirectory + "/extra/"
	version, err := c.Output()
	if err != nil {
		err = fmt.Errorf("getLatestIntegrationRelease: Error: %v (%s)", err, version)
	}
	versionStr := strings.TrimSpace(string(version))
	return strings.SplitN(versionStr, "\n", -1), err
}

// suggestCherryPicks suggests cherry-picks to release branches if the PR has been merged to master
func suggestCherryPicks(
	log *logrus.Entry,
	pr *github.PullRequestEvent,
	githubClient clientgithub.Client,
	conf *config,
) error {
	// ignore PRs if they are not closed and merged
	action := pr.GetAction()
	merged := pr.GetPullRequest().GetMerged()
	if action != "closed" || !merged {
		log.Infof("Ignoring cherry-pick suggestions for action: %s, merged: %v", action, merged)
		return nil
	}

	// ignore PRs if they don't target the master branch
	baseRef := pr.GetPullRequest().GetBase().GetRef()
	if baseRef != "master" {
		log.Infof("Ignoring cherry-pick suggestions for base ref: %s", baseRef)
		return nil
	}

	// initialize the git work area
	repo := pr.GetRepo().GetName()
	repoURL := getRemoteURLGitHub(conf.githubProtocol, conf.githubOrganization, repo)
	prNumber := strconv.Itoa(pr.GetNumber())
	prBranchName := "pr_" + prNumber
	state, err := git.Commands(
		git.Command("init", "."),
		git.Command("remote", "add", "github", repoURL),
		git.Command("fetch", "github", "master:local"),
		git.Command("fetch", "github", "pull/"+prNumber+"/head:"+prBranchName),
	)
	defer state.Cleanup()
	if err != nil {
		return err
	}

	// count the number commits with Changelog entries
	baseSHA := pr.GetPullRequest().GetBase().GetSHA()
	countCmd := exec.Command(
		"sh",
		"-c",
		"git log "+baseSHA+"...pr_"+prNumber+" | grep -i -e \"^    Changelog:\" "+
			"| grep -v -i -e \"^    Changelog: *none\" | wc -l",
	)
	countCmd.Dir = state.Dir
	out, err := countCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v returned error: %s: %s", countCmd.Args, out, err.Error())
	}

	changelogs, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	if changelogs == 0 {
		log.Infof("Found no changelog entries, ignoring cherry-pick suggestions")
		return nil
	}

	// fetch all the branches
	err = git.Command("fetch", "github").With(state).Run()
	if err != nil {
		return err
	}

	// nolint:lll
	tmplString := `
Hello :smile_cat: This PR contains changelog entries. Please, verify the need of backporting it to the following release branches:
{{.ReleaseBranches}}
`
	// get list of release versions
	versions, err := getLatestReleaseFromApi(versionsUrl)
	if err != nil {
		versions, err = getLatestIntegrationRelease(3, conf)
		if err != nil {
			return err
		}
		tmplString += fmt.Sprintf(apiWarningString, versionsUrl)
	}
	releaseBranches := []string{}
	for _, version := range versions {
		releaseBranch, err := getServiceRevisionFromIntegration(repo, "origin/"+version, conf)
		if err != nil {
			return err
		} else if releaseBranch != "" {
			if isCherryPickBottable(
				pr.GetRepo().GetName(),
				conf, pr.GetPullRequest(),
				releaseBranch,
			) {
				releaseBranches = append(
					releaseBranches,
					releaseBranch+" (release "+version+")"+" - :robot: :cherries:",
				)
			} else {
				releaseBranches = append(releaseBranches, releaseBranch+" (release "+version+")")
			}
		}
	}

	// no suggestions, stop here
	if len(releaseBranches) == 0 {
		return nil
	}

	// suggest cherry-picking with a comment
	tmpl, err := template.New("Main").Parse(tmplString)
	if err != nil {
		log.Errorf(
			"Failed to parse the build matrix template. Should never happen! Error: %s\n",
			err.Error(),
		)
	}
	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, struct {
		ReleaseBranches string
	}{
		ReleaseBranches: strings.Join(releaseBranches, "\n"),
	}); err != nil {
		log.Errorf("Failed to execute the build matrix template. Error: %s\n", err.Error())
	}

	// Comment with a pipeline-link on the PR
	commentBody := buf.String()
	comment := github.IssueComment{
		Body: &commentBody,
	}
	if err := githubClient.CreateComment(context.Background(), conf.githubOrganization,
		pr.GetRepo().GetName(), pr.GetNumber(), &comment); err != nil {
		log.Infof("Failed to comment on the pr: %v, Error: %s", pr, err.Error())
		return err
	}
	return nil
}

func isCherryPickBottable(
	repoName string,
	conf *config,
	pr *github.PullRequest,
	targetBranch string,
) bool {
	_, state, err := tryCherryPickToBranch(repoName, conf, pr, targetBranch)
	state.Cleanup()
	if err != nil {
		logrus.Errorf("isCherryPickBottable received error: %s", err.Error())
	}
	return err == nil
}

func tryCherryPickToBranch(
	repoName string,
	conf *config,
	pr *github.PullRequest,
	targetBranch string,
) (string, *git.State, error) {
	prBranchName := fmt.Sprintf("cherry-%s-%s",
		targetBranch, pr.GetHead().GetRef())
	state, err := git.Commands(
		git.Command("init", "."),
		git.Command("remote", "add", "mendersoftware",
			getRemoteURLGitHub(conf.githubProtocol, "mendersoftware", repoName)),
		git.Command("fetch", "mendersoftware"),
		git.Command("checkout", "mendersoftware/"+targetBranch),
		git.Command("checkout", "-b", prBranchName),
	)
	if err != nil {
		return "", state, err
	}

	if err = git.Command("cherry-pick", "-x",
		pr.GetHead().GetSHA(), "^"+pr.GetBase().GetSHA()).
		With(state).Run(); err != nil {
		if strings.Contains(err.Error(), "conflict") {
			return "", state, errorCherryPickConflict
		}
		return "", state, err
	}
	return prBranchName, state, nil
}

func cherryPickToBranch(
	log *logrus.Entry,
	comment *github.IssueCommentEvent,
	pr *github.PullRequest,
	conf *config,
	targetBranch string,
	client clientgithub.Client,
) (*github.PullRequest, error) {

	prBranchName, state, err := tryCherryPickToBranch(
		comment.GetRepo().GetName(),
		conf,
		pr,
		targetBranch,
	)
	defer state.Cleanup()
	if err != nil {
		return nil, err
	}

	if err = git.Command("push",
		"mendersoftware",
		prBranchName+":"+prBranchName).
		With(state).Run(); err != nil {
		return nil, err
	}

	newPR := &github.NewPullRequest{
		Title: github.String(fmt.Sprintf("[Cherry %s]: %s",
			targetBranch, comment.GetIssue().GetTitle())),
		Head: github.String(prBranchName),
		Base: github.String(targetBranch),
		Body: github.String(
			fmt.Sprintf("Cherry pick of PR: #%d\nFor you %s :)",
				pr.GetNumber(), comment.Sender.GetName())),
		MaintainerCanModify: github.Bool(true),
	}
	newPRRes, err := client.CreatePullRequest(
		context.Background(),
		conf.githubOrganization,
		comment.GetRepo().GetName(),
		newPR)
	if err != nil {
		return nil, fmt.Errorf("Failed to create the PR for: (%s) %v",
			comment.GetRepo().GetName(), err)
	}
	return newPRRes, nil
}

func cherryPickPR(
	log *logrus.Entry,
	comment *github.IssueCommentEvent,
	pr *github.PullRequest,
	conf *config,
	body string,
	githubClient clientgithub.Client,
) error {
	targetBranches, err := parseCherryTargetBranches(body)
	if err != nil {
		return err
	}
	conflicts := make(map[string]bool)
	errors := make(map[string]string)
	success := make(map[string]string)
	for _, targetBranch := range targetBranches {
		if newPR, err := cherryPickToBranch(
			log,
			comment,
			pr,
			conf,
			targetBranch,
			githubClient,
		); err != nil {
			if err == errorCherryPickConflict {
				conflicts[targetBranch] = true
				continue
			}
			log.Errorf("Failed to cherry pick: %s to %s, err: %s",
				comment.GetIssue().GetTitle(), targetBranch, err)
			errors[targetBranch] = err.Error()
		} else {
			success[targetBranch] = fmt.Sprintf("#%d", newPR.GetNumber())
		}
	}
	// Comment with cherry links on the PR
	commentText := `Hi :smiley_cat:
I did my very best, and this is the result of the cherry pick operation:
`
	for _, targetBranch := range targetBranches {
		if !conflicts[targetBranch] && errors[targetBranch] != "" {
			commentText = commentText +
				fmt.Sprintf("* %s :red_circle: Error: %s\n", targetBranch, errors[targetBranch])
		} else if success[targetBranch] != "" {
			commentText = commentText +
				fmt.Sprintf("* %s :heavy_check_mark: %s\n", targetBranch, success[targetBranch])
		} else {
			commentText = commentText +
				fmt.Sprintf("* %s Had merge conflicts, you will have to fix this yourself "+
					":crying_cat_face:\n", targetBranch)
		}
	}

	commentBody := github.IssueComment{
		Body: &commentText,
	}
	if err := githubClient.CreateComment(
		context.Background(),
		conf.githubOrganization,
		comment.GetRepo().GetName(),
		pr.GetNumber(),
		&commentBody,
	); err != nil {
		log.Infof("Failed to comment on the pr: %v, Error: %s", pr, err.Error())
		return err
	}
	return nil
}

func parseCherryTargetBranches(body string) ([]string, error) {
	if matches := parseCherryTargetBranchesMultiLine(body); len(matches) > 0 {
		return matches, nil
	} else if matches := parseCherryTargetBranchesSingleLine(body); len(matches) > 0 {
		return matches, nil
	}
	return nil, fmt.Errorf("No target branches found in the comment body: %s", body)
}

func parseCherryTargetBranchesMultiLine(body string) []string {
	matches := []string{}
	regex := regexp.MustCompile(` *\* *(([[:word:]]+[_\.-]?)+)`)
	for _, line := range strings.Split(body, "\n") {
		if m := regex.FindStringSubmatch(line); len(m) > 1 {
			matches = append(matches, m[1])
		}
	}
	return matches
}

func parseCherryTargetBranchesSingleLine(body string) []string {
	body = strings.TrimPrefix(body, commandCherryPickBranch)
	matches := []string{}
	regex := regexp.MustCompile(`\x60(([[:word:]]+[_\.-]?)+)\x60`)
	for _, m := range regex.FindAllStringSubmatch(body, -1) {
		if len(m) > 1 {
			matches = append(matches, m[1])
		}
	}
	return matches
}
