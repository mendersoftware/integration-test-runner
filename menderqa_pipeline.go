package main

import (
	"bytes"
	"context"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"

	clientgithub "github.com/mendersoftware/integration-test-runner/client/github"
	clientgitlab "github.com/mendersoftware/integration-test-runner/client/gitlab"
)

const LatestStableYoctoBranch = "dunfell"

func say(ctx context.Context, tmplString string, data interface{}, log *logrus.Entry, conf *config, pr *github.PullRequestEvent) error {
	tmpl, err := template.New("Main").Parse(tmplString)
	if err != nil {
		log.Errorf("Failed to parse the build matrix template. Should never happen! Error: %s\n", err.Error())
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

	client := clientgithub.NewGitHubClient(conf.githubToken, conf.dryRunMode)
	err = client.CreateComment(ctx,
		conf.githubOrganization, pr.GetRepo().GetName(), pr.GetNumber(), &comment)
	if err != nil {
		log.Infof("Failed to comment on the pr: %v, Error: %s", pr, err.Error())
	}

	return err
}

func parsePullRequest(log *logrus.Entry, conf *config, action string, pr *github.PullRequestEvent) []buildOptions {
	log.Info("Pull request event with action: ", action)
	var builds []buildOptions

	// github pull request events to trigger a CI job for
	if action == "opened" || action == "edited" || action == "reopened" ||
		action == "synchronize" || action == "ready_for_review" {

		return getBuilds(log, conf, pr)
	}

	return builds
}

func getBuilds(log *logrus.Entry, conf *config, pr *github.PullRequestEvent) []buildOptions {

	var builds []buildOptions

	repo := pr.GetRepo().GetName()

	commitSHA := pr.PullRequest.Head.GetSHA()
	//GetLabel returns "mendersoftware:master", we just want the branch
	baseBranch := strings.Split(pr.PullRequest.Base.GetLabel(), ":")[1]

	makeQEMU := false

	// we need to have the latest integration/master branch in order to use the release_tool.py
	if err := updateIntegrationRepo(conf); err != nil {
		log.Warnf(err.Error())
	}

	watchRepositoriesTriggerPipeline, err := getListOfWatchedRepositories(conf)
	if err != nil {
		log.Warnf(err.Error())
	}

	for _, watchRepo := range watchRepositoriesTriggerPipeline {
		// make sure the repo that the pull request is performed against is
		// one that we are watching.

		if watchRepo == repo {

			// check if we need to build/test yocto
			for _, qemuRepo := range qemuBuildRepositories {
				if repo == qemuRepo {
					makeQEMU = true
				}
			}

			switch repo {
			case "meta-mender", "integration":
				build := buildOptions{
					pr:         strconv.Itoa(pr.GetNumber()),
					repo:       repo,
					baseBranch: baseBranch,
					commitSHA:  commitSHA,
					makeQEMU:   makeQEMU,
				}
				builds = append(builds, build)

			default:
				var err error
				var integrationsToTest []string

				if integrationsToTest, err = getIntegrationVersionsUsingMicroservice(log, repo, baseBranch, conf); err != nil {
					log.Errorf("failed to get related microservices for repo: %s version: %s, failed with: %s\n", repo, baseBranch, err.Error())
					return nil
				}
				log.Infof("the following integration branches: %s are using %s/%s", integrationsToTest, repo, baseBranch)

				// one pull request can trigger multiple builds
				for _, integrationBranch := range integrationsToTest {
					buildOpts := buildOptions{
						pr:         strconv.Itoa(pr.GetNumber()),
						repo:       repo,
						baseBranch: integrationBranch,
						commitSHA:  commitSHA,
						makeQEMU:   makeQEMU,
					}
					builds = append(builds, buildOpts)
				}
			}

		}
	}
	return builds
}

func triggerBuild(log *logrus.Entry, conf *config, build *buildOptions, pr *github.PullRequestEvent, prRepos map[string]string) error {
	gitlabClient, err := clientgitlab.NewGitLabClient(conf.gitlabToken, conf.gitlabBaseURL, conf.dryRunMode)
	if err != nil {
		return err
	}

	buildParameters, err := getBuildParameters(log, conf, build, prRepos)
	if err != nil {
		return err
	}

	// first stop old pipelines with the same buildParameters
	stopStalePipelines(log, gitlabClient, buildParameters)

	// trigger the new pipeline
	integrationPipelinePath := "Northern.tech/Mender/mender-qa"
	ref := "master"
	opt := &gitlab.CreatePipelineOptions{
		Ref:       &ref,
		Variables: buildParameters,
	}

	variablesString := ""
	for _, variable := range opt.Variables {
		variablesString += variable.Key + ":" + variable.Value + ", "
	}
	log.Infof("Creating pipeline in project %s:%s with variables: %s", integrationPipelinePath, *opt.Ref, variablesString)

	pipeline, err := gitlabClient.CreatePipeline(integrationPipelinePath, opt)
	if err != nil {
		log.Errorf("Could not create pipeline: %s", err.Error())
	}
	log.Infof("Created pipeline: %s", pipeline.WebURL)

	// Add the build variable matrix to the pipeline comment under a
	// drop-down tab
	tmplString := `
Hello :smile_cat: I created a pipeline for you here: [Pipeline-{{.Pipeline.ID}}]({{.Pipeline.WebURL}})

<details>
    <summary>Build Configuration Matrix</summary><p>

| Key   | Value |
| ----- | ----- |
{{range $i, $var := .BuildVars}}{{if $var.Value}}| {{$var.Key}} | {{$var.Value}} |{{printf "\n"}}{{end}}{{end}}

 </p></details>
`
	tmpl, err := template.New("Main").Parse(tmplString)
	if err != nil {
		log.Errorf("Failed to parse the build matrix template. Should never happen! Error: %s\n", err.Error())
	}
	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, struct {
		BuildVars []*gitlab.PipelineVariable
		Pipeline  *gitlab.Pipeline
	}{
		BuildVars: opt.Variables,
		Pipeline:  pipeline,
	}); err != nil {
		log.Errorf("Failed to execute the build matrix template. Error: %s\n", err.Error())
	}

	// Comment with a pipeline-link on the PR
	commentBody := buf.String()
	comment := github.IssueComment{
		Body: &commentBody,
	}

	client := clientgithub.NewGitHubClient(conf.githubToken, conf.dryRunMode)
	err = client.CreateComment(context.Background(),
		conf.githubOrganization, pr.GetRepo().GetName(), pr.GetNumber(), &comment)
	if err != nil {
		log.Infof("Failed to comment on the pr: %v, Error: %s", pr, err.Error())
	}

	return err
}

func stopStalePipelines(log *logrus.Entry, client clientgitlab.Client, vars []*gitlab.PipelineVariable) {
	integrationPipelinePath := "Northern.tech/Mender/mender-qa"

	sort.SliceStable(vars, func(i, j int) bool {
		return vars[i].Key < vars[j].Key
	})

	username := githubBotName
	status := gitlab.Pending
	opt := &gitlab.ListProjectPipelinesOptions{
		Username: &username,
		Status:   &status,
	}

	pipelinesPending, err := client.ListProjectPipelines(integrationPipelinePath, opt)
	if err != nil {
		log.Errorf("stopStalePipelines: Could not list pending pipelines: %s", err.Error())
	}

	status = gitlab.Running
	opt = &gitlab.ListProjectPipelinesOptions{
		Username: &username,
		Status:   &status,
	}

	pipelinesRunning, err := client.ListProjectPipelines(integrationPipelinePath, opt)
	if err != nil {
		log.Errorf("stopStalePipelines: Could not list running pipelines: %s", err.Error())
	}

	for _, pipeline := range append(pipelinesPending, pipelinesRunning...) {

		variables, err := client.GetPipelineVariables(integrationPipelinePath, pipeline.ID)
		if err != nil {
			log.Errorf("stopStalePipelines: Could not get variables for pipeline: %s", err.Error())
			continue
		}

		sort.SliceStable(variables, func(i, j int) bool {
			return variables[i].Key < variables[j].Key
		})

		if reflect.DeepEqual(vars, variables) {
			log.Infof("Cancelling stale pipeline %d, url: %s", pipeline.ID, pipeline.WebURL)

			err := client.CancelPipelineBuild(integrationPipelinePath, pipeline.ID)
			if err != nil {
				log.Errorf("stopStalePipelines: Could not cancel pipeline: %s", err.Error())
			}

		}

	}
}

func getBuildParameters(log *logrus.Entry, conf *config, build *buildOptions, prsRepos map[string]string) ([]*gitlab.PipelineVariable, error) {
	var err error
	readHead := "pull/" + build.pr + "/head"
	var buildParameters []*gitlab.PipelineVariable

	var versionedRepositories []string
	if build.repo == "meta-mender" {
		// For meta-mender, pick master versions of all Mender release repos.
		versionedRepositories, err = getListOfVersionedRepositories("origin/master", conf)
	} else {
		versionedRepositories, err = getListOfVersionedRepositories("origin/"+build.baseBranch, conf)
	}
	if err != nil {
		log.Errorf("Could not get list of repositories: %s", err.Error())
		return nil, err
	}

	for _, versionedRepo := range versionedRepositories {
		// iterate over all the repositories (except the one we are testing) and
		// set the correct microservice versions

		// use the default "master" for both mender-qa, and meta-mender (set in CI)
		if versionedRepo != build.repo &&
			versionedRepo != "integration" &&
			build.repo != "meta-mender" {
			if _, exists := prsRepos[versionedRepo]; exists {
				buildParameters = append(buildParameters, &gitlab.PipelineVariable{Key: repoToBuildParameter(versionedRepo), Value: prsRepos[versionedRepo]})
				continue
			}
			version, err := getServiceRevisionFromIntegration(versionedRepo, "origin/"+build.baseBranch, conf)
			if err != nil {
				log.Errorf("failed to determine %s version: %s", versionedRepo, err.Error())
				return nil, err
			}
			log.Infof("%s version %s is being used in %s", versionedRepo, version, build.baseBranch)
			buildParameters = append(buildParameters, &gitlab.PipelineVariable{Key: repoToBuildParameter(versionedRepo), Value: version})
		}
	}

	// set the correct integration branches if we aren't performing a pull request against integration
	if build.repo != "integration" && build.repo != "meta-mender" {
		revision := build.baseBranch
		if _, exists := prsRepos["integration"]; exists {
			revision = prsRepos["integration"]
		}
		buildParameters = append(buildParameters,
			&gitlab.PipelineVariable{
				Key:   repoToBuildParameter("integration"),
				Value: revision})
	}

	// Set poky (& friends) and meta-mender revisions:
	// - If building a master PR, leave everything at defaults, which generally means
	//   meta-mender/master and poky/LatestStableYoctoBranch.
	// - If building meta-mender @ non-master, set poky branches to its baseBranch.
	// - If building any other repo @ non-master, set both meta-mender and poky to
	//   LatestStableYoctoBranch.
	if build.baseBranch != "master" {
		var pokyBranch string
		if build.repo == "meta-mender" {
			pokyBranch = build.baseBranch
		} else {
			pokyBranch = LatestStableYoctoBranch
			buildParameters = append(buildParameters, &gitlab.PipelineVariable{Key: repoToBuildParameter("meta-mender"), Value: pokyBranch})
		}
		buildParameters = append(buildParameters, &gitlab.PipelineVariable{Key: repoToBuildParameter("poky"), Value: pokyBranch})
		buildParameters = append(buildParameters, &gitlab.PipelineVariable{Key: repoToBuildParameter("meta-openembedded"), Value: pokyBranch})
		buildParameters = append(buildParameters, &gitlab.PipelineVariable{Key: repoToBuildParameter("meta-raspberrypi"), Value: pokyBranch})
	}

	// set the rest of the CI build parameters
	buildParameters = append(buildParameters, &gitlab.PipelineVariable{Key: "RUN_INTEGRATION_TESTS", Value: "true"})
	buildParameters = append(buildParameters,
		&gitlab.PipelineVariable{
			Key:   repoToBuildParameter(build.repo),
			Value: readHead,
		})

	var qemuParam string
	if build.makeQEMU {
		qemuParam = "true"
	} else {
		qemuParam = ""
	}

	buildParameters = append(buildParameters, &gitlab.PipelineVariable{Key: "BUILD_QEMUX86_64_UEFI_GRUB", Value: qemuParam})
	buildParameters = append(buildParameters, &gitlab.PipelineVariable{Key: "TEST_QEMUX86_64_UEFI_GRUB", Value: qemuParam})

	buildParameters = append(buildParameters, &gitlab.PipelineVariable{Key: "BUILD_QEMUX86_64_BIOS_GRUB", Value: qemuParam})
	buildParameters = append(buildParameters, &gitlab.PipelineVariable{Key: "TEST_QEMUX86_64_BIOS_GRUB", Value: qemuParam})

	buildParameters = append(buildParameters, &gitlab.PipelineVariable{Key: "BUILD_QEMUX86_64_BIOS_GRUB_GPT", Value: qemuParam})
	buildParameters = append(buildParameters, &gitlab.PipelineVariable{Key: "TEST_QEMUX86_64_BIOS_GRUB_GPT", Value: qemuParam})

	buildParameters = append(buildParameters, &gitlab.PipelineVariable{Key: "BUILD_VEXPRESS_QEMU", Value: qemuParam})
	buildParameters = append(buildParameters, &gitlab.PipelineVariable{Key: "TEST_VEXPRESS_QEMU", Value: qemuParam})

	buildParameters = append(buildParameters, &gitlab.PipelineVariable{Key: "BUILD_VEXPRESS_QEMU_FLASH", Value: qemuParam})
	buildParameters = append(buildParameters, &gitlab.PipelineVariable{Key: "TEST_VEXPRESS_QEMU_FLASH", Value: qemuParam})

	buildParameters = append(buildParameters, &gitlab.PipelineVariable{Key: "BUILD_VEXPRESS_QEMU_UBOOT_UEFI_GRUB", Value: qemuParam})
	buildParameters = append(buildParameters, &gitlab.PipelineVariable{Key: "TEST_VEXPRESS_QEMU_UBOOT_UEFI_GRUB", Value: qemuParam})

	buildParameters = append(buildParameters, &gitlab.PipelineVariable{Key: "BUILD_BEAGLEBONEBLACK", Value: qemuParam})

	// Set BUILD_CLIENT = false, if target repo not in the qemuBuildRepositories list
	buildClient := &gitlab.PipelineVariable{Key: "BUILD_CLIENT", Value: "false"}
	for _, prebuiltClientRepo := range qemuBuildRepositories {
		if build.repo == prebuiltClientRepo {
			buildClient.Value = "true"
		}
	}
	buildParameters = append(buildParameters, buildClient)

	return buildParameters, nil
}

// stopBuildsOfStalePRs stops any running pipelines on a PR which has been merged.
func stopBuildsOfStalePRs(log *logrus.Entry, pr *github.PullRequestEvent, conf *config) error {

	// If the action is "closed" the pull request was merged or just closed,
	// stop builds in both cases.
	if pr.GetAction() != "closed" {
		log.Debugf("stopBuildsOfStalePRs: PR not closed, therefore not stopping it's pipeline")
		return nil
	}

	log.Debug("stopBuildsOfStalePRs: Find any running pipelines and kill mercilessly!")

	for _, build := range getBuilds(log, conf, pr) {

		gitlabClient, err := clientgitlab.NewGitLabClient(conf.gitlabToken, conf.gitlabBaseURL, conf.dryRunMode)
		if err != nil {
			return err
		}

		buildParams, err := getBuildParameters(log, conf, &build, map[string]string{})
		if err != nil {
			log.Debug("stopBuildsOfStalePRs: Failed to get the build-parameters for the build")
			return err
		}

		stopStalePipelines(log, gitlabClient, buildParams)
	}

	return nil

}
