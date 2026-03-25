package main

import (
	"fmt"

	"github.com/google/go-github/v28/github"
)

type gitProtocol int

const (
	gitProtocolSSH gitProtocol = iota
	gitProtocolHTTP
)

func getRemoteURLGitHub(proto gitProtocol, org, repo string) string {
	if proto == gitProtocolSSH {
		return "git@github.com:/" + org + "/" + repo + ".git"
	} else if proto == gitProtocolHTTP {
		return "https://github.com/" + org + "/" + repo
	}
	return ""
}

func getGitLabProjectPath(org, repo string) (string, error) {
	// By default, the GitLab project is Northern.tech/<group>/<repo>
	group, ok := gitHubOrganizationToGitLabGroup[org]
	if !ok {
		return "", fmt.Errorf("Unrecognized organization %q", org)
	}
	path := "Northern.tech/" + group + "/" + repo

	// Override for some specific repos that have a custom GitLab group/project
	if v, ok := gitHubRepoToGitLabProjectCustom[repo]; ok {
		path = v
	}
	return path, nil
}

func getRemoteURLGitLab(org, repo string) (string, error) {
	path, err := getGitLabProjectPath(org, repo)
	if err != nil {
		return "", err
	}
	return "git@gitlab.com:" + path, nil
}

func getGitHubOrganization(webhookType string, webhookEvent interface{}) (string, error) {
	switch webhookType {
	case "pull_request":
		pr := webhookEvent.(*github.PullRequestEvent)
		return pr.GetOrganization().GetLogin(), nil
	case "push":
		push := webhookEvent.(*github.PushEvent)
		return push.GetRepo().GetOrganization(), nil
	case "issue_comment":
		comment := webhookEvent.(*github.IssueCommentEvent)
		return comment.GetRepo().GetOwner().GetLogin(), nil
	}
	return "", fmt.Errorf(
		"getGitHubOrganization cannot get organization from webhook type %q",
		webhookType,
	)

}
