package main

import (
	"context"
	"errors"
	"testing"

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
	commentString := github.String(", Let me know if you want to start the integration pipeline by mentioning me and the command \"")
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
					&github.IssueComment{
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
