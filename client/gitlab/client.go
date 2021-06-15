package gitlab

import (
	"github.com/xanzy/go-gitlab"
)

// GitLabClient represents a GitLab client
type Client interface {
	CancelPipelineBuild(path string, id int) error
	CreatePipeline(path string, options *gitlab.CreatePipelineOptions) (*gitlab.Pipeline, error)
	GetPipelineVariables(path string, id int) ([]*gitlab.PipelineVariable, error)
	ListProjectPipelines(path string, options *gitlab.ListProjectPipelinesOptions) (gitlab.PipelineList, error)
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
	_, _, err := c.client.Pipelines.CancelPipelineBuild(path, id, nil)
	return err
}

// CreatePipeline creates a pipeline
func (c *gitLabClient) CreatePipeline(path string, options *gitlab.CreatePipelineOptions) (*gitlab.Pipeline, error) {
	pipeline, _, err := c.client.Pipelines.CreatePipeline(path, options, nil)
	return pipeline, err
}

// GetPipelineVariables get the pipeline variables
func (c *gitLabClient) GetPipelineVariables(path string, id int) ([]*gitlab.PipelineVariable, error) {
	variables, _, err := c.client.Pipelines.GetPipelineVariables(path, id, nil)
	return variables, err
}

// ListProjectPipelines list the project pipelines
func (c *gitLabClient) ListProjectPipelines(path string, options *gitlab.ListProjectPipelinesOptions) (gitlab.PipelineList, error) {
	pipelines, _, err := c.client.Pipelines.ListProjectPipelines(path, options, nil)
	return pipelines, err
}
