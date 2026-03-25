package main

import (
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	gitlabmocks "github.com/mendersoftware/integration-test-runner/client/gitlab/mocks"
)

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
			expectedErr: `job "review:deploy" in pipeline 100 has status "running" (expected "manual"); builds may still be running`,
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
