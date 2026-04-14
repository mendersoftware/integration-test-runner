package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	clientgitlab "github.com/mendersoftware/integration-test-runner/client/gitlab"
)

const (
	reviewDeployJobName = "review:deploy"
)

func getReviewAppSlug(prNumber int) string {
	return fmt.Sprintf("pr-%d", prNumber)
}

func getReviewAppURL(slug, domain string) string {
	return fmt.Sprintf("https://%s.%s/", slug, domain)
}

// findAndPlayJob finds the latest pipeline for the given ref, locates a job
// by name, and plays it. Returns the played job or an error.
func findAndPlayJob(
	log *logrus.Entry,
	client clientgitlab.Client,
	projectPath string,
	ref string,
	jobName string,
	jobVars []*gitlab.JobVariableOptions,
) (*gitlab.Job, error) {
	// Find the latest pipeline for this ref
	pipelines, err := client.ListProjectPipelines(projectPath, &gitlab.ListProjectPipelinesOptions{
		Ref: &ref,
		ListOptions: gitlab.ListOptions{
			PerPage: 1,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pipelines for ref %s: %w", ref, err)
	}
	if len(pipelines) == 0 {
		return nil, fmt.Errorf("no pipelines found for ref %s in project %s", ref, projectPath)
	}
	latestPipeline := pipelines[0]
	log.Infof("Found latest pipeline %d for ref %s in project %s", latestPipeline.ID, ref, projectPath)

	// List jobs in the pipeline and find the target job
	jobs, err := client.ListPipelineJobs(projectPath, latestPipeline.ID, &gitlab.ListJobsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs for pipeline %d: %w", latestPipeline.ID, err)
	}

	var targetJob *gitlab.Job
	for _, job := range jobs {
		if job.Name == jobName {
			targetJob = job
			break
		}
	}
	if targetJob == nil {
		return nil, fmt.Errorf("job %q not found in pipeline %d", jobName, latestPipeline.ID)
	}

	if targetJob.Status != "manual" {
		return nil, fmt.Errorf(
			"job %q in pipeline %d has status %q (expected \"manual\"); builds may still be running",
			jobName, latestPipeline.ID, targetJob.Status,
		)
	}

	log.Infof("Playing job %q (ID: %d) in pipeline %d", jobName, targetJob.ID, latestPipeline.ID)

	playedJob, err := client.PlayJob(projectPath, targetJob.ID, &gitlab.PlayJobOptions{
		JobVariablesAttributes: &jobVars,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to play job %q (ID: %d): %w", jobName, targetJob.ID, err)
	}

	return playedJob, nil
}

func triggerReviewDeploy(
	log *logrus.Entry,
	conf *config,
	pr *github.PullRequestEvent,
	sender string,
) error {
	repoName := pr.GetRepo().GetName()
	domain, ok := gitHubRepoToReviewAppDomain[repoName]
	if !ok {
		commentBody := fmt.Sprintf(
			"Review app deployment is not supported for repository `%s`.",
			repoName,
		)
		comment := github.IssueComment{Body: &commentBody}
		if err := githubClient.CreateComment(context.Background(),
			conf.githubOrganization, repoName, pr.GetNumber(), &comment); err != nil {
			log.Infof("Failed to comment on the pr: %v, Error: %s", pr, err.Error())
		}
		return fmt.Errorf("no review app domain configured for repo %q", repoName)
	}

	projectPath, err := getGitLabProjectPath(conf.githubOrganization, repoName)
	if err != nil {
		return err
	}

	prNumber := pr.GetNumber()
	slug := getReviewAppSlug(prNumber)

	client, err := clientgitlab.NewGitLabClient(
		conf.gitlabToken,
		conf.gitlabBaseURL,
		conf.dryRunMode,
	)
	if err != nil {
		return err
	}

	ref := "pr_" + strconv.Itoa(prNumber)

	requestedByKey := "REVIEW_REQUESTED_BY"
	jobVars := []*gitlab.JobVariableOptions{
		{Key: &requestedByKey, Value: &sender},
	}

	job, err := findAndPlayJob(log, client, projectPath, ref, reviewDeployJobName, jobVars)
	if err != nil {
		return err
	}

	log.Infof("Started review deploy job: %s", job.WebURL)

	reviewURL := getReviewAppURL(slug, domain)
	commentBody := fmt.Sprintf(
		"Review app deploy triggered: [%s #%d](%s)\n\nReview app will be available at: %s",
		reviewDeployJobName,
		job.ID,
		job.WebURL,
		reviewURL,
	)
	comment := github.IssueComment{
		Body: &commentBody,
	}

	if err := githubClient.CreateComment(context.Background(),
		conf.githubOrganization, repoName, pr.GetNumber(), &comment); err != nil {
		log.Infof("Failed to comment on the pr: %v, Error: %s", pr, err.Error())
	}

	return nil
}

func triggerReviewE2E(
	log *logrus.Entry,
	conf *config,
	pr *github.PullRequestEvent,
) error {
	repoName := pr.GetRepo().GetName()
	domain, ok := gitHubRepoToReviewAppDomain[repoName]
	if !ok {
		commentBody := fmt.Sprintf(
			"Review app e2e tests are not supported for repository `%s`.",
			repoName,
		)
		comment := github.IssueComment{Body: &commentBody}
		if err := githubClient.CreateComment(context.Background(),
			conf.githubOrganization, repoName, pr.GetNumber(), &comment); err != nil {
			log.Infof("Failed to comment on the pr: %v, Error: %s", pr, err.Error())
		}
		return fmt.Errorf("no review app domain configured for repo %q", repoName)
	}

	projectPath, err := getGitLabProjectPath(conf.githubOrganization, repoName)
	if err != nil {
		return err
	}

	prNumber := pr.GetNumber()
	slug := getReviewAppSlug(prNumber)
	baseURL := getReviewAppURL(slug, domain)
	ref := "pr_" + strconv.Itoa(prNumber)

	var variables []*gitlab.PipelineVariableOptions

	runReviewE2EKey := "RUN_REVIEW_E2E"
	runReviewE2EVal := "true"
	variables = append(variables, &gitlab.PipelineVariableOptions{
		Key:   &runReviewE2EKey,
		Value: &runReviewE2EVal,
	})

	baseURLKey := "BASE_URL"
	variables = append(variables, &gitlab.PipelineVariableOptions{
		Key:   &baseURLKey,
		Value: &baseURL,
	})

	client, err := clientgitlab.NewGitLabClient(
		conf.gitlabToken,
		conf.gitlabBaseURL,
		conf.dryRunMode,
	)
	if err != nil {
		return err
	}

	opt := &gitlab.CreatePipelineOptions{
		Ref:       &ref,
		Variables: &variables,
	}

	variablesString := ""
	for _, variable := range variables {
		variablesString += *variable.Key + ":" + *variable.Value + ", "
	}
	log.Infof(
		"Creating review e2e pipeline in project %s:%s with variables: %s",
		projectPath,
		ref,
		variablesString,
	)

	pipeline, err := client.CreatePipeline(projectPath, opt)
	if err != nil {
		log.Errorf("Could not create review e2e pipeline: %s", err.Error())
		return err
	}
	log.Infof("Created review e2e pipeline: %s", pipeline.WebURL)

	commentBody := fmt.Sprintf(
		"Review app e2e test pipeline created: [Pipeline-%d](%s)",
		pipeline.ID,
		pipeline.WebURL,
	)
	comment := github.IssueComment{
		Body: &commentBody,
	}

	if err := githubClient.CreateComment(context.Background(),
		conf.githubOrganization, pr.GetRepo().GetName(), pr.GetNumber(), &comment); err != nil {
		log.Infof("Failed to comment on the pr: %v, Error: %s", pr, err.Error())
	}

	return nil
}
