package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v28/github"
	"github.com/stretchr/testify/assert"
)

func TestIsOrganizationMemberOrgOwnedApp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/apps/org-app":
			_, _ = rw.Write([]byte(`{"owner": {"login": "mendersoftware"}}`))
		case "/apps/foreign-app":
			_, _ = rw.Write([]byte(`{"owner": {"login": "someone-else"}}`))
		default:
			rw.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	ghClient := github.NewClient(nil)
	ghClient.BaseURL, _ = url.Parse(server.URL + "/")
	client := &gitHubClient{client: ghClient}

	ctx := context.Background()
	assert.True(t, client.IsOrganizationMember(ctx, "mendersoftware", "org-app[bot]"))
	assert.False(t, client.IsOrganizationMember(ctx, "mendersoftware", "foreign-app[bot]"))
	assert.False(t, client.IsOrganizationMember(ctx, "mendersoftware", "org-app"))
}
