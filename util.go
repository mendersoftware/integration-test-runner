package main

import (
	"fmt"
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

func getRemoteURLGitLab(org, repo string) (string, error) {
	// By default, the GitLab project is Northern.tech/<group>/<repo>
	group, ok := gitHubOrganizationToGitLabGroup[org]
	if !ok {
		return "", fmt.Errorf("Unrecognized organization %q", org)
	}
	remoteURL := "git@gitlab.com:Northern.tech/" + group + "/" + repo

	// Override for some specific repos have custom GitLab group/project
	if v, ok := gitHubRepoToGitLabProjectCustom[repo]; ok {
		remoteURL = "git@gitlab.com:" + v
	}
	return remoteURL, nil
}
