package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v28/github"
	"golang.org/x/sys/unix"

	clientgithub "github.com/mendersoftware/integration-test-runner/client/github"
	"github.com/mendersoftware/integration-test-runner/git"
	"github.com/mendersoftware/integration-test-runner/logger"

	"github.com/sirupsen/logrus"
)

var mutex = &sync.Mutex{}

type config struct {
	dryRunMode                       bool
	githubSecret                     []byte
	githubProtocol                   gitProtocol
	githubOrganization               string
	githubToken                      string
	gitlabToken                      string
	gitlabBaseURL                    string
	integrationDirectory             string
	watchRepositoriesTriggerPipeline []string // List of repositories for which to trigger mender-qa pipeline
}

type buildOptions struct {
	pr         string
	repo       string
	baseBranch string
	commitSHA  string
	makeQEMU   bool
}

// List of repos for which the integration pipeline shall be run
// It can be overridden with env. variable WATCH_REPOS_PIPELINE
// Keep in sync with release_tool.py --list git --all
var defaultWatchRepositoriesPipeline = []string{
	"auditlogs",
	"create-artifact-worker",
	"deployments",
	"deployments-enterprise",
	"deviceadm",
	"deviceauth",
	"deviceauth-enterprise",
	"deviceconfig",
	"deviceconnect",
	"devicemonitor",
	//"gui",
	"integration",
	"inventory",
	"inventory-enterprise",
	"mender",
	"mender-api-gateway-docker",
	"mender-artifact",
	"mender-cli",
	"mender-conductor",
	"mender-conductor-enterprise",
	"mender-connect",
	"monitor-client",
	"mtls-ambassador",
	"tenantadm",
	"useradm",
	"useradm-enterprise",
	"workflows",
	"workflows-enterprise",
	// repos outside of release_tool.py
	"meta-mender",
}

// Mapping https://github.com/<org> -> https://gitlab.com/Northern.tech/<group>
var gitHubOrganizationToGitLabGroup = map[string]string{
	"mendersoftware": "Mender",
	"cfengine":       "CFEngine",
}

// Mapping of special repos that have a custom group/project
var gitHubRepoToGitLabProjectCustom = map[string]string{
	"saas": "Northern.tech/MenderSaaS/saas",
}

var qemuBuildRepositories = []string{
	"meta-mender",
	"mender",
	"mender-artifact",
	"mender-connect",
	"monitor-client",
}

const (
	gitOperationTimeout = 30
)

const (
	featureBranchPrefix = "feature-"
)

const (
	githubBotName = "mender-test-bot"
)

const (
	commandStartPipeline    = "start pipeline"
	commandCherryPickBranch = "cherry-pick to:"
)

func getConfig() (*config, error) {
	var repositoryWatchListPipeline []string
	dryRunMode := os.Getenv("DRY_RUN") != ""
	githubSecret := os.Getenv("GITHUB_SECRET")
	githubToken := os.Getenv("GITHUB_TOKEN")
	gitlabToken := os.Getenv("GITLAB_TOKEN")
	gitlabBaseURL := os.Getenv("GITLAB_BASE_URL")
	integrationDirectory := os.Getenv("INTEGRATION_DIRECTORY")
	logLevel, found := os.LookupEnv("INTEGRATION_TEST_RUNNER_LOG_LEVEL")

	logrus.SetLevel(logrus.InfoLevel)

	if found {
		lvl, err := logrus.ParseLevel(logLevel)
		if err != nil {
			logrus.Infof("Failed to parse the 'INTEGRATION_TEST_RUNNER_LOG_LEVEL' variable, defaulting to 'InfoLevel'")
		} else {
			logrus.Infof("Set 'LogLevel' to %s", lvl)
			logrus.SetLevel(lvl)
		}
	}

	watchRepositoriesTriggerPipeline, ok := os.LookupEnv("WATCH_REPOS_PIPELINE")
	if ok {
		repositoryWatchListPipeline = strings.Split(watchRepositoriesTriggerPipeline, ",")
	} else {
		repositoryWatchListPipeline = defaultWatchRepositoriesPipeline
	}

	switch {
	case githubSecret == "" && !dryRunMode:
		return &config{}, fmt.Errorf("set GITHUB_SECRET")
	case githubToken == "":
		return &config{}, fmt.Errorf("set GITHUB_TOKEN")
	case gitlabToken == "":
		return &config{}, fmt.Errorf("set GITLAB_TOKEN")
	case gitlabBaseURL == "":
		return &config{}, fmt.Errorf("set GITLAB_BASE_URL")
	case integrationDirectory == "":
		return &config{}, fmt.Errorf("set INTEGRATION_DIRECTORY")
	}

	return &config{
		dryRunMode:                       dryRunMode,
		githubSecret:                     []byte(githubSecret),
		githubProtocol:                   gitProtocolSSH,
		githubToken:                      githubToken,
		gitlabToken:                      gitlabToken,
		gitlabBaseURL:                    gitlabBaseURL,
		integrationDirectory:             integrationDirectory,
		watchRepositoriesTriggerPipeline: repositoryWatchListPipeline,
	}, nil
}

func getCustomLoggerFromContext(ctx *gin.Context) *logrus.Entry {
	deliveryID, ok := ctx.Get("delivery")
	if !ok {
		return nil
	}
	return logrus.WithField("delivery", deliveryID)
}

func processGitHubWebhookRequest(ctx *gin.Context, payload []byte, githubClient clientgithub.Client, conf *config) {
	webhookType := github.WebHookType(ctx.Request)
	webhookEvent, _ := github.ParseWebHook(github.WebHookType(ctx.Request), payload)
	_ = processGitHubWebhook(ctx, webhookType, webhookEvent, githubClient, conf)
}

func processGitHubWebhook(ctx *gin.Context, webhookType string, webhookEvent interface{}, githubClient clientgithub.Client, conf *config) error {
	githubOrganization, err := getGitHubOrganization(webhookType, webhookEvent)
	if err != nil {
		logrus.Warnln("ignoring event: ", err.Error())
		return nil
	}
	conf.githubOrganization = githubOrganization
	switch webhookType {
	case "pull_request":
		pr := webhookEvent.(*github.PullRequestEvent)
		return processGitHubPullRequest(ctx, pr, githubClient, conf)
	case "push":
		push := webhookEvent.(*github.PushEvent)
		return processGitHubPush(ctx, push, githubClient, conf)
	case "issue_comment":
		comment := webhookEvent.(*github.IssueCommentEvent)
		return processGitHubComment(ctx, comment, githubClient, conf)
	}
	return nil
}

func setupLogging(conf *config, requestLogger logger.RequestLogger) {
	// Log to stdout and with JSON format; suitable for GKE
	formatter := &logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "time",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	}

	if conf.dryRunMode {
		mw := io.MultiWriter(os.Stdout, requestLogger)
		logrus.SetOutput(mw)
	} else {
		logrus.SetOutput(os.Stdout)
	}
	logrus.SetFormatter(formatter)
}

func main() {
	doMain()
}

func doMain() {
	conf, err := getConfig()
	if err != nil {
		logrus.Fatalf("failed to load config: %s", err.Error())
	}

	requestLogger := logger.NewRequestLogger()
	logger.SetRequestLogger(requestLogger)

	setupLogging(conf, requestLogger)
	git.SetDryRunMode(conf.dryRunMode)

	logrus.Infoln("using settings: ", spew.Sdump(conf))

	githubClient := clientgithub.NewGitHubClient(conf.githubToken, conf.dryRunMode)
	r := gin.Default()
	r.Use(gin.Recovery())

	// webhook for GitHub
	r.POST("/", func(context *gin.Context) {
		payload, err := github.ValidatePayload(context.Request, conf.githubSecret)
		if err != nil {
			logrus.Warnln("payload failed to validate, ignoring.")
			context.Status(http.StatusForbidden)
			return
		}
		context.Set("delivery", github.DeliveryID(context.Request))
		if conf.dryRunMode {
			processGitHubWebhookRequest(context, payload, githubClient, conf)
		} else {
			go processGitHubWebhookRequest(context, payload, githubClient, conf)
		}
		context.Status(http.StatusAccepted)
	})

	// 200 replay for the loadbalancer
	r.GET("/", func(_ *gin.Context) {})

	// dry-run mode, end-point to retrieve and clear logs
	if conf.dryRunMode {
		r.GET("/logs", func(context *gin.Context) {
			logs := requestLogger.Get()
			context.JSON(http.StatusOK, logs)
		})

		r.DELETE("/logs", func(context *gin.Context) {
			requestLogger.Clear()
			context.Writer.WriteHeader(http.StatusNoContent)
		})
	}

	srv := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Failed listening: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, unix.SIGINT, unix.SIGTERM)
	<-quit

	logrus.Info("Shutdown server ...")

	ctx := context.Background()
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctxWithTimeout); err != nil {
		logrus.Fatal("Failed to shutdown the server: ", err)
	}
}
