package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMenderQARef(t *testing.T) {
	tests := []struct {
		name     string
		build    buildOptions
		options  *BuildOptions
		expected string
	}{
		{
			name:     "default is master",
			build:    buildOptions{repo: "mender", pr: "123"},
			options:  NewBuildOptions(),
			expected: "master",
		},
		{
			name:     "PR on mender-qa itself",
			build:    buildOptions{repo: "mender-qa", pr: "903"},
			options:  NewBuildOptions(),
			expected: "pr_903",
		},
		{
			name:  "--pr mender-qa/903 from another repo",
			build: buildOptions{repo: "meta-mender", pr: "2511"},
			options: &BuildOptions{
				PullRequests: map[string]string{"mender-qa": "pull/903/head"},
			},
			expected: "pr_903",
		},
		{
			name:  "--pr mender-qa/some-branch",
			build: buildOptions{repo: "meta-mender", pr: "2511"},
			options: &BuildOptions{
				PullRequests: map[string]string{"mender-qa": "some-branch"},
			},
			expected: "some-branch",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, getMenderQARef(&tc.build, tc.options))
		})
	}
}
