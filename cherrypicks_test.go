package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	mock_github "github.com/mendersoftware/integration-test-runner/client/github/mocks"
	"github.com/mendersoftware/integration-test-runner/git"
	"github.com/mendersoftware/integration-test-runner/logger"
)

func TestSuggestCherryPicks(t *testing.T) {

	versionsUrl = ""
	warnString := fmt.Sprintf(apiWarningString, versionsUrl)

	gitHubOrg := "mendersoftware"

	testCases := map[string]struct {
		pr      *github.PullRequestEvent
		err     error
		comment *github.IssueComment
	}{
		"no cherry picks, not closed": {
			pr: &github.PullRequestEvent{
				Action: github.String("opened"),
				PullRequest: &github.PullRequest{
					Merged: github.Bool(false),
				},
			},
		},
		"no cherry picks, closed but not merged": {
			pr: &github.PullRequestEvent{
				Action: github.String("closed"),
				PullRequest: &github.PullRequest{
					Merged: github.Bool(false),
				},
			},
		},
		"no cherry picks, ref not master": {
			pr: &github.PullRequestEvent{
				Action: github.String("closed"),
				PullRequest: &github.PullRequest{
					Base: &github.PullRequestBranch{
						Ref: github.String("branch"),
					},
					Merged: github.Bool(true),
				},
				Repo: &github.Repository{
					Name: github.String("workflows"),
				},
			},
		},
		"no cherry picks, no changelogs": {
			pr: &github.PullRequestEvent{
				Action: github.String("closed"),
				Number: github.Int(113),
				PullRequest: &github.PullRequest{
					Base: &github.PullRequestBranch{
						Ref: github.String("master"),
						SHA: github.String("c5f65511d5437ae51da9c2e1c9017587d51044c8"),
					},
					Merged: github.Bool(true),
				},
				Repo: &github.Repository{
					Name: github.String("workflows"),
				},
			},
		},
		"cherry picks, changelogs": {
			pr: &github.PullRequestEvent{
				Action: github.String("closed"),
				Number: github.Int(88),
				PullRequest: &github.PullRequest{
					Base: &github.PullRequestBranch{
						Ref: github.String("master"),
						SHA: github.String("2294fae512f81d781b65b67844182ffb97240e83"),
					},
					Merged: github.Bool(true),
				},
				Repo: &github.Repository{
					Name: github.String("workflows"),
				},
			},
			comment: &github.IssueComment{
				Body: github.String(`
Hello :smile_cat: This PR contains changelog entries. Please, verify the need of backporting it to the following release branches:
2.2.x (release 3.3.x)
2.2.x (release 3.2.x)
2.0.x (release 3.0.x)
` + warnString),
			},
		},
		"cherry picks, changelogs, less than three release branches": {
			pr: &github.PullRequestEvent{
				Action: github.String("closed"),
				Number: github.Int(18),
				PullRequest: &github.PullRequest{
					Base: &github.PullRequestBranch{
						Ref: github.String("master"),
						SHA: github.String("11cc44037981d16e087b11ab7d6afdffae73e74e"),
					},
					Merged: github.Bool(true),
				},
				Repo: &github.Repository{
					Name: github.String("mender-connect"),
				},
			},
			comment: &github.IssueComment{
				Body: github.String(`
Hello :smile_cat: This PR contains changelog entries. Please, verify the need of backporting it to the following release branches:
2.0.x (release 3.3.x)
2.0.x (release 3.2.x)
1.2.x (release 3.0.x)
` + warnString),
			},
		},
		"cherry picks, changelogs, syntax with no space": {
			pr: &github.PullRequestEvent{
				Action: github.String("closed"),
				Number: github.Int(29),
				PullRequest: &github.PullRequest{
					Base: &github.PullRequestBranch{
						Ref: github.String("master"),
						SHA: github.String("c138b0256ec874bcd16d4cae4b598b8615b2d415"),
					},
					Merged: github.Bool(true),
				},
				Repo: &github.Repository{
					Name: github.String("mender-connect"),
				},
			},
			comment: &github.IssueComment{
				Body: github.String(`
Hello :smile_cat: This PR contains changelog entries. Please, verify the need of backporting it to the following release branches:
2.0.x (release 3.3.x)
2.0.x (release 3.2.x)
1.2.x (release 3.0.x)
` + warnString),
			},
		},
		"cherry picks, changelogs, bottable tag added": {
			pr: &github.PullRequestEvent{
				Action: github.String("closed"),
				Number: github.Int(29),
				PullRequest: &github.PullRequest{
					Base: &github.PullRequestBranch{
						Ref: github.String("master"),
						SHA: github.String("4c6d93ba936031ee00d9c115ef2dc61597bc1296"),
					},
					Head: &github.PullRequestBranch{
						Ref: github.String("logbuffering"),
						SHA: github.String("e81727b33d264175f2cd804af767c67281b6fc98"),
					},
					Merged: github.Bool(true),
				},
				Repo: &github.Repository{
					Name: github.String("mender"),
				},
			},
			comment: &github.IssueComment{
				Body: github.String(`
Hello :smile_cat: This PR contains changelog entries. Please, verify the need of backporting it to the following release branches:
3.3.x (release 3.3.x)
3.2.x (release 3.2.x)
3.0.x (release 3.0.x)
` + warnString),
			},
		},
	}

	tmpdir, err := ioutil.TempDir("", "*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	gitSetup := exec.Command("git", "clone", "https://github.com/mendersoftware/integration.git", tmpdir)
	gitSetup.Dir = tmpdir
	_, err = gitSetup.CombinedOutput()
	if err != nil {
		panic(err)
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			mclient := &mock_github.Client{}
			defer mclient.AssertExpectations(t)
			if test.comment != nil {
				mclient.On("CreateComment",
					mock.MatchedBy(func(ctx context.Context) bool {
						return true
					}),
					gitHubOrg,
					*test.pr.Repo.Name,
					*test.pr.Number,
					test.comment,
				).Return(nil)
			}

			conf := &config{
				githubProtocol:     gitProtocolHTTP,
				githubOrganization: gitHubOrg,
			}
			conf.integrationDirectory = tmpdir

			log := logrus.NewEntry(logrus.StandardLogger())
			err := suggestCherryPicks(log, test.pr, mclient, conf)
			if test.err != nil {
				assert.Error(t, err)
				assert.EqualError(t, err, test.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCherryTargetBranches(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected []string
	}{
		"Success nice syntax": {
			input: `
cherry pick to:
    * 2.6.x
    * 2.5.x
    * 2.4.x
`,
			expected: []string{"2.6.x", "2.5.x", "2.4.x"},
		},
		"Success messy syntax": {
			input: `cherry pick to:
 * 2.4.1
* 2.5.3`,
			expected: []string{"2.4.1", "2.5.3"},
		},
	}

	for name, test := range tests {
		t.Log(name)
		output, _ := parseCherryTargetBranches(test.input)
		assert.Equal(t, test.expected, output)
	}
}

func TestCherryPickToReleaseBranches(t *testing.T) {

	tests := map[string]struct {
		pr       *github.PullRequest
		err      error
		comment  *github.IssueCommentEvent
		body     string
		expected []string
	}{
		"cherry picks, changelogs": {
			pr: &github.PullRequest{
				Number: github.Int(749),
				Base: &github.PullRequestBranch{
					Ref: github.String("master"),
					SHA: github.String("04670761d39da501361501e2a4e96581b0645225"),
				},
				Head: &github.PullRequestBranch{
					Ref: github.String("pr-branch"),
					SHA: github.String("33375381a411a07429cac9fb6f800814e21dc2b8"),
				},
				Merged: github.Bool(true),
			},
			comment: &github.IssueCommentEvent{
				Issue: &github.Issue{
					Title: github.String("MEN-4703"),
				},
				Repo: &github.Repository{
					Name: github.String("mender"),
				},
				Comment: &github.IssueComment{
					Body: github.String(`
cherry-pick to:
* 2.6.x
* 2.5.x
* 2.4.x
`),
				},
			},
			body: `
cherry-pick to:
* 2.6.x
* 2.5.x
* 2.4.x
`,
			expected: []string{`I did my very best, and this is the result of the cherry pick operation:`,
				`* 2.6.x :heavy_check_mark: #42`,
				`* 2.5.x :heavy_check_mark: #42`,
				`* 2.4.x :heavy_check_mark: #42`,
			},
		},
	}

	requestLogger := logger.NewRequestLogger()
	logger.SetRequestLogger(requestLogger)
	setupLogging(&config{}, requestLogger)
	git.SetDryRunMode(true)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mclient := &mock_github.Client{}
			defer mclient.AssertExpectations(t)
			conf := &config{
				githubProtocol: gitProtocolHTTP,
			}

			mclient.On("CreateComment",
				mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}),
				conf.githubOrganization,
				*test.comment.Repo.Name,
				*test.pr.Number,
				mock.MatchedBy(func(i *github.IssueComment) bool {
					for _, expected := range test.expected {
						if !strings.Contains(*i.Body, expected) {
							return false
						}
					}
					return true
				}),
			).Return(nil)

			mclient.On("CreatePullRequest",
				mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}),
				conf.githubOrganization,
				test.comment.GetRepo().GetName(),
				mock.MatchedBy(func(_ *github.NewPullRequest) bool { return true }),
			).Return(&github.PullRequest{
				Number: github.Int(42),
			}, nil)

			log := logrus.NewEntry(logrus.StandardLogger())

			err := cherryPickPR(log, test.comment, test.pr, conf, test.body, mclient)

			if test.err != nil {
				assert.Error(t, err)
				assert.EqualError(t, err, test.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseMultiLineCherryTargetBranches(t *testing.T) {
	tests := map[string]struct {
		body     string
		expected []string
	}{
		"master": {
			body: `cherry-pick to:
		* master`,
			expected: []string{"master"},
		},
		"hosted": {
			body: `cherry-pick to:
		* hosted`,
			expected: []string{"hosted"},
		},
		"staging": {
			body: `cherry-pick to:
		* staging`,
			expected: []string{"staging"},
		},
		"feature-branch": {
			body: `cherry-pick to:
		* feature-independe_testing-1`,
			expected: []string{"feature-independe_testing-1"},
		},
		"1.2.x": {
			body: `cherry-pick to:
		* 1.2.x`,
			expected: []string{"1.2.x"},
		},
		"1.2.x with escape char": {
			body: `cherry-pick to:
		* 1.2.x\r`,
			expected: []string{"1.2.x"},
		},
		"multiple branches": {
			body: `cherry-pick to:
		* master
		* hosted
		* example-branch`,
			expected: []string{"master", "hosted", "example-branch"},
		},
	}

	for name, test := range tests {
		t.Log(name)
		res, _ := parseCherryTargetBranches(test.body)
		assert.Equal(t, test.expected, res)
	}
}

func TestParseSingleLineCherryTargetBranches(t *testing.T) {
	tests := map[string]struct {
		body     string
		expected []string
	}{
		"master": {
			body:     "cherry-pick to: `master`",
			expected: []string{"master"},
		},
		"hosted": {
			body:     "cherry-pick to: `hosted`",
			expected: []string{"hosted"},
		},
		"staging": {
			body:     "cherry-pick to: `staging`",
			expected: []string{"staging"},
		},
		"feature-branch": {
			body:     "cherry-pick to: `feature-independe_testing-1`",
			expected: []string{"feature-independe_testing-1"},
		},
		"1.2.x": {
			body:     "cherry-pick to: `1.2.x`",
			expected: []string{"1.2.x"},
		},
		"1.2.x with escape char": {
			body:     "cherry-pick to: `1.2.x`" + `\r`,
			expected: []string{"1.2.x"},
		},
		"multiple space separated branches": {
			body:     "cherry-pick to: `master` `hosted` `example-branch`",
			expected: []string{"master", "hosted", "example-branch"},
		},
		"multiple comma and space separated branches": {
			body:     "cherry-pick to: `master`, `hosted`, `example-branch`, ",
			expected: []string{"master", "hosted", "example-branch"},
		},
	}

	for name, test := range tests {
		t.Log(name)
		res, _ := parseCherryTargetBranches(test.body)
		assert.Equal(t, test.expected, res)
	}
}

const versionsResponse = `{
	"lts": ["3.3", "3.0"],
	"releases": {
		"something": "something"
	}
}
`

func TestGetLatestReleaseFromApi(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(versionsResponse))
	}))
	versions, err := getLatestReleaseFromApi(server.URL)
	assert.Nil(t, err)
	assert.Len(t, versions, 2)
	assert.Equal(t, versions[0], "3.3.x")
	assert.Equal(t, versions[1], "3.0.x")
}
