package main

import (
	"context"
	"errors"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v28/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	mock_github "github.com/mendersoftware/integration-test-runner/client/github/mocks"
)

func TestProcessGitHubWebhook(t *testing.T) {

	gitHubOrg := "mendersoftware"

	testCases := map[string]struct {
		webhookType  string
		webhookEvent interface{}

		repo     string
		prNumber int

		isOrganizationMember *bool
		pullRequest          *github.PullRequest
		pullRequestErr       error

		err error
	}{
		"comment updated, ignore": {
			webhookType: "issue_comment",
			webhookEvent: &github.IssueCommentEvent{
				Action: github.String("updated"),
				Repo: &github.Repository{
					Owner: &github.User{
						Login: github.String(gitHubOrg),
					},
				},
			},
		},
		"comment from non-mendersoftware user, ignore": {
			webhookType: "issue_comment",
			webhookEvent: &github.IssueCommentEvent{
				Action: github.String("created"),
				Sender: &github.User{
					Login: github.String("not-member"),
				},
				Repo: &github.Repository{
					Owner: &github.User{
						Login: github.String(gitHubOrg),
					},
				},
			},

			isOrganizationMember: github.Bool(false),
		},
		"comment from organization user, missing mention": {
			webhookType: "issue_comment",
			webhookEvent: &github.IssueCommentEvent{
				Action: github.String("created"),
				Comment: &github.IssueComment{
					Body: github.String("@friend start pipeline"),
				},
				Sender: &github.User{
					Login: github.String("not-member"),
				},
				Repo: &github.Repository{
					Owner: &github.User{
						Login: github.String(gitHubOrg),
					},
				},
			},

			isOrganizationMember: github.Bool(false),
		},
		"comment from organization user, command not recognized": {
			webhookType: "issue_comment",
			webhookEvent: &github.IssueCommentEvent{
				Action: github.String("created"),
				Comment: &github.IssueComment{
					Body: github.String("@" + githubBotName + " dummy"),
				},
				Sender: &github.User{
					Login: github.String("not-member"),
				},
				Repo: &github.Repository{
					Owner: &github.User{
						Login: github.String(gitHubOrg),
					},
				},
			},

			isOrganizationMember: github.Bool(false),
		},
		"comment from organization user, no pull request associated": {
			webhookType: "issue_comment",
			webhookEvent: &github.IssueCommentEvent{
				Action: github.String("created"),
				Comment: &github.IssueComment{
					Body: github.String("@" + githubBotName + " start pipeline"),
				},
				Sender: &github.User{
					Login: github.String("member"),
				},
				Repo: &github.Repository{
					Owner: &github.User{
						Login: github.String(gitHubOrg),
					},
				},
			},

			isOrganizationMember: github.Bool(true),
		},
		"comment from organization user, wrong pull request link": {
			webhookType: "issue_comment",
			webhookEvent: &github.IssueCommentEvent{
				Action: github.String("created"),
				Comment: &github.IssueComment{
					Body: github.String("@" + githubBotName + " start pipeline"),
				},
				Issue: &github.Issue{
					PullRequestLinks: &github.PullRequestLinks{
						URL: github.String("https://api.github.com/repos/mendersoftware/integration-test-runner/pulls/a"),
					},
				},
				Sender: &github.User{
					Login: github.String("member"),
				},
				Repo: &github.Repository{
					Owner: &github.User{
						Login: github.String(gitHubOrg),
					},
				},
			},

			isOrganizationMember: github.Bool(true),

			err: errors.New("strconv.Atoi: parsing \"a\": invalid syntax"),
		},
		"comment from organization user, error retrieving pull request": {
			webhookType: "issue_comment",
			webhookEvent: &github.IssueCommentEvent{
				Action: github.String("created"),
				Comment: &github.IssueComment{
					Body: github.String("@" + githubBotName + " start pipeline"),
				},
				Issue: &github.Issue{
					PullRequestLinks: &github.PullRequestLinks{
						URL: github.String("https://api.github.com/repos/mendersoftware/integration-test-runner/pulls/78"),
					},
				},
				Repo: &github.Repository{
					Name: github.String("integration-test-runner"),
					Owner: &github.User{
						Login: github.String(gitHubOrg),
					},
				},
				Sender: &github.User{
					Login: github.String("member"),
				},
			},

			isOrganizationMember: github.Bool(true),

			repo:     "integration-test-runner",
			prNumber: 78,

			pullRequestErr: errors.New("generic error"),
			err:            errors.New("generic error"),
		},
		"comment from organization user, start the builds": {
			webhookType: "issue_comment",
			webhookEvent: &github.IssueCommentEvent{
				Action: github.String("created"),
				Comment: &github.IssueComment{
					Body: github.String("@" + githubBotName + " start pipeline"),
				},
				Issue: &github.Issue{
					PullRequestLinks: &github.PullRequestLinks{
						URL: github.String("https://api.github.com/repos/mendersoftware/integration-test-runner/pulls/78"),
					},
				},
				Repo: &github.Repository{
					Name: github.String("integration-test-runner"),
					Owner: &github.User{
						Login: github.String(gitHubOrg),
					},
				},
				Sender: &github.User{
					Login: github.String("member"),
				},
			},

			isOrganizationMember: github.Bool(true),

			repo:     "integration-test-runner",
			prNumber: 78,

			pullRequest: &github.PullRequest{
				Base: &github.PullRequestBranch{
					Label: github.String("user:branch"),
				},
			},
		},
		"comment from organization user, start the builds 2": {
			webhookType: "issue_comment",
			webhookEvent: &github.IssueCommentEvent{
				Action: github.String("created"),
				Comment: &github.IssueComment{
					Body: github.String("@" + githubBotName + " start pipeline --pr mender/pull/16/head --pr deviceconnect/1.0.x"),
				},
				Issue: &github.Issue{
					PullRequestLinks: &github.PullRequestLinks{
						URL: github.String("https://api.github.com/repos/mendersoftware/integration-test-runner/pulls/78"),
					},
				},
				Repo: &github.Repository{
					Name: github.String("integration-test-runner"),
					Owner: &github.User{
						Login: github.String(gitHubOrg),
					},
				},
				Sender: &github.User{
					Login: github.String("member"),
				},
			},

			isOrganizationMember: github.Bool(true),

			repo:     "integration-test-runner",
			prNumber: 78,

			pullRequest: &github.PullRequest{
				Base: &github.PullRequestBranch{
					Label: github.String("user:branch"),
				},
			},
		},
		"comment from organization user, parse error in arguments": {
			webhookType: "issue_comment",
			webhookEvent: &github.IssueCommentEvent{
				Action: github.String("created"),
				Comment: &github.IssueComment{
					Body: github.String("@" + githubBotName + " start pipeline --pr mender/pull/16/head --pr deviceconnect"),
				},
				Issue: &github.Issue{
					PullRequestLinks: &github.PullRequestLinks{
						URL: github.String("https://api.github.com/repos/mendersoftware/integration-test-runner/pulls/78"),
					},
				},
				Repo: &github.Repository{
					Name: github.String("integration-test-runner"),
					Owner: &github.User{
						Login: github.String(gitHubOrg),
					},
				},
				Sender: &github.User{
					Login: github.String("member"),
				},
			},

			isOrganizationMember: github.Bool(true),

			repo:     "integration-test-runner",
			prNumber: 78,

			pullRequest: &github.PullRequest{
				Base: &github.PullRequestBranch{
					Label: github.String("user:branch"),
				},
			},
			err: errors.New("parse error near 'deviceconnect', I need, e.g.: start pipeline --pr somerepo/pull/12/head --pr somerepo/1.0.x "),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			mclient := &mock_github.Client{}
			defer mclient.AssertExpectations(t)

			if tc.isOrganizationMember != nil {
				mclient.On("IsOrganizationMember",
					mock.MatchedBy(func(ctx context.Context) bool {
						return true
					}),
					gitHubOrg,
					tc.webhookEvent.(*github.IssueCommentEvent).GetSender().GetLogin(),
				).Return(*tc.isOrganizationMember)
			}

			if tc.pullRequest != nil || tc.pullRequestErr != nil {
				mclient.On("GetPullRequest",
					mock.MatchedBy(func(ctx context.Context) bool {
						return true
					}),
					gitHubOrg,
					tc.repo,
					tc.prNumber,
				).Return(tc.pullRequest, tc.pullRequestErr)
			}

			conf := &config{
				githubProtocol:     gitProtocolHTTP,
				githubOrganization: gitHubOrg,
			}

			ctx := &gin.Context{}
			ctx.Set("delivery", "dummy")

			err := processGitHubWebhook(ctx, tc.webhookType, tc.webhookEvent, mclient, conf)
			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParsePrOptions(t *testing.T) {
	testCases := map[string]struct {
		StartPipelineComment string
		RepoToPr             map[string]string
		ParseError           error
	}{
		"start pipeline with --pr flags": {
			StartPipelineComment: "start pipeline --pr mender-connect/pull/88/head --pr deviceconnect/pull/12/head --pr mender/3.1.x",
			RepoToPr: map[string]string{
				"mender-connect": "pull/88/head",
				"deviceconnect":  "pull/12/head",
				"mender":         "3.1.x",
			},
		},
		"start pipeline with parse error in --pr flags": {
			StartPipelineComment: "start pipeline --pr mender-connect/pull/88/head --pr deviceconnect --pr mender/3.1.x",
			RepoToPr: map[string]string{
				"mender-connect": "pull/88/head",
			},
			ParseError: errors.New("parse error near 'deviceconnect', I need, e.g.: start pipeline --pr somerepo/pull/12/head --pr somerepo/1.0.x "),
		},
		"start pipeline with --pr flags and some sugar": {
			StartPipelineComment: "start pipeline --pr mender-connect/pull/88/head --pr deviceconnect/pull/12/head --pr mender/3.1.x sugar pretty please",
			RepoToPr: map[string]string{
				"mender-connect": "pull/88/head",
				"deviceconnect":  "pull/12/head",
				"mender":         "3.1.x",
			},
		},
		"start pipeline with --pr flags and some sugar with multiple spaces": {
			StartPipelineComment: "start pipeline  --pr          mender-connect/pull/88/head          --pr          deviceconnect/pull/12/head --pr mender/3. 1.x     sugar pretty please",
			RepoToPr: map[string]string{
				"mender-connect": "pull/88/head",
				"deviceconnect":  "pull/12/head",
				"mender":         "3.",
			},
		},
		"start pipeline with one --pr flag": {
			StartPipelineComment: "start pipeline --pr mender-connect/pull/88/head",
			RepoToPr: map[string]string{
				"mender-connect": "pull/88/head",
			},
		},
		"start pipeline without--pr flags": {
			StartPipelineComment: "start pipeline",
			RepoToPr:             map[string]string{},
		},
		"start pipeline incomplete --pr": {
			StartPipelineComment: "start pipeline --pr",
			RepoToPr:             map[string]string{},
		},
		"start pipeline incomplete --pr param": {
			StartPipelineComment: "start pipeline --pr some",
			RepoToPr:             map[string]string{},
		},
		"start pipeline incomplete --pr params": {
			StartPipelineComment: "start pipeline --pr --pr a --pr some",
			RepoToPr:             map[string]string{},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			actualRepoToPr, err := parsePrOptions(tc.StartPipelineComment)
			if tc.ParseError != nil {
				assert.EqualError(t, err, tc.ParseError.Error())
			} else {
				assert.Equal(t, tc.RepoToPr, actualRepoToPr)
			}
		})
	}
}
