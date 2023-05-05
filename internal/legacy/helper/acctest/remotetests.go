// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acctest

import (
	"net/http"
	"os"
	"testing"
)

// SkipRemoteTestsEnvVar is an environment variable that can be set by a user
// running the tests in an environment with limited network connectivity. By
// default, tests requiring internet connectivity make an effort to skip if no
// internet is available, but in some cases the smoke test will pass even
// though the test should still be skipped.
const SkipRemoteTestsEnvVar = "TF_SKIP_REMOTE_TESTS"

// RemoteTestPrecheck is meant to be run by any unit test that requires
// outbound internet connectivity. The test will be skipped if it's
// unavailable.
func RemoteTestPrecheck(t *testing.T) {
	if os.Getenv(SkipRemoteTestsEnvVar) != "" {
		t.Skipf("skipping test, %s was set", SkipRemoteTestsEnvVar)
	}

	if _, err := http.Get("http://google.com"); err != nil {
		t.Skipf("skipping, internet seems to not be available: %s", err)
	}
}
