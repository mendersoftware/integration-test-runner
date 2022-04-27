package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"

	clientgithub "github.com/mendersoftware/integration-test-runner/client/github"
	"github.com/mendersoftware/integration-test-runner/git"
)

const (
	footer = `Changelog: All
Ticket: None`
	commentErrorPrefix = "I did my very best, but:\n"
)

func conventionalComittifyDependabotPr(
	log *logrus.Entry,
	comment *github.IssueCommentEvent,
	pr *github.PullRequest,
	conf *config,
	body string,
	githubClient clientgithub.Client,
) error {
	err := attemptConventionalComittifyDependabotPr(log, pr, body)

	if err == nil {
		return nil
	}

	commentBody := commentErrorPrefix + err.Error()
	if err := githubClient.CreateComment(
		context.Background(),
		conf.githubOrganization,
		comment.GetRepo().GetName(),
		pr.GetNumber(),
		&github.IssueComment{Body: &commentBody}); err != nil {
		log.Infof("Failed to comment on the pr: %v, Error: %s", pr, err.Error())
	}

	return err
}

func attemptConventionalComittifyDependabotPr(
	log *logrus.Entry,
	pr *github.PullRequest,
	body string,
) error {

	typeKeyword, err := getTypeKeyword(body)
	if err != nil {
		return err
	}

	// take message, and conventional committify it
	headBranch := pr.GetHead().GetRef()
	sshCloneUrl := pr.GetHead().GetRepo().GetSSHURL()
	state, err := git.Commands(
		git.Command("clone", "--branch", headBranch, "--single-branch", sshCloneUrl, "."),
	)
	defer state.Cleanup()

	if err != nil {
		return fmt.Errorf("could not clone branch %s from %s, with error:\n%w",
			headBranch, sshCloneUrl, err)
	}

	messageBytes, err := git.Command("--no-pager", "show", "--no-patch", "--format=%B", "HEAD").
		With(state).CombinedOutput()
	if err != nil {
		return fmt.Errorf("could not retrieve last commit message with error:\n%w", err)
	}

	message := string(messageBytes)

	newMessage := conventionalComittifyDependabotMessage(message, typeKeyword)

	if err := git.CommandsWithState(
		state,
		git.Command("commit", "--amend", "-m", newMessage),
		git.Command("push", "--force"),
	); err != nil {
		return fmt.Errorf("could not amend and push with error:\n%w", err)
	}

	return nil
}

func conventionalComittifyDependabotMessage(message string, typeKeyword string) string {
	message = strings.TrimSpace(message)
	message = strings.TrimPrefix(message, "Changelog:All: ")
	message = strings.TrimPrefix(message, "chore: ")
	message = typeKeyword + ": " + message

	fi := strings.Index(message, "Signed-off-by")
	if fi < 0 {
		fi = len(message)
	}
	message = message[:fi] + footer + "\n" + message[fi:]
	return strings.TrimSpace(message)
}

func getTypeKeyword(s string) (string, error) {
	r := regexp.MustCompile(commandConventionalCommit +
		`(?::\s*|\s+)(fix|feat|([[:word:]]+))\s*$`)
	matches := r.FindStringSubmatch(s)
	if matches == nil {
		return "", fmt.Errorf("could not parse the body for some reason")
	} else if matches[2] != "" {
		return "", fmt.Errorf("type keyword %s not allowed", matches[2])
	}
	return matches[1], nil
}
