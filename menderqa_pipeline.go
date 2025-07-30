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

const clientPipelinePath = "Northern.tech/Mender/mender-qa"
const LatestStableYoctoBranch = "scarthgap"

func parseClientPullRequest(
	log *logrus.Entry,
	conf *config,
	action string,
	pr *github.PullRequestEvent,
) []buildOptions {
	log.Info("Pull request event with action: ", action)
	var builds []buildOptions

	// github pull request events to trigger a CI job for
	if action == "opened" || action == "edited" || action == "reopened" ||
		action == "synchronize" || action == "ready_for_review" {

		return getClientBuilds(log, conf, pr)
	}

	return builds
}

func getClientBuilds(log *logrus.Entry, conf *config, pr *github.PullRequestEvent) []buildOptions {

	var builds []buildOptions

	repo := pr.GetRepo().GetName()

	commitSHA := pr.PullRequest.Head.GetSHA()
	//GetLabel returns "mendersoftware:master", we just want the branch
	baseBranch := strings.Split(pr.PullRequest.Base.GetLabel(), ":")[1]

	makeQEMU := false

	// we need to have the latest integration/master branch in order to use the release_tool.py
	if err := updateIntegrationRepo(conf); err != nil {
		log.Warn(err.Error())
	}

	for _, watchRepo := range pipelineRepositories {
		// make sure the repo that the pull request is performed against is
		// one that we are watching.

		if watchRepo == repo {

			// check if we need to build/test yocto
			for _, qemuRepo := range clientPipelineRepositories {
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

				if integrationsToTest, err = getIntegrationVersionsUsingMicroservice(
					log,
					repo,
					baseBranch,
					conf,
				); err != nil {
					log.Errorf(
						"failed to get related microservices for repo: %s version: %s, failed "+
							"with: %s\n",
						repo,
						baseBranch,
						err.Error(),
					)
					return nil
				}
				log.Infof(
					"the following integration branches: %s are using %s/%s",
					integrationsToTest,
					repo,
					baseBranch,
				)

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

func getMenderQARef(build *buildOptions, buildParameters []*gitlab.PipelineVariableOptions) string {
	return "master"
}

func triggerClientBuild(
	log *logrus.Entry,
	conf *config,
	build *buildOptions,
	pr *github.PullRequestEvent,
	buildOptions *BuildOptions,
) error {
	gitlabClient, err := clientgitlab.NewGitLabClient(
		conf.gitlabToken,
		conf.gitlabBaseURL,
		conf.dryRunMode,
	)
	if err != nil {
		return err
	}

	buildParameters, err := getClientBuildParameters(log, conf, build, buildOptions)
	if err != nil {
		return err
	}

	// first stop old pipelines with the same buildParameters
	stopStalePipelines(clientPipelinePath, log, gitlabClient, buildParameters)

	// trigger the new pipeline
	clientPipelinePath := "Northern.tech/Mender/mender-qa"
	ref := getMenderQARef(build, buildParameters)
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
		clientPipelinePath,
		*opt.Ref,
		variablesString,
	)

	pipeline, err := gitlabClient.CreatePipeline(clientPipelinePath, opt)
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

func getClientBuildParameters(
	log *logrus.Entry,
	conf *config,
	build *buildOptions,
	buildOptions *BuildOptions,
) ([]*gitlab.PipelineVariableOptions, error) {
	var err error
	readHead := "pull/" + build.pr + "/head"
	var buildParameters []*gitlab.PipelineVariableOptions

	var versionedRepositories []string
	if build.repo == "meta-mender" {
		// For meta-mender, pick master versions of all Mender release repos.
		versionedRepositories, err = getListOfVersionedRepositories("origin/master", conf)
	} else {
		versionedRepositories, err = getListOfVersionedRepositories(
			"origin/"+build.baseBranch,
			conf,
		)
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

			repoKey := repoToBuildParameter(versionedRepo)

			if _, exists := buildOptions.PullRequests[versionedRepo]; exists {
				prVersion := buildOptions.PullRequests[versionedRepo]
				buildParameters = append(
					buildParameters,
					&gitlab.PipelineVariableOptions{
						Key:   &repoKey,
						Value: &prVersion,
					},
				)
				continue
			}
			version, err := getServiceRevisionFromIntegration(
				versionedRepo,
				"origin/"+build.baseBranch,
				conf,
			)
			if err != nil {
				log.Errorf("failed to determine %s version: %s", versionedRepo, err.Error())
				return nil, err
			}
			log.Infof("%s version %s is being used in %s", versionedRepo, version, build.baseBranch)
			buildParameters = append(
				buildParameters,
				&gitlab.PipelineVariableOptions{
					Key:   &repoKey,
					Value: &version,
				},
			)
		}
	}

	// set the correct integration branches if we aren't performing a pull request against
	// integration
	if build.repo != "integration" && build.repo != "meta-mender" {
		integrationRevision := build.baseBranch
		if _, exists := buildOptions.PullRequests["integration"]; exists {
			integrationRevision = buildOptions.PullRequests["integration"]
		}
		integrationRepoKey := repoToBuildParameter("integration")
		buildParameters = append(buildParameters,
			&gitlab.PipelineVariableOptions{
				Key:   &integrationRepoKey,
				Value: &integrationRevision})

		if _, exists := buildOptions.PullRequests["meta-mender"]; exists {
			metaMenderRevision := buildOptions.PullRequests["meta-mender"]
			metaMenderRepoKey := repoToBuildParameter("meta-mender")
			buildParameters = append(buildParameters,
				&gitlab.PipelineVariableOptions{
					Key:   &metaMenderRepoKey,
					Value: &metaMenderRevision})
		}
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
			metaMenderBranch := pokyBranch
			metaMenderBranchKey := repoToBuildParameter("meta-mender")
			buildParameters = append(
				buildParameters,
				&gitlab.PipelineVariableOptions{
					Key:   &metaMenderBranchKey,
					Value: &metaMenderBranch,
				},
			)
		}
		pokyBranchKey := repoToBuildParameter("poky")
		buildParameters = append(
			buildParameters,
			&gitlab.PipelineVariableOptions{Key: &pokyBranchKey, Value: &pokyBranch},
		)
		metaOEPokyBranchKey := repoToBuildParameter("meta-openembedded")
		buildParameters = append(
			buildParameters,
			&gitlab.PipelineVariableOptions{
				Key:   &metaOEPokyBranchKey,
				Value: &pokyBranch,
			},
		)
		metaRPIPokyBranchKey := repoToBuildParameter("meta-raspberrypi")
		buildParameters = append(
			buildParameters,
			&gitlab.PipelineVariableOptions{
				Key:   &metaRPIPokyBranchKey,
				Value: &pokyBranch,
			},
		)
	}

	// set the rest of the CI build parameters
	runIntegrationTests := "true"
	runIntegrationTestsKey := "RUN_INTEGRATION_TESTS"
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

	return getClientAcceptanceBuildParameters(buildParameters, build)
}

func getClientAcceptanceBuildParameters(
	buildParameters []*gitlab.PipelineVariableOptions,
	build *buildOptions,
) ([]*gitlab.PipelineVariableOptions, error) {
	var qemuParam string
	if build.makeQEMU {
		qemuParam = "true"
	} else {
		qemuParam = ""
	}

	buildX86UefiGrubKey := "BUILD_QEMUX86_64_UEFI_GRUB"
	buildParameters = append(
		buildParameters,
		&gitlab.PipelineVariableOptions{Key: &buildX86UefiGrubKey, Value: &qemuParam},
	)
	testX86UefiGrubKey := "TEST_QEMUX86_64_UEFI_GRUB"
	buildParameters = append(
		buildParameters,
		&gitlab.PipelineVariableOptions{Key: &testX86UefiGrubKey, Value: &qemuParam},
	)

	buildX86BiosGrubKey := "BUILD_QEMUX86_64_BIOS_GRUB"
	buildParameters = append(
		buildParameters,
		&gitlab.PipelineVariableOptions{Key: &buildX86BiosGrubKey, Value: &qemuParam},
	)
	testX86BiosGrubKey := "TEST_QEMUX86_64_BIOS_GRUB"
	buildParameters = append(
		buildParameters,
		&gitlab.PipelineVariableOptions{Key: &testX86BiosGrubKey, Value: &qemuParam},
	)

	buildX86BiosGrubGptKey := "BUILD_QEMUX86_64_BIOS_GRUB_GPT"
	buildParameters = append(
		buildParameters,
		&gitlab.PipelineVariableOptions{Key: &buildX86BiosGrubGptKey, Value: &qemuParam},
	)
	testX86BiosGrubGptKey := "TEST_QEMUX86_64_BIOS_GRUB_GPT"
	buildParameters = append(
		buildParameters,
		&gitlab.PipelineVariableOptions{Key: &testX86BiosGrubGptKey, Value: &qemuParam},
	)

	buildVexpress := "BUILD_VEXPRESS_QEMU"
	buildParameters = append(
		buildParameters,
		&gitlab.PipelineVariableOptions{Key: &buildVexpress, Value: &qemuParam},
	)
	testVexpress := "TEST_VEXPRESS_QEMU"
	buildParameters = append(
		buildParameters,
		&gitlab.PipelineVariableOptions{Key: &testVexpress, Value: &qemuParam},
	)

	buildVexpressFlash := "BUILD_VEXPRESS_QEMU_FLASH"
	buildParameters = append(
		buildParameters,
		&gitlab.PipelineVariableOptions{Key: &buildVexpressFlash, Value: &qemuParam},
	)
	testVexpressFlash := "TEST_VEXPRESS_QEMU_FLASH"
	buildParameters = append(
		buildParameters,
		&gitlab.PipelineVariableOptions{Key: &testVexpressFlash, Value: &qemuParam},
	)

	buildVexpressUbootUefiGrub := "BUILD_VEXPRESS_QEMU_UBOOT_UEFI_GRUB"
	buildParameters = append(
		buildParameters,
		&gitlab.PipelineVariableOptions{Key: &buildVexpressUbootUefiGrub, Value: &qemuParam},
	)
	testVexpressUbootUefiGrub := "TEST_VEXPRESS_QEMU_UBOOT_UEFI_GRUB"
	buildParameters = append(
		buildParameters,
		&gitlab.PipelineVariableOptions{Key: &testVexpressUbootUefiGrub, Value: &qemuParam},
	)

	buildBBBKey := "BUILD_BEAGLEBONEBLACK"
	buildParameters = append(
		buildParameters,
		&gitlab.PipelineVariableOptions{Key: &buildBBBKey, Value: &qemuParam},
	)

	buildClientKey := "BUILD_CLIENT"
	buildClient := "true"
	buildParameters = append(
		buildParameters,
		&gitlab.PipelineVariableOptions{Key: &buildClientKey, Value: &buildClient},
	)

	return buildParameters, nil
}

// stopBuildsOfStaleClientPRs stops any running pipelines on a PR which has been merged.
func stopBuildsOfStaleClientPRs(
	log *logrus.Entry,
	pr *github.PullRequestEvent,
	conf *config) error {

	// If the action is "closed" the pull request was merged or just closed,
	// stop builds in both cases.
	if pr.GetAction() != "closed" {
		log.Debugf("stopBuildsOfStaleClientPRs: PR not closed, " +
			"therefore not stopping it's pipeline")
		return nil
	}

	log.Debug("stopBuildsOfStaleClientPRs: Find any running pipelines and kill mercilessly!")

	for _, build := range getClientBuilds(log, conf, pr) {

		gitlabClient, err := clientgitlab.NewGitLabClient(
			conf.gitlabToken,
			conf.gitlabBaseURL,
			conf.dryRunMode,
		)
		if err != nil {
			return err
		}

		buildParams, err := getClientBuildParameters(log, conf, &build, NewBuildOptions())
		if err != nil {
			log.Debug("stopBuildsOfStaleClientPRs: Failed to get the" +
				"build-parameters for the build")
			return err
		}

		stopStalePipelines(clientPipelinePath, log, gitlabClient, buildParams)
	}

	return nil

}
