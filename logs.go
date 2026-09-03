package main

import (
	"fmt"
	"net/url"
	"strings"
)

// cloudLoggingURL returns "" when the deployment does not know where its own
// logs are, so that callers omit the link rather than point at the wrong pod.
func (c *config) cloudLoggingURL(deliveryID string) string {
	if c.gcpProject == "" || c.k8sPodLabelRun == "" {
		return ""
	}

	filters := []string{
		`resource.type="k8s_container"`,
		fmt.Sprintf(`labels."k8s-pod/run"=%q`, c.k8sPodLabelRun),
		// The paths that comment log at Error; drop this line in the console
		// to see the surrounding context.
		`severity>=WARNING`,
	}
	if c.gkeCluster != "" {
		filters = append(filters, fmt.Sprintf(`resource.labels.cluster_name=%q`, c.gkeCluster))
	}
	if c.k8sNamespace != "" {
		filters = append(filters, fmt.Sprintf(`resource.labels.namespace_name=%q`, c.k8sNamespace))
	}
	if deliveryID != "" {
		filters = append(filters, fmt.Sprintf(`jsonPayload.delivery=%q`, deliveryID))
	}

	// The query sits in a path segment, so it needs full percent-encoding, and
	// a "+" would not decode back to a space there.
	query := strings.ReplaceAll(url.QueryEscape(strings.Join(filters, "\n")), "+", "%20")
	return fmt.Sprintf(
		"https://console.cloud.google.com/logs/query;query=%s;duration=P7D?project=%s",
		query, url.QueryEscape(c.gcpProject),
	)
}

// msgDetailsLogs is the tail of a failure comment.
func (c *config) msgDetailsLogs(deliveryID string) string {
	link := c.cloudLoggingURL(deliveryID)
	if link == "" {
		return "please contact the SRE team for details."
	}
	return "see <a href=\"" + link + "\">logs</a> for details."
}
