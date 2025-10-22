package githubactionsreceiver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestParseTimestamp(t *testing.T) {
	tests := map[string]struct {
		logline      string
		expectError  bool
		expectedTime time.Time
	}{
		"github-10-7": {
			logline:      "2025-07-17T10:26:38.7039891Z ##[group]Run actions/cache@v4",
			expectedTime: time.Date(2025, time.July, 17, 10, 26, 38, 703989100, time.UTC),
		},
		"rfc3339-nano": {
			logline:      "2025-07-17T10:26:38.703989101Z ##[group]Run actions/cache@v4",
			expectedTime: time.Date(2025, time.July, 17, 10, 26, 38, 703989101, time.UTC),
		},
		"rfc3339": {
			logline:      "2025-07-17T10:26:38Z ##[group]Run actions/cache@v4",
			expectedTime: time.Date(2025, time.July, 17, 10, 26, 38, 0, time.UTC),
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			logger := zap.NewNop()
			ts, _, ok := parseTimestamp([]byte(test.logline), logger)
			if test.expectError {
				require.False(t, ok)
			} else {
				require.True(t, ok)
				require.Equal(t, test.expectedTime, ts)
			}
		})
	}
}

func TestExtractStepNumberFromFileName(t *testing.T) {
	tests := map[string]struct {
		fileName      string
		jobName       string
		expectedStep  int
		expectError   bool
		errorContains string
	}{
		"step number with underscore": {
			fileName:     "test/2_Run tests.txt",
			jobName:      "test",
			expectedStep: 2,
		},
		"system.txt file should be skipped": {
			fileName:      "Shellcheck scripts/system.txt",
			jobName:       "Shellcheck scripts",
			expectError:   true,
			errorContains: "skipping system file",
		},
		"job name with spaces": {
			fileName:     "Build and Test/1_Setup.txt",
			jobName:      "Build and Test",
			expectedStep: 1,
		},
		"invalid step number": {
			fileName:      "build/abc_Invalid.txt",
			jobName:       "build",
			expectError:   true,
			errorContains: "invalid syntax",
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			stepNum, err := extractStepNumberFromFileName(test.fileName, test.jobName)
			if test.expectError {
				require.Error(t, err)
				if test.errorContains != "" {
					require.Contains(t, err.Error(), test.errorContains)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expectedStep, stepNum)
			}
		})
	}
}
