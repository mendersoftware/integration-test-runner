package main

import (
	"context"
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

const versionsResponse = `{
  "lts": ["3.3", "3.0"],
  "releases": {
    "3.4": {
      "3.4.0": {
	      "release_date": "2022-09-25",
	      "release": "3.4.0",
	      "repos": [
		{ "name": "auditlogs", "version": "3.0.2" },
		{ "name": "create-artifact-worker", "version": "1.2.0" },
		{ "name": "deployments", "version": "4.3.0" },
		{ "name": "deployments-enterprise", "version": "4.3.0" },
		{ "name": "deviceauth", "version": "3.3.0" },
		{ "name": "deviceauth-enterprise", "version": "3.3.0" },
		{ "name": "deviceconfig", "version": "1.2.2" },
		{ "name": "deviceconnect", "version": "1.3.3" },
		{ "name": "devicemonitor", "version": "1.3.0" },
		{ "name": "gui", "version": "3.4.0" },
		{ "name": "integration", "version": "3.4.0" },
		{ "name": "inventory", "version": "4.2.1" },
		{ "name": "inventory-enterprise", "version": "4.2.1" },
		{ "name": "iot-manager", "version": "1.1.0" },
		{ "name": "mender", "version": "3.4.0" },
		{ "name": "mender-artifact", "version": "3.9.0" },
		{ "name": "mender-binary-delta", "version": "1.4.1" },
		{ "name": "mender-cli", "version": "1.9.0" },
		{ "name": "mender-configure-module", "version": "1.0.4" },
		{ "name": "mender-connect", "version": "2.1.0" },
		{ "name": "mender-convert", "version": "3.0.1" },
		{ "name": "mender-gateway", "version": "1.0.1" },
		{ "name": "monitor-client", "version": "1.2.1" },
		{ "name": "mtls-ambassador", "version": "1.1.0" },
		{ "name": "tenantadm", "version": "3.5.0" },
		{ "name": "useradm", "version": "1.19.0" },
		{ "name": "useradm-enterprise", "version": "1.19.0" },
		{ "name": "workflows", "version": "2.3.0" },
		{ "name": "workflows-enterprise", "version": "2.3.0" }
	      ]
	    }
    },
"3.3": {
      "supported_until": "2023-06",
      "3.3.1": {
        "release_date": "2022-10-19",
        "release": "3.3.1",
        "repos": [
          { "name": "auditlogs", "version": "3.0.2" },
          { "name": "create-artifact-worker", "version": "1.1.2" },
          { "name": "deployments", "version": "4.2.1" },
          { "name": "deployments-enterprise", "version": "4.2.1" },
          { "name": "deviceauth", "version": "3.2.2" },
          { "name": "deviceauth-enterprise", "version": "3.2.2" },
          { "name": "deviceconfig", "version": "1.2.2" },
          { "name": "deviceconnect", "version": "1.3.3" },
          { "name": "devicemonitor", "version": "1.2.1" },
          { "name": "gui", "version": "3.3.1" },
          { "name": "integration", "version": "3.3.1" },
          { "name": "inventory", "version": "4.2.1" },
          { "name": "inventory-enterprise", "version": "4.2.1" },
          { "name": "iot-manager", "version": "1.0.3" },
          { "name": "mender", "version": "3.3.1" },
          { "name": "mender-artifact", "version": "3.8.1" },
          { "name": "mender-binary-delta", "version": "1.4.1" },
          { "name": "mender-cli", "version": "1.8.1" },
          { "name": "mender-configure-module", "version": "1.0.4" },
          { "name": "mender-connect", "version": "2.0.2" },
          { "name": "mender-convert", "version": "3.0.1" },
          { "name": "mender-gateway", "version": "1.0.1" },
          { "name": "monitor-client", "version": "1.2.1" },
          { "name": "mtls-ambassador", "version": "1.0.2" },
          { "name": "tenantadm", "version": "3.4.1" },
          { "name": "useradm", "version": "1.18.1" },
          { "name": "useradm-enterprise", "version": "1.18.1" },
          { "name": "workflows", "version": "2.2.2" },
          { "name": "workflows-enterprise", "version": "2.2.2" }
        ]
      },
      "3.3.0": {
        "release_date": "2022-06-14",
        "release": "3.3.0",
        "repos": [
          { "name": "auditlogs", "version": "3.0.1" },
          { "name": "create-artifact-worker", "version": "1.1.2" },
          { "name": "deployments", "version": "4.2.0" },
          { "name": "deployments-enterprise", "version": "4.2.0" },
          { "name": "deviceauth", "version": "3.2.1" },
          { "name": "deviceauth-enterprise", "version": "3.2.1" },
          { "name": "deviceconfig", "version": "1.2.1" },
          { "name": "deviceconnect", "version": "1.3.2" },
          { "name": "devicemonitor", "version": "1.2.0" },
          { "name": "gui", "version": "3.3.0" },
          { "name": "integration", "version": "3.3.0" },
          { "name": "inventory", "version": "4.2.0" },
          { "name": "inventory-enterprise", "version": "4.2.0" },
          { "name": "iot-manager", "version": "1.0.2" },
          { "name": "mender", "version": "3.3.0" },
          { "name": "mender-artifact", "version": "3.8.0" },
          { "name": "mender-binary-delta", "version": "1.3.1" },
          { "name": "mender-cli", "version": "1.8.0" },
          { "name": "mender-configure-module", "version": "1.0.4" },
          { "name": "mender-connect", "version": "2.0.2" },
          { "name": "mender-convert", "version": "3.0.0" },
          { "name": "mender-gateway", "version": "1.0.0" },
          { "name": "monitor-client", "version": "1.2.0" },
          { "name": "mtls-ambassador", "version": "1.0.2" },
          { "name": "reporting", "version": "master" },
          { "name": "tenantadm", "version": "3.4.0" },
          { "name": "useradm", "version": "1.18.0" },
          { "name": "useradm-enterprise", "version": "1.18.0" },
          { "name": "workflows", "version": "2.2.1" },
          { "name": "workflows-enterprise", "version": "2.2.1" }
        ]
      }
    },
    "3.2": {
      "3.2.2": {
        "release_date": "2022-04-21",
        "release": "3.2.2",
        "repos": [
          { "name": "auditlogs", "version": "3.0.0" },
          { "name": "create-artifact-worker", "version": "1.1.1" },
          { "name": "deployments", "version": "4.1.0" },
          { "name": "deployments-enterprise", "version": "4.1.0" },
          { "name": "deviceauth", "version": "3.2.0" },
          { "name": "deviceauth-enterprise", "version": "3.2.0" },
          { "name": "deviceconfig", "version": "1.2.0" },
          { "name": "deviceconnect", "version": "1.3.1" },
          { "name": "devicemonitor", "version": "1.1.0" },
          { "name": "gui", "version": "3.2.0" },
          { "name": "integration", "version": "3.2.2" },
          { "name": "inventory", "version": "4.1.0" },
          { "name": "inventory-enterprise", "version": "4.1.0" },
          { "name": "iot-manager", "version": "1.0.1" },
          { "name": "mender", "version": "3.2.1" },
          { "name": "mender-artifact", "version": "3.7.1" },
          { "name": "mender-binary-delta", "version": "1.3.0" },
          { "name": "mender-cli", "version": "1.7.0" },
          { "name": "mender-configure-module", "version": "1.0.3" },
          { "name": "mender-connect", "version": "2.0.1" },
          { "name": "mender-convert", "version": "2.6.2" },
          { "name": "monitor-client", "version": "1.1.0" },
          { "name": "mtls-ambassador", "version": "1.0.1" },
          { "name": "tenantadm", "version": "3.3.0" },
          { "name": "useradm", "version": "1.17.0" },
          { "name": "useradm-enterprise", "version": "1.17.0" },
          { "name": "workflows", "version": "2.2.0" },
          { "name": "workflows-enterprise", "version": "2.2.0" }
        ]
      },
      "3.2.1": {
        "release_date": "2022-02-02",
        "release": "3.2.1",
        "repos": [
          { "name": "auditlogs", "version": "3.0.0" },
          { "name": "create-artifact-worker", "version": "1.1.0" },
          { "name": "deployments", "version": "4.1.0" },
          { "name": "deployments-enterprise", "version": "4.1.0" },
          { "name": "deviceauth", "version": "3.2.0" },
          { "name": "deviceauth-enterprise", "version": "3.2.0" },
          { "name": "deviceconfig", "version": "1.2.0" },
          { "name": "deviceconnect", "version": "1.3.0" },
          { "name": "devicemonitor", "version": "1.1.0" },
          { "name": "gui", "version": "3.2.0" },
          { "name": "integration", "version": "3.2.1" },
          { "name": "inventory", "version": "4.1.0" },
          { "name": "inventory-enterprise", "version": "4.1.0" },
          { "name": "iot-manager", "version": "1.0.0" },
          { "name": "mender", "version": "3.2.1" },
          { "name": "mender-artifact", "version": "3.7.0" },
          { "name": "mender-binary-delta", "version": "1.3.0" },
          { "name": "mender-cli", "version": "1.7.0" },
          { "name": "mender-configure-module", "version": "1.0.3" },
          { "name": "mender-connect", "version": "2.0.1" },
          { "name": "mender-convert", "version": "2.6.2" },
          { "name": "monitor-client", "version": "1.1.0" },
          { "name": "mtls-ambassador", "version": "1.0.1" },
          { "name": "tenantadm", "version": "3.3.0" },
          { "name": "useradm", "version": "1.17.0" },
          { "name": "useradm-enterprise", "version": "1.17.0" },
          { "name": "workflows", "version": "2.2.0" },
          { "name": "workflows-enterprise", "version": "2.2.0" }
        ]
      },
      "3.2.0": {
        "release_date": "2022-01-24",
        "release": "3.2.0",
        "repos": [
          { "name": "auditlogs", "version": "3.0.0" },
          { "name": "create-artifact-worker", "version": "1.1.0" },
          { "name": "deployments", "version": "4.1.0" },
          { "name": "deployments-enterprise", "version": "4.1.0" },
          { "name": "deviceauth", "version": "3.2.0" },
          { "name": "deviceauth-enterprise", "version": "3.2.0" },
          { "name": "deviceconfig", "version": "1.2.0" },
          { "name": "deviceconnect", "version": "1.3.0" },
          { "name": "devicemonitor", "version": "1.1.0" },
          { "name": "gui", "version": "3.2.0" },
          { "name": "integration", "version": "3.2.0" },
          { "name": "inventory", "version": "4.1.0" },
          { "name": "inventory-enterprise", "version": "4.1.0" },
          { "name": "iot-manager", "version": "1.0.0" },
          { "name": "mender", "version": "3.2.0" },
          { "name": "mender-artifact", "version": "3.7.0" },
          { "name": "mender-binary-delta", "version": "1.3.0" },
          { "name": "mender-cli", "version": "1.7.0" },
          { "name": "mender-configure-module", "version": "1.0.3" },
          { "name": "mender-connect", "version": "2.0.0" },
          { "name": "mender-convert", "version": "2.6.1" },
          { "name": "monitor-client", "version": "1.1.0" },
          { "name": "mtls-ambassador", "version": "1.0.1" },
          { "name": "tenantadm", "version": "3.3.0" },
          { "name": "useradm", "version": "1.17.0" },
          { "name": "useradm-enterprise", "version": "1.17.0" },
          { "name": "workflows", "version": "2.2.0" },
          { "name": "workflows-enterprise", "version": "2.2.0" }
        ]
      }
    },
    "3.1": {
      "3.1.1": {
        "release_date": "2022-02-09",
        "release": "3.1.1",
        "repos": [
          { "name": "auditlogs", "version": "2.0.0" },
          { "name": "create-artifact-worker", "version": "1.0.3" },
          { "name": "deployments", "version": "4.0.1" },
          { "name": "deployments-enterprise", "version": "4.0.1" },
          { "name": "deviceauth", "version": "3.1.0" },
          { "name": "deviceconfig", "version": "1.1.0" },
          { "name": "deviceconnect", "version": "1.2.1" },
          { "name": "devicemonitor", "version": "1.0.1" },
          { "name": "gui", "version": "3.1.1" },
          { "name": "integration", "version": "3.1.1" },
          { "name": "inventory", "version": "4.0.1" },
          { "name": "inventory-enterprise", "version": "4.0.1" },
          { "name": "mender", "version": "3.1.1" },
          { "name": "mender-artifact", "version": "3.6.1" },
          { "name": "mender-cli", "version": "1.7.0" },
          { "name": "mender-connect", "version": "1.2.1" },
          { "name": "monitor-client", "version": "1.0.1" },
          { "name": "mtls-ambassador", "version": "1.0.1" },
          { "name": "tenantadm", "version": "3.3.0" },
          { "name": "useradm", "version": "1.16.0" },
          { "name": "useradm-enterprise", "version": "1.16.0" },
          { "name": "workflows", "version": "2.1.0" },
          { "name": "workflows-enterprise", "version": "2.1.0" }
        ]
      },
      "3.1.0": {
        "release_date": "2021-09-28",
        "release": "3.1.0",
        "repos": [
          { "name": "auditlogs", "version": "2.0.0" },
          { "name": "create-artifact-worker", "version": "1.0.2" },
          { "name": "deployments", "version": "4.0.0" },
          { "name": "deployments-enterprise", "version": "4.0.0" },
          { "name": "deviceauth", "version": "3.1.0" },
          { "name": "deviceconfig", "version": "1.1.0" },
          { "name": "deviceconnect", "version": "1.2.1" },
          { "name": "devicemonitor", "version": "1.0.0" },
          { "name": "gui", "version": "3.1.0" },
          { "name": "integration", "version": "3.1.0" },
          { "name": "inventory", "version": "4.0.0" },
          { "name": "inventory-enterprise", "version": "4.0.0" },
          { "name": "mender", "version": "3.1.0" },
          { "name": "mender-artifact", "version": "3.6.1" },
          { "name": "mender-cli", "version": "1.7.0" },
          { "name": "mender-connect", "version": "1.2.0" },
          { "name": "monitor-client", "version": "1.0.0" },
          { "name": "mtls-ambassador", "version": "1.0.0" },
          { "name": "tenantadm", "version": "3.3.0" },
          { "name": "useradm", "version": "1.16.0" },
          { "name": "useradm-enterprise", "version": "1.16.0" },
          { "name": "workflows", "version": "2.1.0" },
          { "name": "workflows-enterprise", "version": "2.1.0" }
        ]
      }
    },
    "3.0": {
      "supported_until": "2022-07",
      "3.0.2": {
        "release_date": "2022-02-09",
        "release": "3.0.2",
        "repos": [
          { "name": "auditlogs", "version": "1.2.0" },
          { "name": "create-artifact-worker", "version": "1.0.3" },
          { "name": "deployments", "version": "3.0.1" },
          { "name": "deployments-enterprise", "version": "3.0.1" },
          { "name": "deviceauth", "version": "3.0.0" },
          { "name": "deviceconfig", "version": "1.1.0" },
          { "name": "deviceconnect", "version": "1.2.1" },
          { "name": "gui", "version": "3.0.2" },
          { "name": "integration", "version": "3.0.2" },
          { "name": "inventory", "version": "3.0.1" },
          { "name": "inventory-enterprise", "version": "3.0.1" },
          { "name": "mender", "version": "3.0.2" },
          { "name": "mender-artifact", "version": "3.6.1" },
          { "name": "mender-cli", "version": "1.7.0" },
          { "name": "mender-connect", "version": "1.2.1" },
          { "name": "mtls-ambassador", "version": "1.0.1" },
          { "name": "tenantadm", "version": "3.2.0" },
          { "name": "useradm", "version": "1.15.0" },
          { "name": "useradm-enterprise", "version": "1.15.0" },
          { "name": "workflows", "version": "2.0.0" },
          { "name": "workflows-enterprise", "version": "2.0.0" }
        ]
      },
      "3.0.1": {
        "release_date": "2021-09-29",
        "release": "3.0.1",
        "repos": [
          { "name": "auditlogs", "version": "1.2.0" },
          { "name": "create-artifact-worker", "version": "1.0.2" },
          { "name": "deployments", "version": "3.0.1" },
          { "name": "deployments-enterprise", "version": "3.0.1" },
          { "name": "deviceauth", "version": "3.0.0" },
          { "name": "deviceconfig", "version": "1.1.0" },
          { "name": "deviceconnect", "version": "1.2.1" },
          { "name": "gui", "version": "3.0.1" },
          { "name": "integration", "version": "3.0.1" },
          { "name": "inventory", "version": "3.0.0" },
          { "name": "inventory-enterprise", "version": "3.0.0" },
          { "name": "mender", "version": "3.0.1" },
          { "name": "mender-artifact", "version": "3.6.1" },
          { "name": "mender-cli", "version": "1.7.0" },
          { "name": "mender-connect", "version": "1.2.0" },
          { "name": "mtls-ambassador", "version": "1.0.0" },
          { "name": "tenantadm", "version": "3.2.0" },
          { "name": "useradm", "version": "1.15.0" },
          { "name": "useradm-enterprise", "version": "1.15.0" },
          { "name": "workflows", "version": "2.0.0" },
          { "name": "workflows-enterprise", "version": "2.0.0" }
        ]
      },
      "3.0.0": {
        "release_date": "2021-07-13",
        "release": "3.0.0",
        "repos": [
          { "name": "auditlogs", "version": "1.2.0" },
          { "name": "create-artifact-worker", "version": "1.0.2" },
          { "name": "deployments", "version": "3.0.0" },
          { "name": "deployments-enterprise", "version": "3.0.0" },
          { "name": "deviceauth", "version": "3.0.0" },
          { "name": "deviceconfig", "version": "1.1.0" },
          { "name": "deviceconnect", "version": "1.2.0" },
          { "name": "gui", "version": "3.0.0" },
          { "name": "integration", "version": "3.0.0" },
          { "name": "inventory", "version": "3.0.0" },
          { "name": "inventory-enterprise", "version": "3.0.0" },
          { "name": "mender", "version": "3.0.0" },
          { "name": "mender-artifact", "version": "3.6.0" },
          { "name": "mender-cli", "version": "1.7.0" },
          { "name": "mender-connect", "version": "1.2.0" },
          { "name": "mtls-ambassador", "version": "1.0.0" },
          { "name": "tenantadm", "version": "3.2.0" },
          { "name": "useradm", "version": "1.15.0" },
          { "name": "useradm-enterprise", "version": "1.15.0" },
          { "name": "workflows", "version": "2.0.0" },
          { "name": "workflows-enterprise", "version": "2.0.0" }
        ]
      }
    }
  },
  "saas": [
    { "tag": "saas-v2022.03.10", "date": "2022-03-09" },
    { "tag": "saas-v2022.01.24", "date": "2022-01-22" },
    { "tag": "saas-v2021.11.02", "date": "2021-11-02" },
    { "tag": "saas-v2021.01.14", "date": "2021-01-14" },
    { "tag": "saas-v2020.12.02", "date": "2020-12-02" },
    { "tag": "saas-v2020.11.19", "date": "2020-11-19" },
    { "tag": "saas-v2020.10.14", "date": "2020-10-14" },
    { "tag": "saas-v2020.09.25", "date": "2020-09-24" },
    { "tag": "saas-v2020.09.09", "date": "2020-09-08" },
    { "tag": "saas-v2020.08.11", "date": "2020-08-11" },
    { "tag": "saas-v2020.07.31", "date": "2020-07-31" },
    { "tag": "saas-v2020.07.22", "date": "2020-07-22" },
    { "tag": "saas-v2020.07.09", "date": "2020-07-09" }
  ]
}
`

func TestSuggestCherryPicks(t *testing.T) {

	gitHubOrg := "mendersoftware"
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(versionsResponse))
	}))
	versionsUrl = server.URL

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
Hello :smiley_cat: This PR contains changelog entries. Please, verify the need of backporting it to the following release branches:
2.3.x (release 3.4.x)
2.2.x (release 3.3.x)
2.0.x (release 3.0.x)
`),
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
Hello :smiley_cat: This PR contains changelog entries. Please, verify the need of backporting it to the following release branches:
2.1.x (release 3.4.x)
2.0.x (release 3.3.x)
1.2.x (release 3.0.x)
`),
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
Hello :smiley_cat: This PR contains changelog entries. Please, verify the need of backporting it to the following release branches:
2.1.x (release 3.4.x)
2.0.x (release 3.3.x)
1.2.x (release 3.0.x)
`),
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
Hello :smiley_cat: This PR contains changelog entries. Please, verify the need of backporting it to the following release branches:
3.4.x (release 3.4.x)
3.3.x (release 3.3.x)
3.0.x (release 3.0.x)
`),
			},
		},
	}

	tmpdir, err := os.MkdirTemp("", "*")
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
			log.Infof(" TEST: %s", name)
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
		"main": {
			body: `cherry-pick to:
		* main`,
			expected: []string{"main"},
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
		* main
		* hosted
		* example-branch`,
			expected: []string{"master", "main", "hosted", "example-branch"},
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
		"main": {
			body:     "cherry-pick to: `main`",
			expected: []string{"main"},
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
