package main

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
)

var runAcceptanceTests bool

var _ = func() bool {
	testing.Init()
	return true
}()

func init() {
	flag.BoolVar(&runAcceptanceTests, "acceptance-tests", false, "set flag when running acceptance tests")
	flag.Parse()
}

func TestRunMain(t *testing.T) {
	if !runAcceptanceTests {
		t.Skip()
	}
	doMain()
}

func TestRedactedSecret(t *testing.T) {
	assert.Equal(t, "<unset>", redactedSecret(""))
	assert.Equal(t, "<set,3 bytes>", redactedSecret("abc"))
}

func TestConfigStringRedactsSecrets(t *testing.T) {
	conf := &config{
		dryRunMode:           true,
		githubOrganization:   "mendersoftware",
		gitlabBaseURL:        "https://gitlab.com/api/v4",
		integrationDirectory: "/integration/",
		reposSyncList:        []string{"website", "build-website"},
		githubSecret:         []byte("hmac-secret-value"),
		githubToken:          "ghp_realtokenvalue",
		gitlabToken:          "glpat-realtokenvalue",
	}

	secrets := []string{
		"hmac-secret-value",
		"ghp_realtokenvalue",
		"glpat-realtokenvalue",
		// spew renders a []byte as a hex dump, which is what made the leak decodable.
		"68 6d 61 63",
	}

	outputs := map[string]string{
		"%v":         fmt.Sprintf("%v", conf),
		"%s":         fmt.Sprintf("%s", conf),
		"%+v":        fmt.Sprintf("%+v", conf),
		"Sprint":     fmt.Sprint(conf),
		"spew.Sdump": spew.Sdump(conf),
	}

	for name, out := range outputs {
		for _, secret := range secrets {
			assert.NotContains(t, out, secret, "%s leaked a secret", name)
		}
	}

	settings := fmt.Sprintf("%v", conf)
	assert.Contains(t, settings, `githubOrganization:"mendersoftware"`)
	assert.Contains(t, settings, `gitlabBaseURL:"https://gitlab.com/api/v4"`)
	assert.Contains(t, settings, "reposSyncList:[website build-website]")

	// A set secret must read as present, not be omitted: "not configured" and
	// "configured but wrong" are different bugs.
	assert.Contains(t, settings, "githubSecret:<set,17 bytes>")
	assert.Contains(t, settings, "githubToken:<set,18 bytes>")
	assert.Contains(t, settings, "gitlabToken:<set,20 bytes>")
	assert.Contains(t, (&config{}).String(), "gitlabToken:<unset>")
}

func TestSplitReposSyncList(t *testing.T) {
	testCases := map[string]struct {
		raw      string
		expected []string
	}{
		"empty":           {raw: "", expected: nil},
		"only separators": {raw: ",,", expected: nil},
		"only whitespace": {raw: " , ", expected: nil},
		"single":          {raw: "website", expected: []string{"website"}},
		"several": {
			raw:      "website,build-website",
			expected: []string{"website", "build-website"},
		},
		"padded":           {raw: "a, b ,c", expected: []string{"a", "b", "c"}},
		"empty entries":    {raw: "a,,b,", expected: []string{"a", "b"}},
		"trailing newline": {raw: "a,b\n", expected: []string{"a", "b"}},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, splitReposSyncList(tc.raw))
		})
	}
}

func TestGetConfigReposSyncList(t *testing.T) {
	testCases := map[string]struct {
		set      bool
		raw      string
		expected []string
	}{
		"unset": {set: false, expected: nil},
		"empty": {set: true, raw: "", expected: nil},
		"padded": {
			set:      true,
			raw:      "website, build-website",
			expected: []string{"website", "build-website"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Setenv("GITHUB_SECRET", "secret")
			t.Setenv("GITHUB_TOKEN", "github-token")
			t.Setenv("GITLAB_TOKEN", "gitlab-token")
			t.Setenv("GITLAB_BASE_URL", "https://gitlab.com/api/v4")
			// t.Setenv registers the restore, so this Unsetenv is undone on cleanup.
			t.Setenv("SYNC_REPOS_LIST", tc.raw)
			if !tc.set {
				assert.NoError(t, os.Unsetenv("SYNC_REPOS_LIST"))
			}

			conf, err := getConfig()
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, conf.reposSyncList)
		})
	}
}

func TestIsRepoInSyncList(t *testing.T) {
	northernTechHQ := []string{
		"alvaldi-docs", "alvaldi-gui", "alvaldi-helm", "nt-connect",
		"nt-boilerplate-pipeline", "nt-iac", "nt-gui", "sales-outreach-tool",
		"nt-app", "mender-cn-website", "renovate-ring",
	}

	testCases := map[string]struct {
		reposSyncList []string
		repo          string
		expected      bool
	}{
		"nil list":    {reposSyncList: nil, repo: "libntech", expected: true},
		"empty list":  {reposSyncList: []string{}, repo: "libntech", expected: true},
		"in list":     {reposSyncList: northernTechHQ, repo: "nt-connect", expected: true},
		"not in list": {reposSyncList: northernTechHQ, repo: "libntech", expected: false},
		"no substring match": {
			reposSyncList: []string{"website"}, repo: "build-website", expected: false,
		},
		"exact match only": {
			reposSyncList: []string{"build-website"}, repo: "website", expected: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			conf := &config{reposSyncList: tc.reposSyncList}
			assert.Equal(t, tc.expected, conf.isRepoInSyncList(tc.repo))
		})
	}
}
