package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	clientgithub "github.com/mendersoftware/integration-test-runner/client/github"
	"github.com/mendersoftware/integration-test-runner/git"
	"github.com/mendersoftware/integration-test-runner/logger"
)

type config struct {
	dryRunMode             bool
	githubSecret           []byte
	githubProtocol         gitProtocol
	githubOrganization     string
	githubToken            string
	gitlabToken            string
	gitlabBaseURL          string
	integrationDirectory   string
	isProcessPushEvents    bool
	isProcessPREvents      bool
	isProcessCommentEvents bool
	reposSyncList          []string
}

type buildOptions struct {
	pr         string
	repo       string
	baseBranch string
	commitSHA  string
	makeQEMU   bool
}

// Mapping https://github.com/<org> -> https://gitlab.com/Northern.tech/<group>
var gitHubOrganizationToGitLabGroup = map[string]string{
	"mendersoftware": "Mender",
	"cfengine":       "CFEngine",
	"NorthernTechHQ": "NorthernTechHQ",
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
	"mender-auth-azure-iot",
	"mender-gateway",
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
	commandStartPipeline      = "start pipeline"
	commandCherryPickBranch   = "cherry-pick to:"
	commandConventionalCommit = "mark-pr as"
	commandSyncRepos          = "sync"
)

func getConfig() (*config, error) {
	var reposSyncList []string
	dryRunMode := os.Getenv("DRY_RUN") != ""
	githubSecret := os.Getenv("GITHUB_SECRET")
	githubToken := os.Getenv("GITHUB_TOKEN")
	gitlabToken := os.Getenv("GITLAB_TOKEN")
	gitlabBaseURL := os.Getenv("GITLAB_BASE_URL")
	integrationDirectory := "/integration/"
	if integrationDirEnv := os.Getenv("INTEGRATION_DIRECTORY"); integrationDirEnv != "" {
		integrationDirectory = integrationDirEnv
	}

	//
	// Currently we don't have a distinguishment between GitHub events and features.
	// Different features might be implemented across different events, but in future
	// it's probability that we might implement proper features selection. For now the
	// straight goal is to being able to configure the runner to only sync repos on
	// push events and disable all the rest (to be used by the CFEngine team).
	//
	// default: process push events (sync repos) if not explicitly disabled
	isProcessPushEvents := os.Getenv("DISABLE_PUSH_EVENTS_PROCESSING") == ""
	// default: process PR events if not explicitly disabled
	isProcessPREvents := os.Getenv("DISABLE_PR_EVENTS_PROCESSING") == ""
	// default: process comment events if not explicitly disabled
	isProcessCommentEvents := os.Getenv("DISABLE_COMMENT_EVENTS_PROCESSING") == ""

	logLevel, found := os.LookupEnv("INTEGRATION_TEST_RUNNER_LOG_LEVEL")
	logrus.SetLevel(logrus.InfoLevel)
	if found {
		lvl, err := logrus.ParseLevel(logLevel)
		if err != nil {
			logrus.Infof(
				"Failed to parse the 'INTEGRATION_TEST_RUNNER_LOG_LEVEL' variable, " +
					"defaulting to 'InfoLevel'",
			)
		} else {
			logrus.Infof("Set 'LogLevel' to %s", lvl)
			logrus.SetLevel(lvl)
		}
	}

	// Comma separated list of repos to sync (GitHub->GitLab)
	reposSyncListRaw, found := os.LookupEnv("SYNC_REPOS_LIST")
	if found {
		reposSyncList = strings.Split(reposSyncListRaw, ",")
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
		dryRunMode:             dryRunMode,
		githubSecret:           []byte(githubSecret),
		githubProtocol:         gitProtocolSSH,
		githubToken:            githubToken,
		gitlabToken:            gitlabToken,
		gitlabBaseURL:          gitlabBaseURL,
		integrationDirectory:   integrationDirectory,
		isProcessPushEvents:    isProcessPushEvents,
		isProcessPREvents:      isProcessPREvents,
		isProcessCommentEvents: isProcessCommentEvents,
		reposSyncList:          reposSyncList,
	}, nil
}

func getCustomLoggerFromContext(ctx *gin.Context) *logrus.Entry {
	deliveryID, ok := ctx.Get("delivery")
	if !ok || !isStringType(deliveryID) {
		return logrus.WithField("delivery", "nil")
	}
	return logrus.WithField("delivery", deliveryID)
}

func isStringType(i interface{}) bool {
	switch i.(type) {
	case string:
		return true
	default:
		return false
	}
}

func processGitHubWebhookRequest(
	ctx *gin.Context,
	payload []byte,
	githubClient clientgithub.Client,
	conf *config,
) {
	webhookType := github.WebHookType(ctx.Request)
	webhookEvent, _ := github.ParseWebHook(webhookType, payload)
	_ = processGitHubWebhook(ctx, webhookType, webhookEvent, githubClient, conf)
}

func processGitHubWebhook(
	ctx *gin.Context,
	webhookType string,
	webhookEvent interface{},
	githubClient clientgithub.Client,
	conf *config,
) error {
	githubOrganization, err := getGitHubOrganization(webhookType, webhookEvent)
	if err != nil {
		logrus.Warnln("ignoring event: ", err.Error())
		return nil
	}
	conf.githubOrganization = githubOrganization
	switch webhookType {
	case "pull_request":
		if conf.isProcessPREvents {
			pr := webhookEvent.(*github.PullRequestEvent)
			return processGitHubPullRequest(ctx, pr, githubClient, conf)
		} else {
			logrus.Infof("Webhook event %s processing is skipped", webhookType)
		}
	case "push":
		if conf.isProcessPushEvents {
			push := webhookEvent.(*github.PushEvent)
			return processGitHubPush(ctx, push, githubClient, conf)
		} else {
			logrus.Infof("Webhook event %s processing is skipped", webhookType)
		}
	case "issue_comment":
		if conf.isProcessCommentEvents {
			comment := webhookEvent.(*github.IssueCommentEvent)
			return processGitHubComment(ctx, comment, githubClient, conf)
		} else {
			logrus.Infof("Webhook event %s processing is skipped", webhookType)
		}
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

var githubClient clientgithub.Client

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

	githubClient = clientgithub.NewGitHubClient(conf.githubToken, conf.dryRunMode)

	r := gin.Default()
	filter := "/_health"
	if logrus.GetLevel() == logrus.DebugLevel || logrus.GetLevel() == logrus.TraceLevel {
		filter = ""
	}
	r.Use(gin.LoggerWithWriter(gin.DefaultWriter, filter))
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
	r.GET("/_health", func(_ *gin.Context) {})
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
