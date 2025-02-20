package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStartPipeline(t *testing.T) {
	testCases := map[string]struct {
		branchName     string
		expectedResult bool
	}{
		"start client pipeline 1": {
			branchName:     "master",
			expectedResult: true,
		},
		"start client pipeline 2": {
			branchName:     "staging",
			expectedResult: true,
		},
		"start client pipeline 3": {
			branchName:     "production",
			expectedResult: true,
		},
		"start client pipeline 5": {
			branchName:     "3.1.x",
			expectedResult: true,
		},
		"start client pipeline 6": {
			branchName:     "pr_1",
			expectedResult: true,
		},
		"do not start client pipeline 1": {
			branchName:     "other-branch",
			expectedResult: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expectedResult, shouldStartPipeline(tc.branchName))
		})
	}
}
