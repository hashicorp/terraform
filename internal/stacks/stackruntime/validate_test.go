package stackruntime

import (
	"context"
	"testing"
)

// TestValidate_valid tests that a variety of configurations under the main
// test source bundle each generate no diagnostics at all, as a
// relatively-simple way to detect accidental regressions.
//
// Any stack configuration directory that we expect should be valid can
// potentially be included in here unless it depends on provider plugins
// to complete validation, since this test cannot supply provider plugins.
func TestValidate_valid(t *testing.T) {
	validConfigDirs := []string{
		"empty",
		"variable-output-roundtrip",
		"variable-output-roundtrip-nested",
	}

	for _, name := range validConfigDirs {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			cfg := loadMainBundleConfigForTest(t, name)

			diags := Validate(ctx, &ValidateRequest{
				Config: cfg,
			})

			// The following will fail the test if there are any error diagnostics.
			reportDiagnosticsForTest(t, diags)

			// We also want to fail if there are just warnings, since the
			// configurations here are supposed to be totally problem-free.
			if len(diags) != 0 {
				t.FailNow() // reportDiagnosticsForTest already showed the diagnostics in the log
			}
		})
	}
}
