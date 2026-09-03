package main

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	mock_github "github.com/mendersoftware/integration-test-runner/client/github/mocks"
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

func testPullRequestEvent(sender string, pr *github.PullRequest) *github.PullRequestEvent {
	return &github.PullRequestEvent{
		Number:      github.Int(42),
		PullRequest: pr,
		Sender:      &github.User{Login: github.String(sender)},
		Repo:        &github.Repository{Name: github.String("mender-mcu")},
	}
}

func TestSkipPipelineForReleasePR(t *testing.T) {
	const (
		org  = "mendersoftware"
		repo = "mender-mcu"
		head = "0795080615c478041ac2032e97fcc5cdb4762789"
	)
	releasePR := func() *github.PullRequest {
		pr := testPullRequest(githubBotName, org+"/"+repo,
			"release-please--branches--main", releasePleaseLabel)
		pr.Head.SHA = github.String(head)
		pr.Base.SHA = github.String("aaaabbbbccccddddeeeeffff0000111122223333")
		return pr
	}
	changelogOnly := &github.CommitsComparison{
		Files: []github.CommitFile{{Filename: github.String("CHANGELOG.md")}},
	}

	tests := map[string]struct {
		event    *github.PullRequestEvent
		setup    func(*mock_github.Client)
		expected bool
	}{
		"skips the pipeline and reports the check": {
			event: testPullRequestEvent(githubBotName, releasePR()),
			setup: func(c *mock_github.Client) {
				c.On("CompareCommits", mock.Anything, org, repo,
					mock.AnythingOfType("string"), head).Return(changelogOnly, nil)
				c.On("CreateStatus", mock.Anything, org, repo, head,
					mock.MatchedBy(func(s *github.RepoStatus) bool {
						return s.GetContext() == gitLabStatusContext &&
							s.GetState() == "success"
					})).Return(nil)
			},
			expected: true,
		},
		"human pull request is left alone": {
			event: testPullRequestEvent("someone",
				testPullRequest("someone", org+"/"+repo, "fix-something")),
			setup:    func(c *mock_github.Client) {},
			expected: false,
		},
		"bot as sender does not make it a release pull request": {
			event: testPullRequestEvent(githubBotName,
				testPullRequest("someone", org+"/"+repo,
					"release-please--branches--main", releasePleaseLabel)),
			setup:    func(c *mock_github.Client) {},
			expected: false,
		},
		"comparison failure runs the pipeline": {
			event: testPullRequestEvent(githubBotName, releasePR()),
			setup: func(c *mock_github.Client) {
				c.On("CompareCommits", mock.Anything, org, repo,
					mock.AnythingOfType("string"), head).
					Return(nil, errors.New("boom"))
			},
			expected: false,
		},
		"unexpected file runs the pipeline": {
			event: testPullRequestEvent(githubBotName, releasePR()),
			setup: func(c *mock_github.Client) {
				c.On("CompareCommits", mock.Anything, org, repo,
					mock.AnythingOfType("string"), head).
					Return(&github.CommitsComparison{
						Files: []github.CommitFile{
							{Filename: github.String("CHANGELOG.md")},
							{Filename: github.String("src/mender.c")},
						},
					}, nil)
			},
			expected: false,
		},
		"status failure runs the pipeline": {
			event: testPullRequestEvent(githubBotName, releasePR()),
			setup: func(c *mock_github.Client) {
				c.On("CompareCommits", mock.Anything, org, repo,
					mock.AnythingOfType("string"), head).Return(changelogOnly, nil)
				c.On("CreateStatus", mock.Anything, org, repo, head,
					mock.Anything).Return(errors.New("403 forbidden"))
			},
			expected: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := &mock_github.Client{}
			tc.setup(client)
			log := logrus.NewEntry(logrus.StandardLogger())

			skipped := skipPipelineForReleasePR(
				context.Background(), log, client, tc.event, org)

			assert.Equal(t, tc.expected, skipped)
			client.AssertExpectations(t)
		})
	}
}
