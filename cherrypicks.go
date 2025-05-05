package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"

	"slices"

	clientgithub "github.com/mendersoftware/integration-test-runner/client/github"
	"github.com/mendersoftware/integration-test-runner/git"
)

var versionsUrl = "https://docs.mender.io/releases/versions.json"

var errorCherryPickConflict = errors.New("Cherry pick had conflicts")

type versions struct {
	Releases map[string]map[string]interface{} `json:"releases"`
	Lts      []string                          `json:",omitempty"`
}

// Returns the supported LTS versions, as well as the latest release if it is
// not LTS.
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
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	v := versions{}
	err = json.Unmarshal(body, &v)
	if err != nil {
		return nil, err
	}
	if len(v.Lts) == 0 {
		return nil, errors.New("getLatestReleaseFromApi: lts version list is empty")
	}
	for idx, val := range v.Lts {
		v.Lts[idx] = val + ".x"
	}
	allReleases := []string{}
	for key := range v.Releases {
		allReleases = append(allReleases, key+".x")
	}
	// Only add to the list if the latest patch != latest LTS
	sort.Sort(sort.Reverse(sort.StringSlice(allReleases)))
	if allReleases[0] != v.Lts[0] {
		return append([]string{allReleases[0]}, v.Lts...), nil
	}
	return v.Lts, nil
}

func getReleaseBranchesForCherryPick(
	log *logrus.Entry,
	pr *github.PullRequestEvent,
	conf *config,
	state *git.State,
) ([]string, error) {

	releaseBranches := []string{}

	// fetch all the branches
	err := git.Command("fetch", "github").With(state).Run()
	if err != nil {
		return releaseBranches, err
	}

	// get list of release versions
	versions, err := getLatestReleaseFromApi(versionsUrl)
	if err != nil {
		return releaseBranches, err
	}

	repo := pr.GetRepo().GetName()
	for _, version := range versions {
		releaseBranch, err := getServiceRevisionFromIntegration(repo, "origin/"+version, conf)
		if err != nil {
			return releaseBranches, err
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

	return releaseBranches, nil
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

	// ignore PRs if they don't target the master or main branch
	baseRef := pr.GetPullRequest().GetBase().GetRef()
	if baseRef != "master" && baseRef != "main" {
		log.Infof("Ignoring cherry-pick suggestions for base ref: %s", baseRef)
		return nil
	}

	repo := pr.GetRepo().GetName()

	var ltsRepo bool
	for _, watchRepo := range ltsRepositories {
		if watchRepo == repo {
			ltsRepo = true
			break
		}
	}

	if !ltsRepo {
		log.Infof("Ignoring non-LTS repository: %s", repo)
		return nil
	}

	// initialize the git work area
	repoURL := getRemoteURLGitHub(conf.githubProtocol, conf.githubOrganization, repo)
	prNumber := strconv.Itoa(pr.GetNumber())
	prBranchName := "pr_" + prNumber
	state, err := git.Commands(
		git.Command("init", "."),
		git.Command("remote", "add", "github", repoURL),
		git.Command("fetch", "github", baseRef+":local"),
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

	var releaseBranches []string
	if slices.Contains(clientRepositories, repo) {
		releaseBranches, err = getReleaseBranchesForCherryPick(log, pr, conf, state)
		if err != nil {
			return err
		}
	}

	var commentBody string
	if len(releaseBranches) == 0 {
		// No suggestions for the client repo or not a client repo: drop a generic message
		// nolint:lll
		commentBody = `
Hello :smiley_cat: This PR contains changelog entries. Please, verify the need of backporting it to the supported release branches.
`
	} else {
		// nolint:lll
		tmplString := `
Hello :smiley_cat: This PR contains changelog entries. Please, verify the need of backporting it to the following release branches:
{{.ReleaseBranches}}
`
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

		commentBody = buf.String()

	}

	// Comment with a pipeline-link on the PR
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

	if err = git.Command("cherry-pick", "-x", "--allow-empty",
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
