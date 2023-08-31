package ks3

import (
	"strings"
	"testing"
	"time"
)

func TestLockDurationParse(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		ld             string
		expectDuration time.Duration
		expectErr      string
	}{
		"hour": {
			"1h",
			1 * time.Hour,
			"",
		},
		"minute": {
			"1m",
			1 * time.Minute,
			"",
		},
		"unlimited": {
			"-1",
			9999 * time.Hour,
			"",
		},
		"ignore": {
			"0",
			0,
			"",
		},
		"0 minute": {
			"0m",
			0,
			"",
		},
		"second": {
			"30s",
			0,
			"unexpected error",
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := lockDurationParse(tcase.ld)
			if err != nil {
				t.Log(err)
				if !strings.Contains(err.Error(), tcase.expectErr) {
					t.Errorf("parse time duration err: %s", err)
				}
			}
			if got != tcase.expectDuration {
				t.Errorf("expect %d, but got %d", tcase.expectDuration, got)
			}
		})
	}
}
