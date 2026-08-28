package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	mock_github "github.com/mendersoftware/integration-test-runner/client/github/mocks"
)

func TestGetFirstMatchingBotCommentInPR(t *testing.T) {
	// Needed because the original is const, and we need to take address-of.
	githubBotName := githubBotName

	type returnValues struct {
		issueComments []*github.IssueComment
		error         error
	}
	commentString := github.String(", Let me know if you want to start the client pipeline by mentioning me and the command \"")
	conf := &config{
		githubOrganization: "mendersoftware",
	}
	testCases := map[string]struct {
		pr         *github.PullRequestEvent
		expectNil  bool
		returnVals returnValues
	}{
		"Bot has not commented": {
			pr: &github.PullRequestEvent{
				PullRequest: &github.PullRequest{
					Merged: github.Bool(false),
				},
				Repo: &github.Repository{
					Name: github.String("I am not the bot"),
					Owner: &github.User{
						Name: github.String("mendersoftware"),
					},
				},
				Number: github.Int(6),
			},
			returnVals: returnValues{
				issueComments: nil,
				error:         errors.New("Failed to retrieve the comments"),
			},
			expectNil: true,
		},
		"Bot has commented": {
			pr: &github.PullRequestEvent{
				PullRequest: &github.PullRequest{
					Merged: github.Bool(false),
				},
				Repo: &github.Repository{
					Name: commentString,
					Owner: &github.User{
						Name: github.String("mendersoftware"),
					},
				},
				Number: github.Int(6),
			},
			returnVals: returnValues{
				issueComments: []*github.IssueComment{
					{
						Body: commentString,
						User: &github.User{
							Login: &githubBotName,
						},
					},
				},
				error: nil,
			},
			expectNil: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			mclient := &mock_github.Client{}
			defer mclient.AssertExpectations(t)

			mclient.On("ListComments",
				mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}),
				*tc.pr.Repo.Owner.Name,
				*tc.pr.Repo.Name,
				*tc.pr.Number,
				mock.MatchedBy(func(*github.IssueListCommentsOptions) bool {
					return true
				}),
			).Return(tc.returnVals.issueComments, tc.returnVals.error)

			log := logrus.NewEntry(logrus.StandardLogger())
			issue := getFirstMatchingBotCommentInPR(log, mclient, tc.pr, *commentString, conf)
			if tc.expectNil {
				assert.Nil(t, issue)
			} else {
				require.NotNil(t, issue)
				assert.Equal(t, githubBotName, *issue.User.Login)
			}
		})
	}
}

func TestChangelogComments(t *testing.T) {
	// Needed because the original is const, and we need to take address-of.
	githubBotName := githubBotName

	dummyName := "dummyName"

	const (
		noIssue = iota
		matchingIssue
		nonMatchingIssue
	)

	testRepo := "test-repo"
	conf := &config{
		githubOrganization: "mendersoftware",
	}
	testCases := map[string]struct {
		pr            *github.PullRequestEvent
		issue         int // Constants from above.
		changelogText string
		warningText   string
		update        bool
		deletion      bool
		commentID     int64
		userName      string
	}{
		"No comment exists": {
			pr: &github.PullRequestEvent{
				PullRequest: &github.PullRequest{
					Merged: github.Bool(false),
				},
				Repo: &github.Repository{
					Name: &testRepo,
					Owner: &github.User{
						Name: github.String("mendersoftware"),
					},
				},
				Number: github.Int(6),
			},
			issue:         noIssue,
			changelogText: "No comment exists",
			update:        true,
			deletion:      false,
		},
		"Existing, identical comment": {
			pr: &github.PullRequestEvent{
				PullRequest: &github.PullRequest{
					Merged: github.Bool(false),
				},
				Repo: &github.Repository{
					Name: &testRepo,
					Owner: &github.User{
						Name: github.String("mendersoftware"),
					},
				},
				Number: github.Int(6),
			},
			issue:         matchingIssue,
			changelogText: "Existing, identical comment",
			update:        false,
			deletion:      false,
			commentID:     123,
			userName:      githubBotName,
		},
		"Existing, identical comment by different user": {
			pr: &github.PullRequestEvent{
				PullRequest: &github.PullRequest{
					Merged: github.Bool(false),
				},
				Repo: &github.Repository{
					Name: &testRepo,
					Owner: &github.User{
						Name: github.String("mendersoftware"),
					},
				},
				Number: github.Int(6),
			},
			issue:         matchingIssue,
			changelogText: "Existing, identical comment by different user",
			update:        true,
			deletion:      false,
			commentID:     123,
			userName:      dummyName,
		},
		"Existing, different comment": {
			pr: &github.PullRequestEvent{
				PullRequest: &github.PullRequest{
					Merged: github.Bool(false),
				},
				Repo: &github.Repository{
					Name: &testRepo,
					Owner: &github.User{
						Name: github.String("mendersoftware"),
					},
				},
				Number: github.Int(6),
			},
			issue:         nonMatchingIssue,
			changelogText: "Existing, different comment",
			update:        true,
			deletion:      true,
			commentID:     123,
			userName:      githubBotName,
		},
		"Existing, different comment by different user": {
			pr: &github.PullRequestEvent{
				PullRequest: &github.PullRequest{
					Merged: github.Bool(false),
				},
				Repo: &github.Repository{
					Name: &testRepo,
					Owner: &github.User{
						Name: github.String("mendersoftware"),
					},
				},
				Number: github.Int(6),
			},
			issue:         nonMatchingIssue,
			changelogText: "Existing, different comment by different user",
			update:        true,
			deletion:      false,
			commentID:     123,
			userName:      dummyName,
		},
		"Existing, identical comment with warnings": {
			pr: &github.PullRequestEvent{
				PullRequest: &github.PullRequest{
					Merged: github.Bool(false),
				},
				Repo: &github.Repository{
					Name: &testRepo,
					Owner: &github.User{
						Name: github.String("mendersoftware"),
					},
				},
				Number: github.Int(6),
			},
			issue:         matchingIssue,
			changelogText: "Existing, identical comment",
			warningText:   "Various warnings",
			update:        false,
			deletion:      false,
			commentID:     123,
			userName:      githubBotName,
		},
		"Existing, different comment with warnings": {
			pr: &github.PullRequestEvent{
				PullRequest: &github.PullRequest{
					Merged: github.Bool(false),
				},
				Repo: &github.Repository{
					Name: &testRepo,
					Owner: &github.User{
						Name: github.String("mendersoftware"),
					},
				},
				Number: github.Int(6),
			},
			issue:         nonMatchingIssue,
			changelogText: "Existing, different comment",
			warningText:   "Various warnings",
			update:        true,
			deletion:      true,
			commentID:     123,
			userName:      githubBotName,
		},
		"Empty changelog, and no previous comment": {
			pr: &github.PullRequestEvent{
				PullRequest: &github.PullRequest{
					Merged: github.Bool(false),
				},
				Repo: &github.Repository{
					Name: &testRepo,
					Owner: &github.User{
						Name: github.String("mendersoftware"),
					},
				},
				Number: github.Int(6),
			},
			issue:         noIssue,
			changelogText: "### Changelogs\n\n",
			update:        false,
			deletion:      false,
			commentID:     123,
			userName:      githubBotName,
		},
		"Empty changelog, and previous, different comment": {
			pr: &github.PullRequestEvent{
				PullRequest: &github.PullRequest{
					Merged: github.Bool(false),
				},
				Repo: &github.Repository{
					Name: &testRepo,
					Owner: &github.User{
						Name: github.String("mendersoftware"),
					},
				},
				Number: github.Int(6),
			},
			issue:         nonMatchingIssue,
			changelogText: "### Changelogs\n\n",
			update:        true,
			deletion:      true,
			commentID:     123,
			userName:      githubBotName,
		},
		"Empty changelog, and previous, identical comment": {
			pr: &github.PullRequestEvent{
				PullRequest: &github.PullRequest{
					Merged: github.Bool(false),
				},
				Repo: &github.Repository{
					Name: &testRepo,
					Owner: &github.User{
						Name: github.String("mendersoftware"),
					},
				},
				Number: github.Int(6),
			},
			issue:         matchingIssue,
			changelogText: "### Changelogs\n\n",
			update:        false,
			deletion:      false,
			commentID:     123,
			userName:      githubBotName,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			mclient := &mock_github.Client{}
			defer mclient.AssertExpectations(t)

			commentText := assembleCommentText(tc.changelogText, tc.warningText)

			mclient.On("ListComments",
				mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}),
				*tc.pr.Repo.Owner.Name,
				*tc.pr.Repo.Name,
				*tc.pr.Number,
				mock.MatchedBy(func(*github.IssueListCommentsOptions) bool {
					return true
				}),
			).Return(func() []*github.IssueComment {
				var text string
				switch tc.issue {
				case noIssue:
					return []*github.IssueComment{}
				case matchingIssue:
					text = commentText
				case nonMatchingIssue:
					text = changelogPrefix + "non-matching-text"
				default:
					t.Fatal("Invalid issue type in tc")
					// Will never get here, but Golang requires it.
					return nil
				}
				return []*github.IssueComment{
					{
						ID:   &tc.commentID,
						Body: &text,
						User: &github.User{
							Login: &tc.userName,
						},
					},
				}
			}(), nil)

			if tc.deletion {
				mclient.On("DeleteComment",
					mock.MatchedBy(func(ctx context.Context) bool {
						return true
					}),
					*tc.pr.Repo.Owner.Name,
					*tc.pr.Repo.Name,
					tc.commentID,
				).Return(nil)
			}

			if tc.update {
				mclient.On("CreateComment",
					mock.MatchedBy(func(ctx context.Context) bool {
						return true
					}),
					*tc.pr.Repo.Owner.Name,
					*tc.pr.Repo.Name,
					*tc.pr.Number,
					mock.MatchedBy(func(issue *github.IssueComment) bool {
						assert.Equal(t, commentText, *issue.Body)
						if tc.warningText == "" {
							assert.NotContains(t, *issue.Body, warningHeader)
						} else {
							assert.Contains(t, *issue.Body, warningHeader)
						}
						return true
					}),
				).Return(nil)
			}

			log := logrus.NewEntry(logrus.StandardLogger())

			updatePullRequestChangelogComments(
				log,
				&gin.Context{},
				mclient,
				tc.pr,
				conf,
				tc.changelogText,
				tc.warningText,
			)

			if !tc.update {
				mclient.AssertNotCalled(t, "CreateComment")
			}

			if !tc.deletion {
				mclient.AssertNotCalled(t, "DeleteComment")
			}
		})
	}
}

func TestGetTitleOptions(t *testing.T) {
	testCases := map[string]struct {
		InputTitle string
		Output     TitleOptions
	}{
		"NoCI": {
			InputTitle: "[NoCI] This is a title",
			Output:     TitleOptions{SkipCI: true},
		},
		"No options": {
			InputTitle: "This is a title",
			Output:     TitleOptions{},
		},
		"Ignore unknown options": {
			InputTitle: "[unknown options] This is a title",
			Output:     TitleOptions{},
		},
	}
	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			titleOptions := getTitleOptions(tc.InputTitle)
			assert.Equal(t, tc.Output, titleOptions)
		})
	}
}

func TestLabelPR(t *testing.T) {
	conf := &config{githubOrganization: "mendersoftware"}
	pr := &github.PullRequestEvent{
		Repo: &github.Repository{
			Name: github.String("mender"),
		},
		Number: github.Int(42),
	}
	log := logrus.NewEntry(logrus.New())

	mclient := &mock_github.Client{}
	mclient.On("AddLabelsToPullRequest",
		mock.Anything,
		"mendersoftware",
		"mender",
		42,
		[]string{externalContributionLabel},
	).Return(nil).Once()
	labelPR(context.Background(), log, mclient, pr, conf, externalContributionLabel)
	mclient.AssertExpectations(t)
}

func TestRetryOnError(t *testing.T) {
	boom := errors.New("boom")

	testCases := map[string]struct {
		results          []error
		decide           func(error) retryDecision
		expectedErr      error
		expectedAttempts int
		expectedSleeps   []time.Duration
	}{
		"stops on the first success": {
			results:          []error{nil, boom},
			decide:           func(e error) retryDecision { return retryStop },
			expectedErr:      nil,
			expectedAttempts: 1,
		},
		"stops without retrying when told to": {
			results:          []error{boom, nil},
			decide:           func(e error) retryDecision { return retryStop },
			expectedErr:      boom,
			expectedAttempts: 1,
		},
		"retries until the decision changes": {
			results: []error{boom, boom, nil},
			decide: func(e error) retryDecision {
				if e == nil {
					return retryStop
				}
				return retryAgain
			},
			expectedErr:      nil,
			expectedAttempts: 3,
			expectedSleeps:   []time.Duration{2 * time.Second, 4 * time.Second},
		},
		"gives up after retryMaxAttempts and returns the last error": {
			decide:           func(e error) retryDecision { return retryAgain },
			expectedErr:      boom,
			expectedAttempts: retryMaxAttempts,
			expectedSleeps: []time.Duration{
				2 * time.Second, 4 * time.Second, 8 * time.Second,
				16 * time.Second, 32 * time.Second, 64 * time.Second,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			var slept []time.Duration
			realSleep := sleep
			sleep = func(d time.Duration) { slept = append(slept, d) }
			defer func() { sleep = realSleep }()

			attempts := 0
			err := retryOnError(retryParams{
				retryFunc: func() error {
					attempts++
					if attempts <= len(tc.results) {
						return tc.results[attempts-1]
					}
					return boom
				},
				decide: tc.decide,
			})

			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedAttempts, attempts)
			assert.Equal(t, tc.expectedSleeps, slept)
			assert.Len(t, slept, tc.expectedAttempts-1,
				"there must be no sleep after the final attempt")
		})
	}
}

func TestNoPipelineToRunPattern(t *testing.T) {
	assert.True(t, noPipelineToRunPattern.MatchString(
		`POST .../pipeline: 400 {message: {base: [Missing CI config file]}}`))
	assert.True(t, noPipelineToRunPattern.MatchString(
		"No stages / jobs for this pipeline"))
	assert.False(t, noPipelineToRunPattern.MatchString(
		"403 Forbidden"))
}
