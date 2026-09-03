package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v28/github"

	clientgithub "github.com/mendersoftware/integration-test-runner/client/github"
)

func processGitHubPush(
	ctx *gin.Context,
	push *github.PushEvent,
	githubClient clientgithub.Client,
	conf *config,
) error {
	log := getCustomLoggerFromContext(ctx)

	repoName := push.GetRepo().GetName()
	repoOrg := push.GetRepo().GetOrganization()
	refName := push.GetRef()

	log.Debugf("Got push event :: repo %s :: ref %s", repoName, refName)

	log.Debugf("Syncing repo %s/%s", repoOrg, repoName)
	if err := syncRemoteRef(log, repoOrg, repoName, refName, conf); err != nil {
		log.Errorf("Could not sync branch: %s", err.Error())
		return err
	}
	return nil
}
