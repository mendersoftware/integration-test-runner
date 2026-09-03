package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	mock_gitlab "github.com/mendersoftware/integration-test-runner/client/gitlab/mocks"
)

func TestStartPRPipelineClassifiesErrors(t *testing.T) {
	testCases := map[string]struct {
		createPipelineErr error
		expectSkipped     bool
		expectErr         bool
	}{
		"pipeline started": {
			createPipelineErr: nil,
		},
		"missing CI config is not a failure": {
			createPipelineErr: errors.New(
				`POST .../pipeline: 400 {message: {base: [Missing CI config file]}}`),
			expectSkipped: true,
			expectErr:     true,
		},
		"no stages is not a failure": {
			createPipelineErr: errors.New("No stages / jobs for this pipeline"),
			expectSkipped:     true,
			expectErr:         true,
		},
		"anything else is a failure": {
			createPipelineErr: errors.New("403 Forbidden"),
			expectErr:         true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			client := &mock_gitlab.Client{}
			client.On("CreatePipeline", "Northern.tech/Mender/mender", mock.Anything).
				Return(&gitlab.Pipeline{WebURL: "https://gitlab.com/pipeline/1"},
					tc.createPipelineErr)

			event := &github.PullRequestEvent{
				Number:       github.Int(42),
				Organization: &github.Organization{Login: github.String("mendersoftware")},
				Repo: &github.Repository{
					Name:     github.String("mender"),
					FullName: github.String("mendersoftware/mender"),
				},
				PullRequest: &github.PullRequest{
					Head: &github.PullRequestBranch{
						Ref:  github.String("feature"),
						SHA:  github.String("abc"),
						Repo: &github.Repository{FullName: github.String("mendersoftware/mender")},
					},
					Base: &github.PullRequestBranch{
						Ref: github.String("master"),
						SHA: github.String("def"),
					},
				},
			}

			err := startPRPipelineWithClient(
				logrus.NewEntry(logrus.StandardLogger()),
				"pr_42", event, func() bool { return true }, client,
			)

			if !tc.expectErr {
				assert.NoError(t, err)
				return
			}
			assert.Error(t, err)
			assert.Equal(t, tc.expectSkipped, errors.Is(err, errNoPipelineToRun))
			if tc.expectSkipped {
				// The project path is kept so the log says which repo was skipped.
				assert.Contains(t, err.Error(), "Northern.tech/Mender/mender")
			} else {
				assert.Equal(t, tc.createPipelineErr, err)
			}
		})
	}
}

func TestShouldReportPipelineFailure(t *testing.T) {
	assert.False(t, shouldReportPipelineFailure(nil))
	assert.False(t, shouldReportPipelineFailure(
		fmt.Errorf("Northern.tech/Mender/mender: %w", errNoPipelineToRun)))
	assert.True(t, shouldReportPipelineFailure(errors.New("403 Forbidden")))
}
