package main

import (
	"context"
	"fmt"
	"strings"
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

// pendingJob is the job GitLab hands back once a deploy job is played or retried.
func pendingJob(id int64) *gitlab.Job {
	return &gitlab.Job{
		ID:     id,
		Name:   reviewDeployJobName,
		Status: "pending",
		WebURL: fmt.Sprintf("https://gitlab.com/job/%d", id),
	}
}

func TestFindAndPlayJob(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	projectPath := "Northern.tech/Mender/mender-server"
	ref := "pr_42"
	jobName := reviewDeployJobName

	requestedByKey := "REVIEW_REQUESTED_BY"
	sender := "testuser"
	jobVars := []*gitlab.JobVariableOptions{
		{Key: &requestedByKey, Value: &sender},
	}

	const pipelineID, jobID = int64(100), int64(5)
	eligiblePipelines := []*gitlab.PipelineInfo{{ID: pipelineID}}
	targetJob := func(status string) []*gitlab.Job {
		return []*gitlab.Job{{ID: jobID, Name: jobName, Status: status}}
	}

	testCases := map[string]struct {
		pipelines     []*gitlab.PipelineInfo
		pipelinesErr  error
		jobs          []*gitlab.Job
		jobsErr       error
		playedJob     *gitlab.Job
		playErr       error
		retriedJob    *gitlab.Job
		retryErr      error
		expectedErr   string
		expectedJobID int64
	}{
		"list pipelines error": {
			pipelinesErr: fmt.Errorf("list error"),
			expectedErr:  "failed to list pipelines for ref pr_42: list error",
		},
		"no pipelines found": {
			pipelines: []*gitlab.PipelineInfo{},
			expectedErr: "no eligible (non-skipped) pipelines found for ref pr_42 " +
				"in project " + projectPath,
		},
		"all pipelines skipped or canceled": {
			pipelines: []*gitlab.PipelineInfo{
				{ID: 102, Status: "skipped"},
				{ID: 101, Status: "canceled"},
			},
			expectedErr: "no eligible (non-skipped) pipelines found for ref pr_42 " +
				"in project " + projectPath,
		},
		"skipped duplicate pipeline is bypassed": {
			pipelines: []*gitlab.PipelineInfo{
				{ID: 101, Status: "skipped"},
				{ID: pipelineID, Status: "success"},
			},
			jobs:          targetJob("manual"),
			playedJob:     pendingJob(jobID),
			expectedJobID: jobID,
		},
		"list jobs error": {
			pipelines:   eligiblePipelines,
			jobsErr:     fmt.Errorf("jobs error"),
			expectedErr: "failed to list jobs for pipeline 100: jobs error",
		},
		"job not found": {
			pipelines:   eligiblePipelines,
			jobs:        []*gitlab.Job{{ID: 1, Name: "build:backend:docker", Status: "success"}},
			expectedErr: `job "review:deploy" not found in pipeline 100`,
		},
		"job not manual": {
			pipelines: eligiblePipelines,
			jobs:      targetJob("running"),
			expectedErr: `job "review:deploy" in pipeline 100 has status ` +
				`"running" (expected "manual"); builds may still be running`,
		},
		"play job error": {
			pipelines:   eligiblePipelines,
			jobs:        targetJob("manual"),
			playErr:     fmt.Errorf("play error"),
			expectedErr: `failed to play job "review:deploy" (ID: 5): play error`,
		},
		"happy path": {
			pipelines:     eligiblePipelines,
			jobs:          targetJob("manual"),
			playedJob:     pendingJob(jobID),
			expectedJobID: jobID,
		},
		"failed job is retried": {
			pipelines:     eligiblePipelines,
			jobs:          targetJob("failed"),
			retriedJob:    pendingJob(6),
			expectedJobID: 6,
		},
		"succeeded job is retried": {
			pipelines:     eligiblePipelines,
			jobs:          targetJob("success"),
			retriedJob:    pendingJob(7),
			expectedJobID: 7,
		},
		"canceled job is retried": {
			pipelines:     eligiblePipelines,
			jobs:          targetJob("canceled"),
			retriedJob:    pendingJob(8),
			expectedJobID: 8,
		},
		"retry job error": {
			pipelines:   eligiblePipelines,
			jobs:        targetJob("failed"),
			retryErr:    fmt.Errorf("retry error"),
			expectedErr: `failed to retry job "review:deploy" (ID: 5): retry error`,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			client := gitlabmocks.NewClient(t)
			client.On("ListProjectPipelines", projectPath, mock.Anything).
				Return(tc.pipelines, tc.pipelinesErr)
			if tc.jobs != nil || tc.jobsErr != nil {
				client.On("ListPipelineJobs", projectPath, pipelineID, mock.Anything).
					Return(tc.jobs, tc.jobsErr)
			}
			if tc.playedJob != nil || tc.playErr != nil {
				client.On("PlayJob", projectPath, jobID, mock.Anything).
					Return(tc.playedJob, tc.playErr)
			}
			if tc.retriedJob != nil || tc.retryErr != nil {
				client.On("RetryJob", projectPath, jobID).
					Return(tc.retriedJob, tc.retryErr)
			}

			job, err := findAndPlayJob(log, client, projectPath, ref, jobName, jobVars)

			if tc.expectedErr != "" {
				assert.EqualError(t, err, tc.expectedErr)
				assert.Nil(t, job)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, job)
			assert.Equal(t, tc.expectedJobID, job.ID)
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

	const pipelineID, jobID = int64(200), int64(10)

	expectJobLookup := func(c *gitlabmocks.Client, status string) {
		c.On("ListProjectPipelines", mock.Anything, mock.Anything).
			Return([]*gitlab.PipelineInfo{{ID: pipelineID}}, nil)
		c.On("ListPipelineJobs", mock.Anything, pipelineID, mock.Anything).
			Return([]*gitlab.Job{
				{ID: jobID, Name: reviewDeployJobName, Status: status},
			}, nil)
	}
	expectComment := func(c *githubmocks.Client, prNumber int, contains string) {
		c.On("CreateComment", mock.Anything, "mendersoftware", "mender-server", prNumber,
			mock.MatchedBy(func(comment *github.IssueComment) bool {
				return comment.Body != nil && strings.Contains(*comment.Body, contains)
			})).Return(nil)
	}

	testCases := map[string]struct {
		repoName   string
		prNumber   int
		sender     string
		enterprise bool
		setupEnv   func(*testing.T)
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
				expectJobLookup(c, "manual")
				c.On("PlayJob", mock.Anything, jobID, mock.Anything).
					Return(pendingJob(jobID), nil)
			},
			setupGH: func(c *githubmocks.Client) {
				expectComment(c, 42, "Review app deploy triggered (OS)")
			},
		},
		"happy path enterprise": {
			repoName:   "mender-server",
			prNumber:   42,
			sender:     "testuser",
			enterprise: true,
			setupGL: func(c *gitlabmocks.Client) {
				expectJobLookup(c, "manual")
				c.On("PlayJob", mock.Anything, jobID, mock.Anything).
					Return(pendingJob(jobID), nil)
			},
			setupGH: func(c *githubmocks.Client) {
				expectComment(c, 42, "Review app deploy triggered (Enterprise)")
			},
		},
		"happy path enterprise ent-on-os": {
			repoName:   "mender-server",
			prNumber:   42,
			sender:     "testuser",
			enterprise: true,
			setupEnv: func(t *testing.T) {
				t.Setenv("REGISTRY_MENDER_IO_USERNAME", "test-registry-user")
				t.Setenv("REGISTRY_MENDER_IO_PASSWORD", "test-registry-token")
			},
			setupGL: func(c *gitlabmocks.Client) {
				expectJobLookup(c, "manual")
				c.On("PlayJob", mock.Anything, jobID,
					mock.MatchedBy(func(opt *gitlab.PlayJobOptions) bool {
						if opt.JobVariablesAttributes == nil {
							return false
						}
						vars := map[string]string{}
						for _, v := range *opt.JobVariablesAttributes {
							vars[*v.Key] = *v.Value
						}
						return vars["REVIEW_APPS_ENT_ON_OS"] == "true" &&
							vars["REGISTRY_MENDER_IO_USERNAME"] == "test-registry-user" &&
							vars["REGISTRY_MENDER_IO_PASSWORD"] == "test-registry-token"
					})).Return(pendingJob(jobID), nil)
			},
			setupGH: func(c *githubmocks.Client) {
				expectComment(c, 42, "Review app deploy triggered (Enterprise)")
			},
		},
		"prior job is retried": {
			repoName: "mender-server",
			prNumber: 42,
			sender:   "testuser",
			setupGL: func(c *gitlabmocks.Client) {
				expectJobLookup(c, "success")
				c.On("RetryJob", mock.Anything, jobID).Return(pendingJob(11), nil)
			},
			setupGH: func(c *githubmocks.Client) {
				expectComment(c, 42, "https://gitlab.com/job/11")
			},
		},
		"github comment fails silently": {
			repoName: "mender-server",
			prNumber: 42,
			sender:   "testuser",
			setupGL: func(c *gitlabmocks.Client) {
				expectJobLookup(c, "manual")
				c.On("PlayJob", mock.Anything, jobID, mock.Anything).
					Return(pendingJob(jobID), nil)
			},
			setupGH: func(c *githubmocks.Client) {
				c.On("CreateComment", mock.Anything, mock.Anything, mock.Anything,
					mock.Anything, mock.Anything).Return(fmt.Errorf("github comment error"))
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
			errContain: "no eligible (non-skipped) pipelines found",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			if tc.setupEnv != nil {
				tc.setupEnv(t)
			}
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
