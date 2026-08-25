package main

import (
	"context"
	"path"
	"strings"

	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"

	clientgithub "github.com/mendersoftware/integration-test-runner/client/github"
)

const (
	releasePleaseBranchPrefix = "release-please--branches--"
	releasePleaseLabel        = "autorelease: pending"
	releasePleaseManifest     = ".release-please-manifest.json"
	releasePleaseChangelog    = "CHANGELOG*.md"
	gitLabStatusContext       = "ci/gitlab"
)

func skipPipelineForReleasePR(
	ctx context.Context,
	log *logrus.Entry,
	githubClient clientgithub.Client,
	pr *github.PullRequestEvent,
	org string,
) bool {
	req := pr.GetPullRequest()
	if ok, reason := isReleasePleasePR(req, githubBotName); !ok {
		if strings.HasPrefix(req.GetHead().GetRef(), releasePleaseBranchPrefix) {
			log.Infof("PR %d is on a release-please branch but needs a pipeline: %s",
				pr.GetNumber(), reason)
		}
		return false
	}

	repo := pr.GetRepo().GetName()
	head := req.GetHead().GetSHA()
	comparison, err := githubClient.CompareCommits(
		ctx, org, repo, req.GetBase().GetSHA(), head,
	)
	if err != nil {
		log.Errorf("CompareCommits failed for PR %d, running the pipeline: %s",
			pr.GetNumber(), err)
		return false
	}

	ok, reason := isChangelogOnlyDiff(comparison.Files)
	if !ok {
		log.Infof("PR %d at %s needs a pipeline: %s", pr.GetNumber(), head, reason)
		return false
	}

	status := &github.RepoStatus{
		State:       github.String("success"),
		Context:     github.String(gitLabStatusContext),
		Description: github.String("Skipped: release-please PR, " + reason),
	}
	if err := githubClient.CreateStatus(ctx, org, repo, head, status); err != nil {
		log.Errorf("CreateStatus failed for PR %d at %s, running the pipeline: %s",
			pr.GetNumber(), head, err)
		return false
	}

	log.Infof("PR %d at %s: skipped the pipeline, %s, files=%v",
		pr.GetNumber(), head, reason, changedFileNames(comparison.Files))
	return true
}

func changedFileNames(files []github.CommitFile) []string {
	names := make([]string, 0, len(files))
	for _, file := range files {
		names = append(names, file.GetFilename())
	}
	return names
}

func isReleasePleasePR(pr *github.PullRequest, botName string) (bool, string) {
	if pr.GetUser().GetLogin() != botName {
		return false, "author is not " + botName
	}
	if pr.GetHead().GetRepo().GetFullName() != pr.GetBase().GetRepo().GetFullName() {
		return false, "head is a fork"
	}
	if !strings.HasPrefix(pr.GetHead().GetRef(), releasePleaseBranchPrefix) {
		return false, "head branch is not a release-please branch"
	}
	if !hasLabel(pr, releasePleaseLabel) {
		return false, "missing the " + releasePleaseLabel + " label"
	}
	return true, "release-please pull request"
}

func isChangelogOnlyDiff(files []github.CommitFile) (bool, string) {
	if len(files) == 0 {
		return false, "empty diff"
	}
	for _, file := range files {
		for _, name := range []string{file.GetFilename(), file.GetPreviousFilename()} {
			if name == "" {
				continue
			}
			if !isChangelogPath(name) {
				return false, "diff touches " + name
			}
		}
	}
	return true, "changelog only"
}

func isChangelogPath(name string) bool {
	if name == releasePleaseManifest {
		return true
	}
	matched, err := path.Match(releasePleaseChangelog, path.Base(name))
	return err == nil && matched
}

func hasLabel(pr *github.PullRequest, name string) bool {
	for _, label := range pr.Labels {
		if label.GetName() == name {
			return true
		}
	}
	return false
}
