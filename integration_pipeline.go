package main

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"text/template"

	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	clientgitlab "github.com/mendersoftware/integration-test-runner/client/gitlab"
)

const integrationPipelinePath = "Northern.tech/Mender/integration"

func getIntegrationBuild(
	log *logrus.Entry,
	conf *config,
	pr *github.PullRequestEvent,
) buildOptions {

	repo := pr.GetRepo().GetName()
	commitSHA := pr.PullRequest.Base.GetSHA()

	//GetLabel returns "mendersoftware:master", we just want the branch
	baseBranch := strings.Split(pr.PullRequest.Base.GetLabel(), ":")[1]

	build := buildOptions{
		pr:         strconv.Itoa(pr.GetNumber()),
		repo:       repo,
		baseBranch: baseBranch,
		commitSHA:  commitSHA,
	}

	return build
}

func triggerIntegrationBuild(
	log *logrus.Entry,
	conf *config,
	build *buildOptions,
	pr *github.PullRequestEvent,
	buildOptions *BuildOptions,
) error {
	gitlabIntegration, err := clientgitlab.NewGitLabClient(
		conf.gitlabToken,
		conf.gitlabBaseURL,
		conf.dryRunMode,
	)
	if err != nil {
		return err
	}

	buildParameters, err := getIntegrationBuildParameters(log, conf, build, buildOptions)
	if err != nil {
		return err
	}

	// first stop old pipelines with the same buildParameters
	stopStalePipelines(integrationPipelinePath, log, gitlabIntegration, buildParameters)

	// trigger the new pipeline
	ref := "pr_" + strconv.Itoa(pr.GetNumber()) + "_protected"
	opt := &gitlab.CreatePipelineOptions{
		Ref:       &ref,
		Variables: &buildParameters,
	}

	variablesString := ""
	for _, variable := range *opt.Variables {
		variablesString += *variable.Key + ":" + *variable.Value + ", "
	}
	log.Infof(
		"Creating pipeline in project %s:%s with variables: %s",
		integrationPipelinePath,
		*opt.Ref,
		variablesString,
	)

	pipeline, err := gitlabIntegration.CreatePipeline(integrationPipelinePath, opt)
	if err != nil {
		log.Errorf("Could not create pipeline: %s", err.Error())
		return err
	}
	log.Infof("Created pipeline: %s", pipeline.WebURL)

	// Add the build variable matrix to the pipeline comment under a
	// drop-down tab
	// nolint:lll
	tmplString := `
Hello :smiley_cat: I created a pipeline for you here: [Pipeline-{{.Pipeline.ID}}]({{.Pipeline.WebURL}})

<details>
    <summary>Build Configuration Matrix</summary><p>

| Key   | Value |
| ----- | ----- |
{{range $i, $var := .BuildVars}}{{if $var.Value}}| {{$var.Key}} | {{$var.Value}} |{{printf "\n"}}{{end}}{{end}}

 </p></details>
`
	tmpl, err := template.New("Main").Parse(tmplString)
	if err != nil {
		log.Errorf(
			"Failed to parse the build matrix template. Should never happen! Error: %s\n",
			err.Error(),
		)
		return err
	}
	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, struct {
		BuildVars []*gitlab.PipelineVariableOptions
		Pipeline  *gitlab.Pipeline
	}{
		BuildVars: filterOutEmptyVariables(*opt.Variables),
		Pipeline:  pipeline,
	}); err != nil {
		log.Errorf("Failed to execute the build matrix template. Error: %s\n", err.Error())
		return err
	}

	// Comment with a pipeline-link on the PR
	commentBody := buf.String()
	comment := github.IssueComment{
		Body: &commentBody,
	}

	err = githubClient.CreateComment(context.Background(),
		conf.githubOrganization, pr.GetRepo().GetName(), pr.GetNumber(), &comment)
	if err != nil {
		log.Infof("Failed to comment on the pr: %v, Error: %s", pr, err.Error())
	}

	return err
}

func getIntegrationBuildParameters(
	log *logrus.Entry,
	conf *config,
	build *buildOptions,
	buildOptions *BuildOptions,
) ([]*gitlab.PipelineVariableOptions, error) {
	readHead := "pull/" + build.pr + "/head"
	var buildParameters []*gitlab.PipelineVariableOptions

	runIntegrationTests := "true"
	runIntegrationTestsKey := "RUN_TESTS_FULL_INTEGRATION"
	buildParameters = append(
		buildParameters,
		&gitlab.PipelineVariableOptions{Key: &runIntegrationTestsKey, Value: &runIntegrationTests},
	)

	buildRepoKey := repoToBuildParameter(build.repo)
	buildParameters = append(buildParameters,
		&gitlab.PipelineVariableOptions{
			Key:   &buildRepoKey,
			Value: &readHead,
		})

	return buildParameters, nil

}
