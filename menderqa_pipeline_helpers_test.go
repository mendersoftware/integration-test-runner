package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepoNameFromSource(t *testing.T) {
	tests := []struct {
		source   string
		expected string
	}{
		{"github.com/mendersoftware/mender", "mender"},
		{"github.com/mendersoftware/monitor-client", "monitor-client"},
		{"github.com/mendersoftware/mender-connect", "mender-connect"},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.expected, repoNameFromSource(tc.source))
	}
}

func TestMaintenanceBranchPattern(t *testing.T) {
	assert.True(t, maintenanceBranchPattern.MatchString("6.0.x"))
	assert.True(t, maintenanceBranchPattern.MatchString("10.20.x"))
	assert.False(t, maintenanceBranchPattern.MatchString("master"))
	assert.False(t, maintenanceBranchPattern.MatchString("main"))
	assert.False(t, maintenanceBranchPattern.MatchString("6.0.0"))
	assert.False(t, maintenanceBranchPattern.MatchString("feature-something"))
}

func TestFindMatchingReleases(t *testing.T) {
	releases := []MenderClientRelease{
		{
			Version: "6.0.x",
			Subcomponents: []MenderClientSubcomponent{
				{Name: "mender-auth", Version: "5.1.x", Source: "github.com/mendersoftware/mender"},
				{Name: "mender-update", Version: "5.1.x", Source: "github.com/mendersoftware/mender"},
				{Name: "mender-connect", Version: "3.0.x", Source: "github.com/mendersoftware/mender-connect"},
				{Name: "mender-monitor", Version: "1.5.x", Source: "github.com/mendersoftware/monitor-client"},
			},
		},
		{
			Version: "6.1.x",
			Subcomponents: []MenderClientSubcomponent{
				{Name: "mender-auth", Version: "5.2.x", Source: "github.com/mendersoftware/mender"},
				{Name: "mender-update", Version: "5.2.x", Source: "github.com/mendersoftware/mender"},
				{Name: "mender-connect", Version: "3.0.x", Source: "github.com/mendersoftware/mender-connect"},
				{Name: "mender-monitor", Version: "1.6.x", Source: "github.com/mendersoftware/monitor-client"},
			},
		},
	}

	t.Run("component in multiple releases", func(t *testing.T) {
		matched := findMatchingReleases(releases, "mender-connect", "3.0.x")
		assert.Len(t, matched, 2)
		assert.Equal(t, "6.0.x", matched[0].Version)
		assert.Equal(t, "6.1.x", matched[1].Version)
	})

	t.Run("component in single release", func(t *testing.T) {
		matched := findMatchingReleases(releases, "mender", "5.1.x")
		assert.Len(t, matched, 1)
		assert.Equal(t, "6.0.x", matched[0].Version)
	})

	t.Run("branch not found", func(t *testing.T) {
		matched := findMatchingReleases(releases, "mender-connect", "2.0.x")
		assert.Empty(t, matched)
	})

	t.Run("repo not in any release", func(t *testing.T) {
		matched := findMatchingReleases(releases, "nonexistent-repo", "1.0.x")
		assert.Empty(t, matched)
	})

	t.Run("monitor-client source mapping", func(t *testing.T) {
		matched := findMatchingReleases(releases, "monitor-client", "1.5.x")
		assert.Len(t, matched, 1)
		assert.Equal(t, "6.0.x", matched[0].Version)
	})
}

func TestReleaseVersions(t *testing.T) {
	releases := []MenderClientRelease{
		{Version: "6.0.x"},
		{Version: "6.1.x"},
	}
	assert.Equal(t, []string{"6.0.x", "6.1.x"}, releaseVersions(releases))
	assert.Equal(t, []string{}, releaseVersions(nil))
}
