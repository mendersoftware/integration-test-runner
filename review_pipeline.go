package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	clientgithub "github.com/mendersoftware/integration-test-runner/client/github"
	clientgitlab "github.com/mendersoftware/integration-test-runner/client/gitlab"
)

const (
	reviewDeployJobName        = "review:deploy"
	defaultReviewAdminUsername = "mender-demo@example.com"
	defaultReviewAdminPassword = "mysecretpassword!123"
	defaultTestEnvironment     = "os"
	adminUsernameKey           = "REVIEW_APPS_ADMIN_USERNAME"
	adminPasswordKey           = "REVIEW_APPS_ADMIN_PASSWORD"
)

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getReviewAppURL(domain string, projectPrefix string, prNumber int) string {
	return fmt.Sprintf("https://%s-pr-%d.%s/", projectPrefix, prNumber, domain)
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
	log.Infof(
		"Found latest pipeline %d for ref %s in project %s",
		latestPipeline.ID,
		ref,
		projectPath,
	)

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
			"job %q in pipeline %d has status %q"+
				" (expected \"manual\"); builds may still be running",
			jobName,
			latestPipeline.ID,
			targetJob.Status,
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
	enterprise bool,
	githubClient clientgithub.Client,
) error {
	client, err := clientgitlab.NewGitLabClient(
		conf.gitlabToken,
		conf.gitlabBaseURL,
		conf.dryRunMode,
	)
	if err != nil {
		return err
	}
	return triggerReviewDeployWithClient(log, conf, pr, sender, enterprise, client, githubClient)
}

func triggerReviewDeployWithClient(
	log *logrus.Entry,
	conf *config,
	pr *github.PullRequestEvent,
	sender string,
	enterprise bool,
	gitlabClient clientgitlab.Client,
	githubClient clientgithub.Client,
) error {
	repoName := pr.GetRepo().GetName()
	appConf, ok := gitHubRepoToReviewAppConfig[repoName]
	if !ok {
		return fmt.Errorf(
			"review app deployment is not supported for repository %q",
			repoName,
		)
	}

	projectPath, err := getGitLabProjectPath(conf.githubOrganization, repoName)
	if err != nil {
		return err
	}

	prNumber := pr.GetNumber()
	ref := "pr_" + strconv.Itoa(prNumber)

	reviewAdminUsernameKey := adminUsernameKey
	reviewPasswordKey := adminPasswordKey
	adminUsername := getEnvOrDefault(adminUsernameKey, defaultReviewAdminUsername)
	adminPassword := getEnvOrDefault(adminPasswordKey, defaultReviewAdminPassword)
	enterpriseKey := "REVIEW_APPS_ENTERPRISE"
	enterpriseVal := "false"
	if enterprise {
		enterpriseVal = "true"
	}
	requestedByKey := "REVIEW_REQUESTED_BY"
	jobVars := []*gitlab.JobVariableOptions{
		{Key: &reviewAdminUsernameKey, Value: &adminUsername},
		{Key: &reviewPasswordKey, Value: &adminPassword},
		{Key: &enterpriseKey, Value: &enterpriseVal},
		{Key: &requestedByKey, Value: &sender},
	}

	job, err := findAndPlayJob(log, gitlabClient, projectPath, ref, reviewDeployJobName, jobVars)
	if err != nil {
		return err
	}

	log.Infof("Started review deploy job: %s", job.WebURL)

	variant := "OS"
	if enterprise {
		variant = "Enterprise"
	}
	reviewURL := getReviewAppURL(appConf.domain, appConf.projectPrefix, prNumber)
	commentBody := fmt.Sprintf(
		"Review app deploy triggered (%s): [%s #%d](%s)\n\n"+
			"Review app will be available at: %s\n\n"+
			"Credentials: `%s` / `%s`",
		variant,
		reviewDeployJobName,
		job.ID,
		job.WebURL,
		reviewURL,
		adminUsername,
		adminPassword,
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
	testEnvironment string,
	githubClient clientgithub.Client,
) error {
	client, err := clientgitlab.NewGitLabClient(
		conf.gitlabToken,
		conf.gitlabBaseURL,
		conf.dryRunMode,
	)
	if err != nil {
		return err
	}
	return triggerReviewE2EWithClient(log, conf, pr, testEnvironment, client, githubClient)
}

func triggerReviewE2EWithClient(
	log *logrus.Entry,
	conf *config,
	pr *github.PullRequestEvent,
	testEnvironment string,
	gitlabClient clientgitlab.Client,
	githubClient clientgithub.Client,
) error {
	repoName := pr.GetRepo().GetName()
	appConf, ok := gitHubRepoToReviewAppConfig[repoName]
	if !ok {
		return fmt.Errorf(
			"review app e2e tests are not supported for repository %q",
			repoName,
		)
	}

	if testEnvironment == "" {
		testEnvironment = defaultTestEnvironment
	}

	projectPath, err := getGitLabProjectPath(conf.githubOrganization, repoName)
	if err != nil {
		return err
	}

	prNumber := pr.GetNumber()
	baseURL := getReviewAppURL(appConf.domain, appConf.projectPrefix, prNumber)
	ref := "pr_" + strconv.Itoa(prNumber)

	reviewAdminUsernameKey := adminUsernameKey
	reviewPasswordKey := adminPasswordKey
	adminUsername := getEnvOrDefault(adminUsernameKey, defaultReviewAdminUsername)
	adminPassword := getEnvOrDefault(adminPasswordKey, defaultReviewAdminPassword)
	baseURLKey := "REVIEW_APPS_BASE_URL"
	runReviewE2EKey := "RUN_REVIEW_E2E"
	runReviewE2EVal := "true"
	testEnvKey := "REVIEW_APPS_TEST_ENVIRONMENT"

	variables := []*gitlab.PipelineVariableOptions{
		{Key: &reviewAdminUsernameKey, Value: &adminUsername},
		{Key: &reviewPasswordKey, Value: &adminPassword},
		{Key: &baseURLKey, Value: &baseURL},
		{Key: &runReviewE2EKey, Value: &runReviewE2EVal},
		{Key: &testEnvKey, Value: &testEnvironment},
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

	pipeline, err := gitlabClient.CreatePipeline(projectPath, opt)
	if err != nil {
		log.Errorf("Could not create review e2e pipeline: %s", err.Error())
		return err
	}
	log.Infof("Created review e2e pipeline: %s", pipeline.WebURL)

	commentBody := fmt.Sprintf(
		"Review app e2e test pipeline created: [Pipeline-%d](%s)\n\nEnvironment: `%s`",
		pipeline.ID,
		pipeline.WebURL,
		testEnvironment,
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
