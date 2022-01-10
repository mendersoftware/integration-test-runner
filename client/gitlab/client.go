package gitlab

import (
	"encoding/json"
	"fmt"

	"github.com/xanzy/go-gitlab"

	"github.com/mendersoftware/integration-test-runner/logger"
)

// Client represents a GitLab client
type Client interface {
	CancelPipelineBuild(path string, id int) error
	CreatePipeline(path string, options *gitlab.CreatePipelineOptions) (*gitlab.Pipeline, error)
	GetPipelineVariables(path string, id int) ([]*gitlab.PipelineVariable, error)
	ListProjectPipelines(
		path string,
		options *gitlab.ListProjectPipelinesOptions,
	) (gitlab.PipelineList, error)
}

type gitLabClient struct {
	client     *gitlab.Client
	dryRunMode bool
}

// NewGitLabClient returns a new GitLabClient for the given conf
func NewGitLabClient(accessToken string, baseURL string, dryRunMode bool) (Client, error) {
	gitlabClient := gitlab.NewClient(nil, accessToken)
	err := gitlabClient.SetBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	return &gitLabClient{
		client:     gitlabClient,
		dryRunMode: dryRunMode,
	}, nil
}

// CancelPipelineBuild cancel a pipeline
func (c *gitLabClient) CancelPipelineBuild(path string, id int) error {
	if c.dryRunMode {
		msg := fmt.Sprintf("gitlab.CancelPipelineBuild: path=%s,id=%d",
			path, id,
		)
		logger.GetRequestLogger().Push(msg)
		return nil
	}
	_, _, err := c.client.Pipelines.CancelPipelineBuild(path, id, nil)
	return err
}

// CreatePipeline creates a pipeline
func (c *gitLabClient) CreatePipeline(
	path string,
	options *gitlab.CreatePipelineOptions,
) (*gitlab.Pipeline, error) {
	if c.dryRunMode {
		optionsJSON, _ := json.Marshal(options)
		msg := fmt.Sprintf("gitlab.CreatePipeline: path=%s,options=%s",
			path, string(optionsJSON),
		)
		logger.GetRequestLogger().Push(msg)
		return &gitlab.Pipeline{}, nil
	}
	pipeline, _, err := c.client.Pipelines.CreatePipeline(path, options, nil)
	return pipeline, err
}

// GetPipelineVariables get the pipeline variables
func (c *gitLabClient) GetPipelineVariables(
	path string,
	id int,
) ([]*gitlab.PipelineVariable, error) {
	if c.dryRunMode {
		msg := fmt.Sprintf("gitlab.GetPipelineVariables: path=%s,id=%d",
			path, id,
		)
		logger.GetRequestLogger().Push(msg)
		return []*gitlab.PipelineVariable{}, nil
	}
	variables, _, err := c.client.Pipelines.GetPipelineVariables(path, id, nil)
	return variables, err
}

// ListProjectPipelines list the project pipelines
func (c *gitLabClient) ListProjectPipelines(
	path string,
	options *gitlab.ListProjectPipelinesOptions,
) (gitlab.PipelineList, error) {
	if c.dryRunMode {
		optionsJSON, _ := json.Marshal(options)
		msg := fmt.Sprintf("gitlab.ListProjectPipelines: path=%s,options=%s",
			path, string(optionsJSON),
		)
		logger.GetRequestLogger().Push(msg)
		return gitlab.PipelineList{
			&gitlab.PipelineInfo{
				ID: 1,
			},
		}, nil
	}
	pipelines, _, err := c.client.Pipelines.ListProjectPipelines(path, options, nil)
	return pipelines, err
}
