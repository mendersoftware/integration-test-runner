package main

import (
	"testing"

	"github.com/google/go-github/v28/github"
	"github.com/stretchr/testify/assert"
)

func testPullRequest(author, headRepo, headRef string, labels ...string) *github.PullRequest {
	prLabels := make([]*github.Label, 0, len(labels))
	for _, name := range labels {
		prLabels = append(prLabels, &github.Label{Name: github.String(name)})
	}
	return &github.PullRequest{
		User:   &github.User{Login: github.String(author)},
		Labels: prLabels,
		Head: &github.PullRequestBranch{
			Ref:  github.String(headRef),
			Repo: &github.Repository{FullName: github.String(headRepo)},
		},
		Base: &github.PullRequestBranch{
			Ref:  github.String("main"),
			Repo: &github.Repository{FullName: github.String("mendersoftware/mender-mcu")},
		},
	}
}

func TestIsReleasePleasePR(t *testing.T) {
	const repo = "mendersoftware/mender-mcu"

	tests := map[string]struct {
		pr       *github.PullRequest
		expected bool
	}{
		"release-please pull request": {
			testPullRequest(githubBotName, repo,
				"release-please--branches--main", releasePleaseLabel),
			true,
		},
		"monorepo component branch": {
			testPullRequest(githubBotName, repo,
				"release-please--branches--main--components--modules/aws", releasePleaseLabel),
			true,
		},
		"label among several": {
			testPullRequest(githubBotName, repo,
				"release-please--branches--main", "aws", releasePleaseLabel, "dependencies"),
			true,
		},
		"human author on a release-please branch": {
			testPullRequest("attacker", repo,
				"release-please--branches--main", releasePleaseLabel),
			false,
		},
		"fork head repository": {
			testPullRequest(githubBotName, "attacker/mender-mcu",
				"release-please--branches--main", releasePleaseLabel),
			false,
		},
		"unrelated branch": {
			testPullRequest(githubBotName, repo, "fix-something", releasePleaseLabel),
			false,
		},
		"branch prefix as a substring only": {
			testPullRequest(githubBotName, repo,
				"not-release-please--branches--main", releasePleaseLabel),
			false,
		},
		"missing label": {
			testPullRequest(githubBotName, repo, "release-please--branches--main"),
			false,
		},
		"wrong label": {
			testPullRequest(githubBotName, repo,
				"release-please--branches--main", "autorelease: tagged"),
			false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			skip, reason := isReleasePleasePR(tc.pr, githubBotName)
			assert.Equal(t, tc.expected, skip)
			assert.NotEmpty(t, reason)
		})
	}
}

func TestIsChangelogOnlyDiff(t *testing.T) {
	files := func(names ...string) []github.CommitFile {
		out := make([]github.CommitFile, 0, len(names))
		for _, name := range names {
			out = append(out, github.CommitFile{Filename: github.String(name)})
		}
		return out
	}

	tests := map[string]struct {
		files    []github.CommitFile
		expected bool
	}{
		"changelog and manifest": {
			files("CHANGELOG.md", ".release-please-manifest.json"),
			true,
		},
		"nested package changelogs": {
			files("modules/aws/CHANGELOG.md", "manifests/CHANGELOG.md",
				".release-please-manifest.json"),
			true,
		},
		"suffixed changelog": {
			files("CHANGELOG-saas.md"),
			true,
		},
		"node package manifest": {
			files("packages/common/CHANGELOG.md", "packages/common/package.json"),
			false,
		},
		"helm chart": {
			files("CHANGELOG.md", "mender/Chart.yaml"),
			false,
		},
		"source file": {
			files("CHANGELOG.md", "src/mender.c"),
			false,
		},
		"changelog with an executable suffix": {
			files("CHANGELOG.md.sh"),
			false,
		},
		"lowercase changelog": {
			files("changelog.md"),
			false,
		},
		"nested manifest": {
			files("modules/aws/.release-please-manifest.json"),
			false,
		},
		"empty diff": {
			nil,
			false,
		},
		"rename from a source file": {
			[]github.CommitFile{{
				Filename:         github.String("CHANGELOG.md"),
				PreviousFilename: github.String("main.go"),
			}},
			false,
		},
		"rename between changelogs": {
			[]github.CommitFile{{
				Filename:         github.String("CHANGELOG.md"),
				PreviousFilename: github.String("CHANGELOG-old.md"),
			}},
			true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			skip, reason := isChangelogOnlyDiff(tc.files)
			assert.Equal(t, tc.expected, skip)
			assert.NotEmpty(t, reason)
		})
	}
}
