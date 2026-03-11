// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduleaddrs

import "testing"

// The actual tests for this live in the other detect_*_test.go files, but
// this file contains helpers that all of those tests share.

func tableTestDetectorFuncs(t *testing.T, cases []struct{ Input, Output string }) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.Input, func(t *testing.T) {
			output, err := detectRemoteSourceShorthands(tc.Input)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if output != tc.Output {
				t.Errorf("wrong result\ninput: %s\ngot:   %s\nwant:  %s", tc.Input, output, tc.Output)
			}
		})
	}
}
