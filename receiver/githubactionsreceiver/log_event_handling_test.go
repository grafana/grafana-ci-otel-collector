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
			ts, _, ok := parseTimestamp(test.logline, logger)
			if test.expectError {
				require.False(t, ok)
			} else {
				require.True(t, ok)
				require.Equal(t, test.expectedTime, ts)
			}
		})
	}
}
