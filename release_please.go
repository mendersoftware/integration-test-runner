package main

import (
	"path"
	"strings"

	"github.com/google/go-github/v28/github"
)

const (
	releasePleaseBranchPrefix = "release-please--branches--"
	releasePleaseLabel        = "autorelease: pending"
	releasePleaseManifest     = ".release-please-manifest.json"
	releasePleaseChangelog    = "CHANGELOG*.md"
)

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
