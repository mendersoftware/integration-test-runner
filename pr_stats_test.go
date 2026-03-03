package main

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-github/v28/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	mock_github "github.com/mendersoftware/integration-test-runner/client/github/mocks"
)

func TestGetPRStats(t *testing.T) {
	mclient := &mock_github.Client{}
	ctx := context.Background()
	org := "mendersoftware"
	repo := "mender"

	now := time.Now()
	twoDaysAgo := now.AddDate(0, 0, -2)
	threeDaysAgo := now.AddDate(0, 0, -3)

	// Mock ListPullRequests for open PRs
	mclient.On("ListPullRequests", ctx, org, repo, mock.MatchedBy(func(opts *github.PullRequestListOptions) bool {
		return opts.State == "open"
	})).Return([]*github.PullRequest{
		{
			Number:    github.Int(1),
			Title:     github.String("Open PR"),
			HTMLURL:   github.String("http://example.com/1"),
			User:      &github.User{Login: github.String("author1")},
			CreatedAt: &twoDaysAgo,
			Draft:     github.Bool(false),
		},
	}, nil)

	// Mock ListPullRequests for closed PRs
	mclient.On("ListPullRequests", ctx, org, repo, mock.MatchedBy(func(opts *github.PullRequestListOptions) bool {
		return opts.State == "closed"
	})).Return([]*github.PullRequest{
		{
			Number:    github.Int(2),
			Title:     github.String("Closed PR"),
			HTMLURL:   github.String("http://example.com/2"),
			User:      &github.User{Login: github.String("author2")},
			CreatedAt: &threeDaysAgo,
			ClosedAt:  &now,
			Draft:     github.Bool(false),
		},
	}, nil)

	// Mock ListReviews
	mclient.On("ListReviews", ctx, org, repo, 1, mock.Anything).Return([]*github.PullRequestReview{
		{
			User:        &github.User{Login: github.String("reviewer1")},
			SubmittedAt: &now,
		},
	}, nil)
	mclient.On("ListReviews", ctx, org, repo, 2, mock.Anything).Return([]*github.PullRequestReview{}, nil)

	// Mock ListTimeline
	mclient.On("ListTimeline", ctx, org, repo, 1, mock.Anything).Return([]*github.Timeline{}, nil)

	opts := PRStatsOptions{
		Repos: []string{repo},
		Mode:  "full",
	}

	report, err := getPRStats(ctx, mclient, org, opts)
	assert.NoError(t, err)
	assert.Contains(t, report, "# PR Metrics for `mender` (Last 30 Days)")
	assert.Contains(t, report, "author1")
	assert.Contains(t, report, "reviewer1")
	assert.Contains(t, report, "author2")

	mclient.AssertExpectations(t)
}

func TestCalculateWorkingTime(t *testing.T) {
	// Thursday to Monday (3 business days + 2 weekend days)
	// 2023-10-26 (Thu) 10:00 to 2023-10-30 (Mon) 10:00
	start := time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC)
	end := time.Date(2023, 10, 30, 10, 0, 0, 0, time.UTC)

	duration := calculateWorkingTime(start, end)
	// Thu 10:00 - Fri 00:00 = 14h
	// Fri 00:00 - Sat 00:00 = 24h
	// Mon 00:00 - Mon 10:00 = 10h
	// Total = 14 + 24 + 10 = 48h
	assert.Equal(t, 48*time.Hour, duration)
}
