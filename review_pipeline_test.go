package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	githubmocks "github.com/mendersoftware/integration-test-runner/client/github/mocks"
	gitlabmocks "github.com/mendersoftware/integration-test-runner/client/gitlab/mocks"
)

func TestGetReviewAppURL(t *testing.T) {
	assert.Equal(t, "https://os-pr-42.staging.hosted.mender.io/",
		getReviewAppURL("staging.hosted.mender.io", "os", 42))
	assert.Equal(t, "https://ent-pr-1.staging.hosted.mender.io/",
		getReviewAppURL("staging.hosted.mender.io", "ent", 1))
}

func TestParseReviewAppEnterprise(t *testing.T) {
	testCases := map[string]struct {
		comment  string
		expected bool
	}{
		"no enterprise flag": {
			comment:  "@bot start review app",
			expected: false,
		},
		"enterprise flag": {
			comment:  "@bot start review app enterprise",
			expected: true,
		},
		"other text after command": {
			comment:  "@bot start review app something",
			expected: false,
		},
		"command not found": {
			comment:  "@bot something else",
			expected: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := parseReviewAppEnterprise(tc.comment)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseReviewTestEnvironment(t *testing.T) {
	testCases := map[string]struct {
		comment  string
		expected string
	}{
		"no environment specified": {
			comment:  "@bot start review tests",
			expected: "os",
		},
		"enterprise environment": {
			comment:  "@bot start review tests enterprise",
			expected: "enterprise",
		},
		"os environment": {
			comment:  "@bot start review tests os",
			expected: "os",
		},
		"invalid environment defaults to os": {
			comment:  "@bot start review tests invalid",
			expected: "os",
		},
		"command not found defaults to os": {
			comment:  "@bot something else",
			expected: "os",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := parseReviewTestEnvironment(tc.comment)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFindAndPlayJob(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	projectPath := "Northern.tech/Mender/mender-server"
	ref := "pr_42"
	jobName := "review:deploy"

	requestedByKey := "REVIEW_REQUESTED_BY"
	sender := "testuser"
	jobVars := []*gitlab.JobVariableOptions{
		{Key: &requestedByKey, Value: &sender},
	}

	testCases := map[string]struct {
		setupMock   func(*gitlabmocks.Client)
		expectedErr string
	}{
		"no pipelines found": {
			setupMock: func(c *gitlabmocks.Client) {
				c.On("ListProjectPipelines", projectPath, mock.Anything).
					Return([]*gitlab.PipelineInfo{}, nil)
			},
			expectedErr: "no pipelines found for ref pr_42 in project Northern.tech/Mender/mender-server",
		},
		"job not found": {
			setupMock: func(c *gitlabmocks.Client) {
				c.On("ListProjectPipelines", projectPath, mock.Anything).
					Return([]*gitlab.PipelineInfo{{ID: 100}}, nil)
				c.On("ListPipelineJobs", projectPath, int64(100), mock.Anything).
					Return([]*gitlab.Job{
						{ID: 1, Name: "build:backend:docker", Status: "success"},
					}, nil)
			},
			expectedErr: `job "review:deploy" not found in pipeline 100`,
		},
		"job not manual": {
			setupMock: func(c *gitlabmocks.Client) {
				c.On("ListProjectPipelines", projectPath, mock.Anything).
					Return([]*gitlab.PipelineInfo{{ID: 100}}, nil)
				c.On("ListPipelineJobs", projectPath, int64(100), mock.Anything).
					Return([]*gitlab.Job{
						{ID: 5, Name: "review:deploy", Status: "running"},
					}, nil)
			},
			expectedErr: `job "review:deploy" in pipeline 100 has status ` +
				`"running" (expected "manual"); builds may still be running`,
		},
		"play job error": {
			setupMock: func(c *gitlabmocks.Client) {
				c.On("ListProjectPipelines", projectPath, mock.Anything).
					Return([]*gitlab.PipelineInfo{{ID: 100}}, nil)
				c.On("ListPipelineJobs", projectPath, int64(100), mock.Anything).
					Return([]*gitlab.Job{
						{ID: 5, Name: "review:deploy", Status: "manual"},
					}, nil)
				c.On("PlayJob", projectPath, int64(5), mock.Anything).
					Return(nil, fmt.Errorf("play error"))
			},
			expectedErr: `failed to play job "review:deploy" (ID: 5): play error`,
		},
		"happy path": {
			setupMock: func(c *gitlabmocks.Client) {
				c.On("ListProjectPipelines", projectPath, mock.Anything).
					Return([]*gitlab.PipelineInfo{{ID: 100}}, nil)
				c.On("ListPipelineJobs", projectPath, int64(100), mock.Anything).
					Return([]*gitlab.Job{
						{ID: 5, Name: "review:deploy", Status: "manual"},
					}, nil)
				c.On("PlayJob", projectPath, int64(5), mock.Anything).
					Return(&gitlab.Job{
						ID:     5,
						Name:   "review:deploy",
						Status: "pending",
						WebURL: "https://gitlab.com/job/5",
					}, nil)
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			client := gitlabmocks.NewClient(t)
			tc.setupMock(client)

			job, err := findAndPlayJob(log, client, projectPath, ref, jobName, jobVars)

			if tc.expectedErr != "" {
				assert.EqualError(t, err, tc.expectedErr)
				assert.Nil(t, job)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, job)
				assert.Equal(t, int64(5), job.ID)
			}
		})
	}
}

func makePREvent(repo string, prNumber int) *github.PullRequestEvent {
	return &github.PullRequestEvent{
		Repo: &github.Repository{
			Name: github.String(repo),
		},
		Number: github.Int(prNumber),
		PullRequest: &github.PullRequest{
			Number: github.Int(prNumber),
		},
	}
}

func TestTriggerReviewDeployWithClient(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	conf := &config{githubOrganization: "mendersoftware"}

	testCases := map[string]struct {
		repoName   string
		prNumber   int
		sender     string
		enterprise bool
		setupGL    func(*gitlabmocks.Client)
		setupGH    func(*githubmocks.Client)
		errContain string
	}{
		"unsupported repo": {
			repoName:   "unknown-repo",
			prNumber:   10,
			setupGL:    func(c *gitlabmocks.Client) {},
			setupGH:    func(c *githubmocks.Client) {},
			errContain: `review app deployment is not supported for repository "unknown-repo"`,
		},
		"happy path OS": {
			repoName: "mender-server",
			prNumber: 42,
			sender:   "testuser",
			setupGL: func(c *gitlabmocks.Client) {
				c.On("ListProjectPipelines", mock.Anything, mock.Anything).
					Return([]*gitlab.PipelineInfo{{ID: 200}}, nil)
				c.On("ListPipelineJobs", mock.Anything, int64(200), mock.Anything).
					Return([]*gitlab.Job{
						{ID: 10, Name: reviewDeployJobName, Status: "manual"},
					}, nil)
				c.On("PlayJob", mock.Anything, int64(10), mock.Anything).
					Return(&gitlab.Job{
						ID:     10,
						Name:   reviewDeployJobName,
						Status: "pending",
						WebURL: "https://gitlab.com/job/10",
					}, nil)
			},
			setupGH: func(c *githubmocks.Client) {
				c.On("CreateComment", mock.Anything, "mendersoftware", "mender-server", 42,
					mock.MatchedBy(func(comment *github.IssueComment) bool {
						return comment.Body != nil &&
							assert.Contains(t, *comment.Body, "Review app deploy triggered (OS)")
					})).Return(nil)
			},
		},
		"happy path enterprise": {
			repoName:   "mender-server",
			prNumber:   42,
			sender:     "testuser",
			enterprise: true,
			setupGL: func(c *gitlabmocks.Client) {
				c.On("ListProjectPipelines", mock.Anything, mock.Anything).
					Return([]*gitlab.PipelineInfo{{ID: 200}}, nil)
				c.On("ListPipelineJobs", mock.Anything, int64(200), mock.Anything).
					Return([]*gitlab.Job{
						{ID: 10, Name: reviewDeployJobName, Status: "manual"},
					}, nil)
				c.On("PlayJob", mock.Anything, int64(10), mock.Anything).
					Return(&gitlab.Job{
						ID:     10,
						Name:   reviewDeployJobName,
						Status: "pending",
						WebURL: "https://gitlab.com/job/10",
					}, nil)
			},
			setupGH: func(c *githubmocks.Client) {
				c.On("CreateComment", mock.Anything, "mendersoftware", "mender-server", 42,
					mock.MatchedBy(func(comment *github.IssueComment) bool {
						return comment.Body != nil &&
							assert.Contains(t, *comment.Body, "Review app deploy triggered (Enterprise)")
					})).Return(nil)
			},
		},
		"findAndPlayJob fails": {
			repoName: "mender-server",
			prNumber: 42,
			sender:   "testuser",
			setupGL: func(c *gitlabmocks.Client) {
				c.On("ListProjectPipelines", mock.Anything, mock.Anything).
					Return([]*gitlab.PipelineInfo{}, nil)
			},
			setupGH:    func(c *githubmocks.Client) {},
			errContain: "no pipelines found",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			glClient := gitlabmocks.NewClient(t)
			ghClient := githubmocks.NewClient(t)
			tc.setupGL(glClient)
			tc.setupGH(ghClient)

			pr := makePREvent(tc.repoName, tc.prNumber)
			err := triggerReviewDeployWithClient(log, conf, pr, tc.sender, tc.enterprise, glClient, ghClient)

			if tc.errContain != "" {
				assert.ErrorContains(t, err, tc.errContain)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTriggerReviewE2EWithClient(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	conf := &config{githubOrganization: "mendersoftware"}

	testCases := map[string]struct {
		repoName        string
		prNumber        int
		testEnvironment string
		setupGL         func(*gitlabmocks.Client)
		setupGH         func(*githubmocks.Client)
		errContain      string
	}{
		"unsupported repo": {
			repoName:   "unknown-repo",
			prNumber:   10,
			setupGL:    func(c *gitlabmocks.Client) {},
			setupGH:    func(c *githubmocks.Client) {},
			errContain: `review app e2e tests are not supported for repository "unknown-repo"`,
		},
		"happy path default environment": {
			repoName:        "mender-server",
			prNumber:        42,
			testEnvironment: "os",
			setupGL: func(c *gitlabmocks.Client) {
				c.On("CreatePipeline", mock.Anything, mock.MatchedBy(func(opt *gitlab.CreatePipelineOptions) bool {
					return opt.Ref != nil && *opt.Ref == "pr_42"
				})).Return(&gitlab.Pipeline{
					ID:     300,
					WebURL: "https://gitlab.com/pipeline/300",
				}, nil)
			},
			setupGH: func(c *githubmocks.Client) {
				c.On("CreateComment", mock.Anything, "mendersoftware", "mender-server", 42,
					mock.MatchedBy(func(comment *github.IssueComment) bool {
						return comment.Body != nil &&
							assert.Contains(t, *comment.Body, "Pipeline-300")
					})).Return(nil)
			},
		},
		"happy path enterprise environment": {
			repoName:        "mender-server",
			prNumber:        7,
			testEnvironment: "enterprise",
			setupGL: func(c *gitlabmocks.Client) {
				c.On("CreatePipeline", mock.Anything, mock.Anything).
					Return(&gitlab.Pipeline{
						ID:     400,
						WebURL: "https://gitlab.com/pipeline/400",
					}, nil)
			},
			setupGH: func(c *githubmocks.Client) {
				c.On("CreateComment", mock.Anything, "mendersoftware", "mender-server", 7,
					mock.MatchedBy(func(comment *github.IssueComment) bool {
						return comment.Body != nil &&
							assert.Contains(t, *comment.Body, "enterprise")
					})).Return(nil)
			},
		},
		"empty environment defaults to os": {
			repoName:        "mender-server",
			prNumber:        42,
			testEnvironment: "",
			setupGL: func(c *gitlabmocks.Client) {
				c.On("CreatePipeline", mock.Anything, mock.Anything).
					Return(&gitlab.Pipeline{
						ID:     500,
						WebURL: "https://gitlab.com/pipeline/500",
					}, nil)
			},
			setupGH: func(c *githubmocks.Client) {
				c.On("CreateComment", mock.Anything, mock.Anything, mock.Anything, mock.Anything,
					mock.MatchedBy(func(comment *github.IssueComment) bool {
						return comment.Body != nil &&
							assert.Contains(t, *comment.Body, "Environment: `os`")
					})).Return(nil)
			},
		},
		"pipeline creation fails": {
			repoName:        "mender-server",
			prNumber:        42,
			testEnvironment: "os",
			setupGL: func(c *gitlabmocks.Client) {
				c.On("CreatePipeline", mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("pipeline error"))
			},
			setupGH:    func(c *githubmocks.Client) {},
			errContain: "pipeline error",
		},
		"github comment fails silently": {
			repoName:        "mender-server",
			prNumber:        42,
			testEnvironment: "os",
			setupGL: func(c *gitlabmocks.Client) {
				c.On("CreatePipeline", mock.Anything, mock.Anything).
					Return(&gitlab.Pipeline{
						ID:     600,
						WebURL: "https://gitlab.com/pipeline/600",
					}, nil)
			},
			setupGH: func(c *githubmocks.Client) {
				c.On("CreateComment",
					mock.MatchedBy(func(_ context.Context) bool { return true }),
					mock.Anything, mock.Anything, mock.Anything, mock.Anything,
				).Return(fmt.Errorf("github comment error"))
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			glClient := gitlabmocks.NewClient(t)
			ghClient := githubmocks.NewClient(t)
			tc.setupGL(glClient)
			tc.setupGH(ghClient)

			pr := makePREvent(tc.repoName, tc.prNumber)
			err := triggerReviewE2EWithClient(log, conf, pr, tc.testEnvironment, glClient, ghClient)

			if tc.errContain != "" {
				assert.ErrorContains(t, err, tc.errContain)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
