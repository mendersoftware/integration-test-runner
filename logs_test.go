package main

import (
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func configuredForLogs() *config {
	return &config{
		gcpProject:     "gp-kubernetes-269000",
		gkeCluster:     "company-websites",
		k8sNamespace:   "default",
		k8sPodLabelRun: "repos-sync-northerntechhq-com",
	}
}

func TestCloudLoggingURLUnconfigured(t *testing.T) {
	assert.Equal(t, "", (&config{}).cloudLoggingURL("abc"))

	noProject := configuredForLogs()
	noProject.gcpProject = ""
	assert.Equal(t, "", noProject.cloudLoggingURL("abc"))

	noPod := configuredForLogs()
	noPod.k8sPodLabelRun = ""
	assert.Equal(t, "", noPod.cloudLoggingURL("abc"))
}

func TestCloudLoggingURL(t *testing.T) {
	link := configuredForLogs().cloudLoggingURL("d3-4f5")

	assert.True(t, strings.HasPrefix(link,
		"https://console.cloud.google.com/logs/query;query="), link)
	assert.Contains(t, link, "?project=gp-kubernetes-269000")
	// A "+" here would be read as a literal plus inside a path segment.
	assert.NotContains(t, link, "+")

	query := decodeLogsQuery(t, link)
	assert.Contains(t, query, `resource.type="k8s_container"`)
	assert.Contains(t, query, `labels."k8s-pod/run"="repos-sync-northerntechhq-com"`)
	assert.Contains(t, query, `resource.labels.cluster_name="company-websites"`)
	assert.Contains(t, query, `resource.labels.namespace_name="default"`)
	assert.Contains(t, query, `jsonPayload.delivery="d3-4f5"`)
	assert.Contains(t, query, `severity>=WARNING`)
}

func TestCloudLoggingURLOmitsUnknownDelivery(t *testing.T) {
	query := decodeLogsQuery(t, configuredForLogs().cloudLoggingURL(""))
	assert.NotContains(t, query, "jsonPayload.delivery")
	assert.Contains(t, query, `severity>=WARNING`)
}

func TestCloudLoggingURLIsScopedPerDeployment(t *testing.T) {
	cfengine := configuredForLogs()
	cfengine.k8sPodLabelRun = "repos-sync-cfengine-com"

	assert.NotEqual(t,
		configuredForLogs().cloudLoggingURL("abc"),
		cfengine.cloudLoggingURL("abc"))
	assert.Contains(t, decodeLogsQuery(t, cfengine.cloudLoggingURL("abc")),
		`labels."k8s-pod/run"="repos-sync-cfengine-com"`)
}

func TestMsgDetailsLogs(t *testing.T) {
	msg := configuredForLogs().msgDetailsLogs("abc")
	assert.True(t, strings.HasPrefix(msg, `see <a href="`), msg)
	assert.True(t, strings.HasSuffix(msg, `">logs</a> for details.`), msg)

	fallback := (&config{}).msgDetailsLogs("abc")
	assert.NotContains(t, fallback, "<a href")
	assert.Equal(t, "please contact the SRE team for details.", fallback)
}

func TestGetDeliveryID(t *testing.T) {
	ctx := &gin.Context{}
	assert.Equal(t, "", getDeliveryID(ctx))

	ctx.Set("delivery", 42)
	assert.Equal(t, "", getDeliveryID(ctx))

	ctx.Set("delivery", "abc")
	assert.Equal(t, "abc", getDeliveryID(ctx))
}

func decodeLogsQuery(t *testing.T, link string) string {
	t.Helper()
	_, after, found := strings.Cut(link, ";query=")
	require.True(t, found, "no ;query= segment in %q", link)
	encoded, _, found := strings.Cut(after, ";")
	require.True(t, found, "no segment after the query in %q", link)
	decoded, err := url.QueryUnescape(encoded)
	require.NoError(t, err)
	return decoded
}
