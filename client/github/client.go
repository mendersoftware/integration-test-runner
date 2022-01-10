package github

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/go-github/v28/github"
	"golang.org/x/oauth2"

	"github.com/mendersoftware/integration-test-runner/logger"
)

// Client represents a GitHub client
type Client interface {
	CreateComment(
		ctx context.Context,
		org string,
		repo string,
		number int,
		comment *github.IssueComment,
	) error
	IsOrganizationMember(ctx context.Context, org string, user string) bool
	CreatePullRequest(
		ctx context.Context,
		org string,
		repo string,
		pr *github.NewPullRequest,
	) (*github.PullRequest, error)
	GetPullRequest(
		ctx context.Context,
		org string,
		repo string,
		pr int,
	) (*github.PullRequest, error)
	ListComments(
		ctx context.Context,
		owner, repo string,
		number int,
		opts *github.IssueListCommentsOptions,
	) ([]*github.IssueComment, error)
}

type gitHubClient struct {
	client     *github.Client
	dryRunMode bool
}

// NewGitHubClient returns a new GitHubClient for the given conf
func NewGitHubClient(accessToken string, dryRunMode bool) Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return &gitHubClient{
		client:     client,
		dryRunMode: dryRunMode,
	}
}

func (c *gitHubClient) CreateComment(
	ctx context.Context,
	org string,
	repo string,
	number int,
	comment *github.IssueComment,
) error {
	if c.dryRunMode {
		commentJSON, _ := json.Marshal(comment)
		msg := fmt.Sprintf("github.CreateComment: org=%s,repo=%s,number=%d,comment=%s",
			org, repo, number, string(commentJSON),
		)
		logger.GetRequestLogger().Push(msg)
		return nil
	}
	_, _, err := c.client.Issues.CreateComment(ctx, org, repo, number, comment)
	return err
}

func (c *gitHubClient) IsOrganizationMember(ctx context.Context, org string, user string) bool {
	if c.dryRunMode {
		msg := fmt.Sprintf("github.IsOrganizationMember: org=%s,user=%s", org, user)
		logger.GetRequestLogger().Push(msg)
		return true
	}
	res, _, _ := c.client.Organizations.IsMember(ctx, org, user)
	return res
}

func (c *gitHubClient) CreatePullRequest(
	ctx context.Context,
	org string,
	repo string,
	pr *github.NewPullRequest,
) (*github.PullRequest, error) {
	if c.dryRunMode {
		prJSON, _ := json.Marshal(pr)
		msg := fmt.Sprintf("github.CreatePullRequest: org=%s,repo=%s,pr=%s",
			org, repo, string(prJSON),
		)
		logger.GetRequestLogger().Push(msg)
		return &github.PullRequest{}, nil
	}
	newPR, _, err := c.client.PullRequests.Create(ctx, org, repo, pr)
	return newPR, err
}

func (c *gitHubClient) GetPullRequest(
	ctx context.Context,
	org string,
	repo string,
	pr int,
) (*github.PullRequest, error) {
	newPR, _, err := c.client.PullRequests.Get(ctx, org, repo, pr)
	return newPR, err
}

func (c *gitHubClient) ListComments(
	ctx context.Context,
	owner, repo string,
	number int,
	opts *github.IssueListCommentsOptions,
) ([]*github.IssueComment, error) {
	comments, _, err := c.client.Issues.ListComments(ctx, owner, repo, number, opts)
	return comments, err
}
