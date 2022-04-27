package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTypeKeyword(t *testing.T) {
	tests := map[string]struct {
		body     string
		expected string
	}{
		"fix": {
			"mark-pr as fix",
			"fix",
		},
		"feat": {
			"mark-pr as feat",
			"feat",
		},
		"with test-bot prefix": {
			`@mender-test-bot
mark-pr as feat`,
			"feat",
		},
		"with test-bot prefix single-line": {
			"@mender-test-bot mark-pr as feat",
			"feat",
		},
		"with test-bot prefix and space chars at the end": {
			"@mender-test-bot mark-pr as feat\n\t\r ",
			"feat",
		},
		"with colon": {
			"mark-pr as: feat",
			"feat",
		},
	}
	testErrors := map[string]string{
		"illegal": "mark-pr as illegal",

		"without propper conventional commit command": `@mender-tets-bot
mark pr as feat`,
	}
	for name, test := range tests {
		t.Log(name)
		res, _ := getTypeKeyword(test.body)
		assert.Equal(t, test.expected, res)
	}
	for name := range testErrors {
		t.Log(name)
		_, err := getTypeKeyword(testErrors[name])
		assert.True(t, err != nil)
	}
}

func TestConventionalComittifyDependabotMessage(t *testing.T) {
	tests := map[string]struct {
		typeKeyword string
		body        string
		expected    string
	}{
		"commit message with sign-off in footer": {
			"feat",
			`feature description

body

Signed-off-by: dependabot[bot]`,
			`feat: feature description

body

Changelog: All
Ticket: None
Signed-off-by: dependabot[bot]`,
		},
		"with changelog prefix and newline at the end": {
			"feat",
			`Changelog:All: feature description

body

Signed-off-by: dependabot[bot]
`,
			`feat: feature description

body

Changelog: All
Ticket: None
Signed-off-by: dependabot[bot]`,
		},
		"with changelog chore prefix": {
			"feat",
			`chore: feature description

body

Signed-off-by: dependabot[bot]`,
			`feat: feature description

body

Changelog: All
Ticket: None
Signed-off-by: dependabot[bot]`,
		},
	}
	for name, test := range tests {
		t.Log(name)
		res := conventionalComittifyDependabotMessage(test.body, test.typeKeyword)
		assert.Equal(t, test.expected, res)
	}
}
