package main

import (
	"bytes"
	"context"
	"reflect"
	"sort"
	"text/template"

	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	clientgitlab "github.com/mendersoftware/integration-test-runner/client/gitlab"
)

func say(
	ctx context.Context,
	tmplString string,
	data interface{},
	log *logrus.Entry,
	conf *config,
	pr *github.PullRequestEvent,
) error {
	tmpl, err := template.New("Main").Parse(tmplString)
	if err != nil {
		log.Errorf(
			"Failed to parse the build matrix template. Should never happen! Error: %s\n",
			err.Error(),
		)
	}
	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, data); err != nil {
		log.Errorf("Failed to execute the build matrix template. Error: %s\n", err.Error())
	}

	// Comment with a pipeline-link on the PR
	commentBody := buf.String()
	comment := github.IssueComment{
		Body: &commentBody,
	}

	err = githubClient.CreateComment(ctx,
		conf.githubOrganization, pr.GetRepo().GetName(), pr.GetNumber(), &comment)
	if err != nil {
		log.Infof("Failed to comment on the pr: %v, Error: %s", pr, err.Error())
	}

	return err
}

func filterOutEmptyVariables(
	optionsIn []*gitlab.PipelineVariableOptions,
) []*gitlab.PipelineVariableOptions {
	var optionsOut []*gitlab.PipelineVariableOptions
	for _, option := range optionsIn {
		if *option.Value != "" {
			optionsOut = append(optionsOut, option)
		}
	}
	return optionsOut
}

func stopStalePipelines(
	pipelinePath string,
	log *logrus.Entry,
	client clientgitlab.Client,
	vars []*gitlab.PipelineVariableOptions,
) {

	sort.SliceStable(vars, func(i, j int) bool {
		return *vars[i].Key < *vars[j].Key
	})

	username := githubBotName
	status := gitlab.Pending
	opt := &gitlab.ListProjectPipelinesOptions{
		Username: &username,
		Status:   &status,
	}

	pipelinesPending, err := client.ListProjectPipelines(clientPipelinePath, opt)
	if err != nil {
		log.Errorf("stopStalePipelines: Could not list pending pipelines: %s", err.Error())
	}

	status = gitlab.Running
	opt = &gitlab.ListProjectPipelinesOptions{
		Username: &username,
		Status:   &status,
	}

	pipelinesRunning, err := client.ListProjectPipelines(clientPipelinePath, opt)
	if err != nil {
		log.Errorf("stopStalePipelines: Could not list running pipelines: %s", err.Error())
	}

	for _, pipeline := range append(pipelinesPending, pipelinesRunning...) {

		variables, err := client.GetPipelineVariables(clientPipelinePath, pipeline.ID)
		if err != nil {
			log.Errorf("stopStalePipelines: Could not get variables for pipeline: %s", err.Error())
			continue
		}

		sort.SliceStable(variables, func(i, j int) bool {
			return variables[i].Key < variables[j].Key
		})

		if reflect.DeepEqual(vars, variables) {
			log.Infof("Cancelling stale pipeline %d, url: %s", pipeline.ID, pipeline.WebURL)

			err := client.CancelPipelineBuild(clientPipelinePath, pipeline.ID)
			if err != nil {
				log.Errorf("stopStalePipelines: Could not cancel pipeline: %s", err.Error())
			}

		}

	}
}
