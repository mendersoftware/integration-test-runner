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
	ref := "refs/heads/master"

	testCases := map[string]struct {
		webhookType  string
		webhookEvent interface{}

		isCommentEventProcessingEnabled  bool
		isPREventsProcessingEnabled      bool
		isCommentEventsProcessingEnabled bool

		repo     string
		prNumber int

		isOrganizationMember *bool
		pullRequest          *github.PullRequest
		pullRequestErr       error

		err           error
		createComment bool
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
			isCommentEventProcessingEnabled: true,
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

			isCommentEventProcessingEnabled: true,

			isOrganizationMember: github.Bool(false),
		},
		"comment from organization user, missing mention": {
			webhookType: "issue_comment",
			webhookEvent: &github.IssueCommentEvent{
				Action: github.String("created"),
				Comment: &github.IssueComment{
					Body: github.String("@friend start client pipeline"),
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

			isCommentEventProcessingEnabled: true,

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

			isCommentEventProcessingEnabled: true,

			isOrganizationMember: github.Bool(false),
		},
		"comment from organization user, no pull request associated": {
			webhookType: "issue_comment",
			webhookEvent: &github.IssueCommentEvent{
				Action: github.String("created"),
				Comment: &github.IssueComment{
					Body: github.String("@" + githubBotName + " start client pipeline"),
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

			isCommentEventProcessingEnabled: true,

			isOrganizationMember: github.Bool(true),
		},
		"comment from organization user, wrong pull request link": {
			webhookType: "issue_comment",
			webhookEvent: &github.IssueCommentEvent{
				Action: github.String("created"),
				Comment: &github.IssueComment{
					Body: github.String("@" + githubBotName + " start client pipeline"),
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

			isCommentEventProcessingEnabled: true,

			isOrganizationMember: github.Bool(true),

			err: errors.New("strconv.Atoi: parsing \"a\": invalid syntax"),
		},
		"comment from organization user, error retrieving pull request": {
			webhookType: "issue_comment",
			webhookEvent: &github.IssueCommentEvent{
				Action: github.String("created"),
				Comment: &github.IssueComment{
					Body: github.String("@" + githubBotName + " start client pipeline"),
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

			isCommentEventProcessingEnabled: true,

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
					Body: github.String("@" + githubBotName + " start client pipeline"),
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

			isCommentEventProcessingEnabled: true,

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
					Body: github.String("@" + githubBotName + " start client pipeline --pr mender/pull/16/head --pr deviceconnect/1.0.x"),
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

			isCommentEventProcessingEnabled: true,

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
					Body: github.String("@" + githubBotName + " start client pipeline --pr mender/pull/16/head --pr deviceconnect"),
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

			isCommentEventProcessingEnabled: true,

			isOrganizationMember: github.Bool(true),

			repo:     "integration-test-runner",
			prNumber: 78,

			pullRequest: &github.PullRequest{
				Base: &github.PullRequestBranch{
					Label: github.String("user:branch"),
				},
				Number: github.Int(78),
			},
			err:           errors.New("parse error near 'deviceconnect', I need, e.g.: start client pipeline --pr somerepo/pull/12/head --pr somerepo/1.0.x "),
			createComment: true,
		},
		"comment created, feature disabled": {
			webhookType: "issue_comment",
			webhookEvent: &github.IssueCommentEvent{
				Action: github.String("created"),
			},

			isCommentEventProcessingEnabled: false,
		},
		"pull request created, feature enabled": {
			webhookType: "pull_request",

			webhookEvent: &github.PullRequestEvent{
				PullRequest: &github.PullRequest{
					Merged: github.Bool(false),
				},
				Repo: &github.Repository{
					Name: github.String("integration-test-runner"),
					Owner: &github.User{
						Name: github.String(gitHubOrg),
					},
				},
				Number: github.Int(6),
			},

			isOrganizationMember: github.Bool(true),

			isPREventsProcessingEnabled: true,
		},
		"pull request created, feature disabled": {
			webhookType: "pull_request",

			webhookEvent: &github.PullRequestEvent{
				PullRequest: &github.PullRequest{
					Merged: github.Bool(false),
				},
				Repo: &github.Repository{
					Name: github.String("integration-test-runner"),
					Owner: &github.User{
						Name: github.String(gitHubOrg),
					},
				},
				Number: github.Int(6),
			},

			isPREventsProcessingEnabled: false,
		},
		"push event, feature enabled": {
			webhookType: "push",

			webhookEvent: &github.PushEvent{
				Repo: &github.PushEventRepository{
					Name:         github.String("integration-test-runner"),
					Organization: github.String(gitHubOrg),
				},
				Ref: &ref,
			},

			isCommentEventsProcessingEnabled: true,
		},
		"push event, feature disabled": {
			webhookType: "push",

			webhookEvent: &github.PushEvent{
				Repo: &github.PushEventRepository{
					Name:         github.String("integration-test-runner"),
					Organization: github.String(gitHubOrg),
				},
			},

			isCommentEventsProcessingEnabled: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			oldGithubClient := githubClient
			defer func() {
				githubClient = oldGithubClient
			}()

			mclient := &mock_github.Client{}
			defer mclient.AssertExpectations(t)

			githubClient = mclient

			if tc.isOrganizationMember != nil {
				var user string
				org := gitHubOrg
				if tc.isCommentEventProcessingEnabled {
					user = tc.webhookEvent.(*github.IssueCommentEvent).GetSender().GetLogin()
				} else if tc.isPREventsProcessingEnabled {
					user = tc.webhookEvent.(*github.PullRequestEvent).GetSender().GetLogin()
					org = ""
				} else if tc.isCommentEventsProcessingEnabled {
					user = tc.webhookEvent.(*github.PushEvent).GetRepo().GetOrganization()
				}
				mclient.On("IsOrganizationMember",
					mock.MatchedBy(func(ctx context.Context) bool {
						return true
					}),
					org,
					user,
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

			if tc.createComment {
				mclient.On("CreateComment",
					mock.MatchedBy(func(ctx context.Context) bool {
						return true
					}),
					gitHubOrg,
					tc.repo,
					tc.prNumber,
					mock.AnythingOfType("*github.IssueComment"),
				).Return(nil)
			}

			conf := &config{
				githubProtocol:         gitProtocolHTTP,
				githubOrganization:     gitHubOrg,
				isProcessCommentEvents: tc.isCommentEventProcessingEnabled,
				isProcessPREvents:      tc.isPREventsProcessingEnabled,
				isProcessPushEvents:    tc.isCommentEventsProcessingEnabled,
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

func TestParseBuildOptions(t *testing.T) {
	testCases := map[string]struct {
		StartPipelineComment string
		BuildOptions         *BuildOptions
		ParseError           error
	}{
		"start client pipeline with --pr flags": {
			StartPipelineComment: "start client pipeline --pr mender-connect/pull/88/head --pr deviceconnect/pull/12/head --pr mender/3.1.x",
			BuildOptions: &BuildOptions{
				PullRequests: map[string]string{
					"mender-connect": "pull/88/head",
					"deviceconnect":  "pull/12/head",
					"mender":         "3.1.x",
				},
			},
		},
		"start client pipeline with --pr and --fast flags": {
			StartPipelineComment: "start client pipeline --pr mender-connect/pull/88/head --pr deviceconnect/pull/12/head --pr mender/3.1.x --fast",
			BuildOptions: &BuildOptions{
				PullRequests: map[string]string{
					"mender-connect": "pull/88/head",
					"deviceconnect":  "pull/12/head",
					"mender":         "3.1.x",
				},
				Fast: true,
			},
		},
		"start client pipeline with parse error in --pr flags": {
			StartPipelineComment: "start client pipeline --pr mender-connect/pull/88/head --pr deviceconnect --pr mender/3.1.x",
			BuildOptions: &BuildOptions{
				PullRequests: map[string]string{
					"mender-connect": "pull/88/head",
				},
			},
			ParseError: errors.New("parse error near 'deviceconnect', I need, e.g.: start client pipeline --pr somerepo/pull/12/head --pr somerepo/1.0.x "),
		},
		"start client pipeline with --pr flags and some sugar": {
			StartPipelineComment: "start client pipeline --pr mender-connect/pull/88/head --pr deviceconnect/pull/12/head --pr mender/3.1.x sugar pretty please",
			BuildOptions: &BuildOptions{
				PullRequests: map[string]string{
					"mender-connect": "pull/88/head",
					"deviceconnect":  "pull/12/head",
					"mender":         "3.1.x",
				},
			},
		},
		"start client pipeline with --pr flags (new syntax)": {
			StartPipelineComment: "start client pipeline --pr mender-connect/pull/88 --pr deviceconnect/pull/12 --pr mender/3.1.x --pr deviceauth/feature-branch",
			BuildOptions: &BuildOptions{
				PullRequests: map[string]string{
					"mender-connect": "pull/88/head",
					"deviceconnect":  "pull/12/head",
					"mender":         "3.1.x",
					"deviceauth":     "feature-branch",
				},
			},
		},
		"start client pipeline with --pr flags (syntax without 'pull' and 'head')": {
			StartPipelineComment: "start client pipeline --pr mender-connect/88 --pr deviceconnect/12 --pr mender/3.1.x",
			BuildOptions: &BuildOptions{
				PullRequests: map[string]string{
					"mender-connect": "pull/88/head",
					"deviceconnect":  "pull/12/head",
					"mender":         "3.1.x",
				},
			},
		},
		"start client pipeline with --pr flags and some sugar with multiple spaces": {
			StartPipelineComment: "start client pipeline  --pr          mender-connect/pull/88/head          --pr          deviceconnect/pull/12/head --pr mender/3. 1.x     sugar pretty please",
			BuildOptions: &BuildOptions{
				PullRequests: map[string]string{
					"mender-connect": "pull/88/head",
					"deviceconnect":  "pull/12/head",
					"mender":         "3.",
				},
			},
		},
		"start client pipeline with one --pr flag": {
			StartPipelineComment: "start client pipeline --pr mender-connect/pull/88/head",
			BuildOptions: &BuildOptions{
				PullRequests: map[string]string{
					"mender-connect": "pull/88/head",
				},
			},
		},
		"start client pipeline without--pr flags": {
			StartPipelineComment: "start client pipeline",
			BuildOptions: &BuildOptions{
				PullRequests: map[string]string{},
			},
		},
		"start client pipeline incomplete --pr": {
			StartPipelineComment: "start client pipeline --pr",
			BuildOptions: &BuildOptions{
				PullRequests: map[string]string{},
			},
		},
		"start client pipeline incomplete --pr param": {
			StartPipelineComment: "start client pipeline --pr some",
			BuildOptions: &BuildOptions{
				PullRequests: map[string]string{},
			},
			ParseError: errors.New("parse error near 'some', I need, e.g.: start client pipeline --pr somerepo/pull/12/head --pr somerepo/1.0.x "),
		},
		"start client pipeline incomplete --pr params": {
			StartPipelineComment: "start client pipeline --pr --pr a --pr some",
			BuildOptions: &BuildOptions{
				PullRequests: map[string]string{},
			},
			ParseError: errors.New("parse error near 'some', I need, e.g.: start client pipeline --pr somerepo/pull/12/head --pr somerepo/1.0.x "),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			actualRepoToPr, err := parseBuildOptions(tc.StartPipelineComment)
			if tc.ParseError != nil {
				assert.EqualError(t, err, tc.ParseError.Error())
			} else {
				assert.Equal(t, tc.BuildOptions, actualRepoToPr)
			}
		})
	}
}
