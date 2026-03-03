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

func TestGetPRStatsFull(t *testing.T) {
	mclient := &mock_github.Client{}
	ctx := context.Background()
	org := "mendersoftware"
	repo := "mender"

	now := time.Now()
	twoDaysAgo := now.AddDate(0, 0, -2)
	threeDaysAgo := now.AddDate(0, 0, -3)

	// Mock ListPullRequests for open PRs
	mclient.On("ListPullRequests", mock.Anything, org, repo, mock.MatchedBy(func(opts *github.PullRequestListOptions) bool {
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
	mclient.On("ListPullRequests", mock.Anything, org, repo, mock.MatchedBy(func(opts *github.PullRequestListOptions) bool {
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

	// Mock ListReviews for open PR #1
	mclient.On("ListReviews", mock.Anything, org, repo, 1, mock.Anything).Return([]*github.PullRequestReview{
		{
			User:        &github.User{Login: github.String("reviewer1")},
			SubmittedAt: &now,
		},
	}, nil)
	// Mock ListReviews for closed PR #2
	mclient.On("ListReviews", mock.Anything, org, repo, 2, mock.Anything).Return([]*github.PullRequestReview{}, nil)

	opts := PRStatsOptions{
		Repos: []string{repo},
		Mode:  prStatsModeFull,
	}

	report, err := getPRStats(ctx, mclient, org, opts)
	assert.NoError(t, err)
	assert.Contains(t, report, "# PR Metrics for `mender` (Last 30 Days)")
	assert.Contains(t, report, "author1")
	assert.Contains(t, report, "reviewer1")
	assert.Contains(t, report, "author2")
	assert.Contains(t, report, "Metrics Summary")
	assert.Contains(t, report, "PRs Needing Attention")
	// Full mode should show review columns
	assert.Contains(t, report, "Reviews (30d)")
	assert.Contains(t, report, "Median TTRv")

	mclient.AssertExpectations(t)
}

func TestGetPRStatsTeamModeSkipsReviews(t *testing.T) {
	mclient := &mock_github.Client{}
	ctx := context.Background()
	org := "mendersoftware"
	repo := "mender"

	now := time.Now()
	twoDaysAgo := now.AddDate(0, 0, -2)

	mclient.On("ListPullRequests", mock.Anything, org, repo, mock.MatchedBy(func(opts *github.PullRequestListOptions) bool {
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

	mclient.On("ListPullRequests", mock.Anything, org, repo, mock.MatchedBy(func(opts *github.PullRequestListOptions) bool {
		return opts.State == "closed"
	})).Return([]*github.PullRequest{}, nil)

	// ListReviews should NOT be called in team mode
	opts := PRStatsOptions{
		Repos: []string{repo},
		Mode:  prStatsModeTeam,
	}

	report, err := getPRStats(ctx, mclient, org, opts)
	assert.NoError(t, err)
	assert.Contains(t, report, "Team Activity")
	assert.NotContains(t, report, "Metrics Summary")
	assert.NotContains(t, report, "PRs Needing Attention")
	// Team mode should not show review columns
	assert.NotContains(t, report, "Reviews (30d)")
	assert.NotContains(t, report, "Median TTRv")

	mclient.AssertExpectations(t)
	mclient.AssertNotCalled(t, "ListReviews", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestGetPRStatsExcludesUsers(t *testing.T) {
	mclient := &mock_github.Client{}
	ctx := context.Background()
	org := "mendersoftware"
	repo := "mender"

	now := time.Now()
	twoDaysAgo := now.AddDate(0, 0, -2)

	mclient.On("ListPullRequests", mock.Anything, org, repo, mock.MatchedBy(func(opts *github.PullRequestListOptions) bool {
		return opts.State == "open"
	})).Return([]*github.PullRequest{
		{
			Number:    github.Int(1),
			Title:     github.String("Open PR from bot"),
			HTMLURL:   github.String("http://example.com/1"),
			User:      &github.User{Login: github.String("dependabot[bot]")},
			CreatedAt: &twoDaysAgo,
			Draft:     github.Bool(false),
		},
		{
			Number:    github.Int(2),
			Title:     github.String("Human PR"),
			HTMLURL:   github.String("http://example.com/2"),
			User:      &github.User{Login: github.String("developer1")},
			CreatedAt: &twoDaysAgo,
			Draft:     github.Bool(false),
		},
	}, nil)

	mclient.On("ListPullRequests", mock.Anything, org, repo, mock.MatchedBy(func(opts *github.PullRequestListOptions) bool {
		return opts.State == "closed"
	})).Return([]*github.PullRequest{}, nil)

	opts := PRStatsOptions{
		Repos:         []string{repo},
		Mode:          prStatsModeTeam,
		ExcludedUsers: map[string]bool{"dependabot[bot]": true},
	}

	report, err := getPRStats(ctx, mclient, org, opts)
	assert.NoError(t, err)
	assert.NotContains(t, report, "dependabot[bot]")
	assert.Contains(t, report, "developer1")

	mclient.AssertExpectations(t)
}

func TestGetPRStatsExcludesDrafts(t *testing.T) {
	mclient := &mock_github.Client{}
	ctx := context.Background()
	org := "mendersoftware"
	repo := "mender"

	now := time.Now()
	twoDaysAgo := now.AddDate(0, 0, -2)

	mclient.On("ListPullRequests", mock.Anything, org, repo, mock.MatchedBy(func(opts *github.PullRequestListOptions) bool {
		return opts.State == "open"
	})).Return([]*github.PullRequest{
		{
			Number:    github.Int(1),
			Title:     github.String("Draft PR"),
			HTMLURL:   github.String("http://example.com/1"),
			User:      &github.User{Login: github.String("author1")},
			CreatedAt: &twoDaysAgo,
			Draft:     github.Bool(true),
		},
		{
			Number:    github.Int(2),
			Title:     github.String("Ready PR"),
			HTMLURL:   github.String("http://example.com/2"),
			User:      &github.User{Login: github.String("author2")},
			CreatedAt: &twoDaysAgo,
			Draft:     github.Bool(false),
		},
	}, nil)

	mclient.On("ListPullRequests", mock.Anything, org, repo, mock.MatchedBy(func(opts *github.PullRequestListOptions) bool {
		return opts.State == "closed"
	})).Return([]*github.PullRequest{}, nil)

	opts := PRStatsOptions{
		Repos:         []string{repo},
		Mode:          prStatsModeTeam,
		ExcludeDrafts: true,
	}

	report, err := getPRStats(ctx, mclient, org, opts)
	assert.NoError(t, err)
	assert.Contains(t, report, "author2")
	// author1 should still show up in team activity (they have 0 counts but get tracked)
	// but the draft PR itself should not be in the open PRs list

	mclient.AssertExpectations(t)
}

func TestGetPRStatsMultiRepo(t *testing.T) {
	mclient := &mock_github.Client{}
	ctx := context.Background()
	org := "mendersoftware"

	now := time.Now()
	twoDaysAgo := now.AddDate(0, 0, -2)

	for _, repo := range []string{"mender", "mender-connect"} {
		mclient.On("ListPullRequests", mock.Anything, org, repo, mock.MatchedBy(func(opts *github.PullRequestListOptions) bool {
			return opts.State == "open"
		})).Return([]*github.PullRequest{
			{
				Number:    github.Int(1),
				Title:     github.String("Open PR in " + repo),
				HTMLURL:   github.String("http://example.com/" + repo + "/1"),
				User:      &github.User{Login: github.String("dev1")},
				CreatedAt: &twoDaysAgo,
				Draft:     github.Bool(false),
			},
		}, nil)

		mclient.On("ListPullRequests", mock.Anything, org, repo, mock.MatchedBy(func(opts *github.PullRequestListOptions) bool {
			return opts.State == "closed"
		})).Return([]*github.PullRequest{}, nil)
	}

	opts := PRStatsOptions{
		Repos: []string{"mender", "mender-connect"},
		Mode:  prStatsModeTeam,
	}

	report, err := getPRStats(ctx, mclient, org, opts)
	assert.NoError(t, err)
	assert.Contains(t, report, "dev1")

	mclient.AssertExpectations(t)
}

func TestCalculateWorkingTime(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		end      time.Time
		expected time.Duration
	}{
		{
			name:     "Thursday to Monday spanning weekend",
			start:    time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC),
			end:      time.Date(2023, 10, 30, 10, 0, 0, 0, time.UTC),
			expected: 48 * time.Hour,
		},
		{
			name:     "Same weekday",
			start:    time.Date(2023, 10, 25, 9, 0, 0, 0, time.UTC),
			end:      time.Date(2023, 10, 25, 17, 0, 0, 0, time.UTC),
			expected: 8 * time.Hour,
		},
		{
			name:     "Saturday to Saturday",
			start:    time.Date(2023, 10, 28, 10, 0, 0, 0, time.UTC),
			end:      time.Date(2023, 10, 28, 18, 0, 0, 0, time.UTC),
			expected: 0,
		},
		{
			name:     "Sunday to Sunday",
			start:    time.Date(2023, 10, 29, 10, 0, 0, 0, time.UTC),
			end:      time.Date(2023, 10, 29, 18, 0, 0, 0, time.UTC),
			expected: 0,
		},
		{
			name:     "Friday to Monday",
			start:    time.Date(2023, 10, 27, 14, 0, 0, 0, time.UTC),
			end:      time.Date(2023, 10, 30, 10, 0, 0, 0, time.UTC),
			expected: 20 * time.Hour, // Fri 14:00->midnight=10h, Mon midnight->10:00=10h
		},
		{
			name:     "Start after end returns zero",
			start:    time.Date(2023, 10, 30, 10, 0, 0, 0, time.UTC),
			end:      time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC),
			expected: 0,
		},
		{
			name:     "Equal times returns zero",
			start:    time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC),
			end:      time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC),
			expected: 0,
		},
		{
			name:     "Monday to Friday full week",
			start:    time.Date(2023, 10, 23, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2023, 10, 28, 0, 0, 0, 0, time.UTC),
			expected: 5 * 24 * time.Hour,
		},
		{
			name:     "Full two weeks",
			start:    time.Date(2023, 10, 16, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2023, 10, 30, 0, 0, 0, 0, time.UTC),
			expected: 10 * 24 * time.Hour,
		},
		{
			name:     "Saturday to Monday",
			start:    time.Date(2023, 10, 28, 10, 0, 0, 0, time.UTC),
			end:      time.Date(2023, 10, 30, 10, 0, 0, 0, time.UTC),
			expected: 10 * time.Hour, // only Mon 00:00->10:00
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateWorkingTime(tt.start, tt.end)
			assert.Equal(t, tt.expected, result, "case: %s", tt.name)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{0, "None"},
		{30 * time.Second, "<1m"},
		{5 * time.Minute, "5m"},
		{2*time.Hour + 30*time.Minute, "2h 30m"},
		{25*time.Hour + 15*time.Minute, "1d 1h 15m"},
		{48 * time.Hour, "2d 0h 0m"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatDuration(tt.input))
		})
	}
}

func TestGetStats(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		avg, med, p90 := getStats(nil)
		assert.Equal(t, "None", avg)
		assert.Equal(t, "None", med)
		assert.Equal(t, "None", p90)
	})

	t.Run("single value", func(t *testing.T) {
		avg, med, p90 := getStats([]time.Duration{2 * time.Hour})
		assert.Equal(t, "2h 0m", avg)
		assert.Equal(t, "2h 0m", med)
		assert.Equal(t, "2h 0m", p90)
	})

	t.Run("multiple values", func(t *testing.T) {
		durations := []time.Duration{
			1 * time.Hour,
			2 * time.Hour,
			3 * time.Hour,
			4 * time.Hour,
			5 * time.Hour,
		}
		avg, med, p90 := getStats(durations)
		assert.Equal(t, "3h 0m", avg)
		assert.Equal(t, "3h 0m", med)
		assert.Equal(t, "5h 0m", p90) // index 4 (0.9*5=4.5 -> 4)
	})

	t.Run("does not mutate input", func(t *testing.T) {
		durations := []time.Duration{5 * time.Hour, 1 * time.Hour, 3 * time.Hour}
		original := make([]time.Duration, len(durations))
		copy(original, durations)
		getStats(durations)
		assert.Equal(t, original, durations)
	})
}

func TestGetTeamRepos(t *testing.T) {
	config := &PRStatsConfig{
		Teams: []TeamConfig{
			{
				Name:             "Client",
				Repositories:     []string{"mender", "meta-mender", "mender-connect"},
				FastRepositories: []string{"mender", "meta-mender"},
			},
			{
				Name:             "Server",
				Repositories:     []string{"mender-server", "mender-helm"},
				FastRepositories: []string{"mender-server"},
			},
		},
	}

	t.Run("nil config returns current repo", func(t *testing.T) {
		repos, label := getTeamRepos("mender", nil, false)
		assert.Equal(t, []string{"mender"}, repos)
		assert.Empty(t, label)
	})

	t.Run("repo not in any team returns current repo", func(t *testing.T) {
		repos, label := getTeamRepos("unknown-repo", config, false)
		assert.Equal(t, []string{"unknown-repo"}, repos)
		assert.Empty(t, label)
	})

	t.Run("slow mode returns all team repos", func(t *testing.T) {
		repos, label := getTeamRepos("mender", config, true)
		assert.Equal(t, []string{"mender", "meta-mender", "mender-connect"}, repos)
		assert.Contains(t, label, "Client")
		assert.Contains(t, label, "All Repos")
	})

	t.Run("fast mode returns fast repos", func(t *testing.T) {
		repos, label := getTeamRepos("mender", config, false)
		assert.Equal(t, []string{"mender", "meta-mender"}, repos)
		assert.Contains(t, label, "Fast Mode")
	})

	t.Run("fast mode includes current repo if not in fast list", func(t *testing.T) {
		repos, label := getTeamRepos("mender-connect", config, false)
		assert.Contains(t, repos, "mender-connect")
		assert.Contains(t, label, "Fast Mode")
	})
}

func TestLoadPRStatsConfig(t *testing.T) {
	t.Run("loads valid config", func(t *testing.T) {
		config, err := loadPRStatsConfig("pr_stats_config.json")
		assert.NoError(t, err)
		assert.NotNil(t, config)
		assert.True(t, config.Global.ExcludeDrafts)
		assert.Equal(t, 48, config.Global.SLAHours)
		assert.Len(t, config.Teams, 2)
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		config, err := loadPRStatsConfig("/nonexistent/path.json")
		assert.Error(t, err)
		assert.Nil(t, config)
	})
}
