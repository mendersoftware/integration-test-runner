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

	if len(conf.reposSyncList) == 0 || isRepoManaged(repoName, conf.reposSyncList) {
		log.Debugf("Syncing repo %s/%s", repoOrg, repoName)
		err := syncRemoteRef(log, repoOrg, repoName, refName, conf)
		if err != nil {
			log.Errorf("Could not sync branch: %s", err.Error())
			return err
		}
	}
	return nil
}

func isRepoManaged(repo string, reposList []string) bool {
	for _, r := range reposList {
		if r == repo {
			return true
		}
	}
	return false
}
