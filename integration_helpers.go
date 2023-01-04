package main

import (
	"fmt"
	"io"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

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

func getChangelogText(repo, versionRange string, conf *config) (stdout,
	stderr string, retErr error) {
	c := exec.Command(
		path.Join(conf.integrationDirectory,
			"extra/changelog-generator/changelog-generator"),
		"--repo",
		"--sort-changelog",
		"--query-github",
		"--github-repo", repo,
		versionRange,
	)
	stdout, stderr, retErr = getBothStdoutAndStderr(c)

	// Replace commit IDs in warnings, to avoid multiple postings when they
	// change.
	matcher, err := regexp.Compile("Commit [0-9a-f]{40} had a number")
	if err == nil {
		stderr = matcher.ReplaceAllString(stderr,
			"One commit had a number")
	}

	return stdout, stderr, retErr
}

// All this because exec doesn't have a SplitOutputs() function...
func getBothStdoutAndStderr(c *exec.Cmd) (stdout, stderr string, retErr error) {
	outPipe, err := c.StdoutPipe()
	if err != nil {
		return "", "", errors.Wrap(err, "getBothStdoutAndStderr: StdoutPipe")
	}
	errPipe, err := c.StderrPipe()
	if err != nil {
		return "", "", errors.Wrap(err, "getBothStdoutAndStderr: StderrPipe")
	}

	err = c.Start()
	if err != nil {
		return "", "", errors.Wrap(err, "getBothStdoutAndStderr: Start")
	}
	defer func() {
		err := c.Wait()
		if err != nil && retErr == nil {
			retErr = err
		}
	}()

	type stringAndError struct {
		string string
		err    error
	}

	outChan := make(chan stringAndError)
	errChan := make(chan stringAndError)

	go func() {
		var s strings.Builder
		_, err := io.Copy(&s, outPipe)
		outChan <- stringAndError{
			s.String(),
			err,
		}
	}()
	go func() {
		var s strings.Builder
		_, err := io.Copy(&s, errPipe)
		errChan <- stringAndError{
			s.String(),
			err,
		}
	}()

	stdoutAndError := <-outChan
	if stdoutAndError.err != nil {
		return "", "", errors.Wrap(err, "getBothStdoutAndStderr: outChan")
	}
	stderrAndError := <-errChan
	if stderrAndError.err != nil {
		return "", "", errors.Wrap(err, "getBothStdoutAndStderr: errChan")
	}

	return stdoutAndError.string, stderrAndError.string, nil
}
