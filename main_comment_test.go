package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	mock_github "github.com/mendersoftware/integration-test-runner/client/github/mocks"
	mock_gitlab "github.com/mendersoftware/integration-test-runner/client/gitlab/mocks"
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

		err                    error
		createComment          bool
		createStatus           bool
		pipelineStatusContexts []string
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
		"comment from organization user, print fast pr stats": {
			webhookType: "issue_comment",
			webhookEvent: &github.IssueCommentEvent{
				Action: github.String("created"),
				Comment: &github.IssueComment{
					Body: github.String("@" + githubBotName + " print fast pr stats"),
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
				Number: github.Int(78),
				Base: &github.PullRequestBranch{
					Label: github.String("user:branch"),
				},
			},
			createComment: true,
		},
		"comment from organization user, skip pipeline": {
			webhookType: "issue_comment",
			webhookEvent: &github.IssueCommentEvent{
				Action: github.String("created"),
				Comment: &github.IssueComment{
					Body: github.String("@" + githubBotName + " skip pipeline"),
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
			isOrganizationMember:            github.Bool(true),

			repo:     "integration-test-runner",
			prNumber: 78,

			pullRequest: &github.PullRequest{
				Number: github.Int(78),
				Head: &github.PullRequestBranch{
					SHA: github.String("abc123"),
				},
			},

			pipelineStatusContexts: []string{"ci/mender-qa", "ci/integration"},
			createStatus:           true,
			createComment:          true,
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

			if tc.createStatus {
				mclient.On("CreateStatus",
					mock.MatchedBy(func(ctx context.Context) bool {
						return true
					}),
					gitHubOrg,
					tc.repo,
					mock.AnythingOfType("string"),
					mock.AnythingOfType("*github.RepoStatus"),
				).Return(nil).Times(len(tc.pipelineStatusContexts))
			}

			// Mock ListPullRequests for PR stats command
			if tc.webhookType == "issue_comment" {
				event := tc.webhookEvent.(*github.IssueCommentEvent)
				if event.Comment != nil && (strings.Contains(event.Comment.GetBody(), commandPrintPRStats) ||
					strings.Contains(event.Comment.GetBody(), commandPrintFullPRStats)) {
					mclient.On("ListPullRequests", mock.Anything, gitHubOrg, mock.Anything, mock.Anything).
						Return([]*github.PullRequest{}, nil).Maybe()
				}
			}

			conf := &config{
				githubProtocol:         gitProtocolHTTP,
				githubOrganization:     gitHubOrg,
				isProcessCommentEvents: tc.isCommentEventProcessingEnabled,
				isProcessPREvents:      tc.isPREventsProcessingEnabled,
				isProcessPushEvents:    tc.isCommentEventsProcessingEnabled,
				pipelineStatusContexts: tc.pipelineStatusContexts,
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
		"start client pipeline with --release": {
			StartPipelineComment: "start client pipeline --release 6.0.x",
			BuildOptions: &BuildOptions{
				PullRequests: map[string]string{},
				Releases:     []string{"6.0.x"},
			},
		},
		"start client pipeline with multiple --release": {
			StartPipelineComment: "start client pipeline --release 6.0.x --release 6.1.x",
			BuildOptions: &BuildOptions{
				PullRequests: map[string]string{},
				Releases:     []string{"6.0.x", "6.1.x"},
			},
		},
		"start client pipeline with --release and --pr": {
			StartPipelineComment: "start client pipeline --release 6.0.x --pr mender/pull/123/head",
			BuildOptions: &BuildOptions{
				PullRequests: map[string]string{
					"mender": "pull/123/head",
				},
				Releases: []string{"6.0.x"},
			},
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

func TestSyncProtectedBranchWithClientProtectsBeforePush(t *testing.T) {
	callOrder := []string{}

	pr := &github.PullRequestEvent{
		Number: github.Int(42),
		Repo: &github.Repository{
			Name: github.String("mender"),
		},
		PullRequest: &github.PullRequest{
			Head: &github.PullRequestBranch{
				SHA: github.String("abc123"),
			},
		},
		Organization: &github.Organization{Login: github.String("mendersoftware")},
	}
	conf := &config{}
	pipelinePath := "Northern.tech/Mender/mender"
	branchName := "pr_42_protected"

	gitlabClient := mock_gitlab.NewClient(t)

	gitlabClient.On("ProtectRepositoryBranches", pipelinePath, mock.MatchedBy(func(opts *gitlab.ProtectRepositoryBranchesOptions) bool {
		return opts.AllowForcePush != nil && *opts.AllowForcePush == true
	})).Run(func(args mock.Arguments) {
		callOrder = append(callOrder, "protect:allow-force")
	}).Return(&gitlab.ProtectedBranch{}, nil).Once()

	syncer := func(branchName string, log *logrus.Entry, pr *github.PullRequestEvent, conf *config) error {
		callOrder = append(callOrder, "push")
		return nil
	}

	log := logrus.WithField("test", true)
	name, err := syncProtectedBranchWithClient(log, pr, conf, pipelinePath, gitlabClient, syncer)

	assert.NoError(t, err)
	assert.Equal(t, branchName, name)
	assert.Equal(t, []string{"protect:allow-force", "push"}, callOrder)
}

func TestSyncProtectedBranchWithClientAlreadyProtected409(t *testing.T) {
	pr := &github.PullRequestEvent{
		Number: github.Int(42),
		Repo: &github.Repository{
			Name: github.String("mender"),
		},
		PullRequest: &github.PullRequest{
			Head: &github.PullRequestBranch{
				SHA: github.String("abc123"),
			},
		},
		Organization: &github.Organization{Login: github.String("mendersoftware")},
	}
	conf := &config{}
	pipelinePath := "Northern.tech/Mender/mender"

	gitlabClient := mock_gitlab.NewClient(t)

	gitlabClient.On("ProtectRepositoryBranches", pipelinePath, mock.Anything).
		Return(nil, &gitlab.ErrorResponse{
			Response: &http.Response{StatusCode: 409},
		}).Once()

	pushed := false
	syncer := func(branchName string, log *logrus.Entry, pr *github.PullRequestEvent, conf *config) error {
		pushed = true
		return nil
	}

	log := logrus.WithField("test", true)
	_, err := syncProtectedBranchWithClient(log, pr, conf, pipelinePath, gitlabClient, syncer)

	assert.NoError(t, err)
	assert.True(t, pushed, "syncer should be called even when protect returns 409")
}

func TestSyncProtectedBranchWithClientPushFailurePropagated(t *testing.T) {
	pr := &github.PullRequestEvent{
		Number: github.Int(7),
		Repo: &github.Repository{
			Name: github.String("mender"),
		},
		PullRequest: &github.PullRequest{
			Head: &github.PullRequestBranch{
				SHA: github.String("def456"),
			},
		},
		Organization: &github.Organization{Login: github.String("mendersoftware")},
	}
	conf := &config{}
	pipelinePath := "Northern.tech/Mender/mender"

	gitlabClient := mock_gitlab.NewClient(t)

	gitlabClient.On("ProtectRepositoryBranches", pipelinePath, mock.MatchedBy(func(opts *gitlab.ProtectRepositoryBranchesOptions) bool {
		return opts.AllowForcePush != nil && *opts.AllowForcePush == true
	})).Return(&gitlab.ProtectedBranch{}, nil).Once()

	pushErr := errors.New("git push failed")
	syncer := func(branchName string, log *logrus.Entry, pr *github.PullRequestEvent, conf *config) error {
		return pushErr
	}

	log := logrus.WithField("test", true)
	_, err := syncProtectedBranchWithClient(log, pr, conf, pipelinePath, gitlabClient, syncer)

	assert.ErrorContains(t, err, "git push failed")
}
