package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/mendersoftware/integration-test-runner/logger"
)

func TestPlayJobDryRunOmitsVariableValues(t *testing.T) {
	requestLogger := logger.NewRequestLogger()
	logger.SetRequestLogger(requestLogger)

	usernameKey, username := "REGISTRY_MENDER_IO_USERNAME", "mender-demo"
	passwordKey, password := "REGISTRY_MENDER_IO_PASSWORD", "mysecretpassword!123"

	client := &gitLabClient{dryRunMode: true}
	_, err := client.PlayJob("Northern.tech/Mender/mender-server", 42, &gitlab.PlayJobOptions{
		JobVariablesAttributes: &[]*gitlab.JobVariableOptions{
			{Key: &usernameKey, Value: &username},
			{Key: &passwordKey, Value: &password},
		},
	})
	assert.NoError(t, err)

	logs := requestLogger.Get()
	assert.Len(t, logs, 1)
	assert.Contains(t, logs[0], usernameKey)
	assert.Contains(t, logs[0], passwordKey)
	assert.NotContains(t, logs[0], username)
	assert.NotContains(t, logs[0], password)
}

func TestJobVariableKeys(t *testing.T) {
	key, value := "A_KEY", "a-value"

	assert.Nil(t, jobVariableKeys(nil))
	assert.Nil(t, jobVariableKeys(&gitlab.PlayJobOptions{}))
	assert.Equal(t, []string{key}, jobVariableKeys(&gitlab.PlayJobOptions{
		JobVariablesAttributes: &[]*gitlab.JobVariableOptions{
			nil,
			{Value: &value},
			{Key: &key, Value: &value},
		},
	}))
}

func TestRetryJobDryRun(t *testing.T) {
	requestLogger := logger.NewRequestLogger()
	logger.SetRequestLogger(requestLogger)

	client := &gitLabClient{dryRunMode: true}
	job, err := client.RetryJob("Northern.tech/Mender/mender-server", 42)
	assert.NoError(t, err)
	assert.NotNil(t, job)

	logs := requestLogger.Get()
	assert.Len(t, logs, 1)
	assert.Contains(t, logs[0], "gitlab.RetryJob")
	assert.Contains(t, logs[0], "path=Northern.tech/Mender/mender-server")
	assert.Contains(t, logs[0], "jobID=42")
}
