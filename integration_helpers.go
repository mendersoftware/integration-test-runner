package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/mendersoftware/integration-test-runner/git"
)

var gitUpdateMutex = &sync.Mutex{}

func updateIntegrationRepo(conf *config) error {
	gitUpdateMutex.Lock()
	defer gitUpdateMutex.Unlock()

	gitcmd := git.Command("pull", "--rebase", "origin")
	gitcmd.Dir = conf.integrationDirectory

	// timeout and kill process after gitOperationTimeout seconds
	t := time.AfterFunc(gitOperationTimeout*time.Second, func() {
		_ = gitcmd.Process.Kill()
	})
	defer t.Stop()

	if err := gitcmd.Run(); err != nil {
		return fmt.Errorf("failed to 'git pull' integration folder: %s", err.Error())
	}
	return nil
}
