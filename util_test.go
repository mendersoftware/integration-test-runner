package main

import (
	"testing"

	"github.com/google/go-github/v28/github"
	"github.com/stretchr/testify/assert"
)

func TestGetRemoteURLGitHub(t *testing.T) {
	url := getRemoteURLGitHub(gitProtocolSSH, "mendersoftware", "workflows")
	assert.Equal(t, "git@github.com:/mendersoftware/workflows.git", url)

	url = getRemoteURLGitHub(gitProtocolHTTP, "mendersoftware", "workflows")
	assert.Equal(t, "https://github.com/mendersoftware/workflows", url)
}

func TestGetRemoteURLGitLab(t *testing.T) {
	url, err := getRemoteURLGitLab("mendersoftware", "workflows")
	assert.NoError(t, err)
	assert.Equal(t, "git@gitlab.com:Northern.tech/Mender/workflows", url)

	url, err = getRemoteURLGitLab("mendersoftware", "saas")
	assert.NoError(t, err)
	assert.Equal(t, "git@gitlab.com:Northern.tech/MenderSaaS/saas", url)

	url, err = getRemoteURLGitLab("unknown", "saas")
	assert.Error(t, err)
	assert.Equal(t, "", url)
}

func TestGetGitHubRepoName(t *testing.T) {
	testCases := map[string]struct {
		webhookType  string
		webhookEvent interface{}
		expected     string
		expectErr    bool
	}{
		"pull request": {
			webhookType: "pull_request",
			webhookEvent: &github.PullRequestEvent{
				Repo: &github.Repository{Name: github.String("libntech")},
			},
			expected: "libntech",
		},
		"push": {
			webhookType: "push",
			webhookEvent: &github.PushEvent{
				Repo: &github.PushEventRepository{Name: github.String("website")},
			},
			expected: "website",
		},
		"issue comment": {
			webhookType: "issue_comment",
			webhookEvent: &github.IssueCommentEvent{
				Repo: &github.Repository{Name: github.String("nt-connect")},
			},
			expected: "nt-connect",
		},
		"unknown type": {
			webhookType:  "release",
			webhookEvent: &github.PushEvent{},
			expectErr:    true,
		},
		"nil event": {
			webhookType:  "push",
			webhookEvent: nil,
			expectErr:    true,
		},
		// github.ParseWebHook returns a nil event for unhandled types, so the
		// type and the event can disagree. Unchecked assertions panic here.
		"type and event disagree": {
			webhookType:  "push",
			webhookEvent: &github.PullRequestEvent{},
			expectErr:    true,
		},
		"typed nil is not an error": {
			webhookType:  "push",
			webhookEvent: (*github.PushEvent)(nil),
			expected:     "",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			repo, err := getGitHubRepoName(tc.webhookType, tc.webhookEvent)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expected, repo)
		})
	}
}
