package main

import (
	"flag"
	"fmt"
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
