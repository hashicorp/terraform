// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduleaddrs

import (
	"testing"
)

func TestDetectGCS(t *testing.T) {
	tableTestDetectorFuncs(t, []struct {
		Input  string
		Output string
	}{
		{
			"www.googleapis.com/storage/v1/bucket/foo",
			"gcs::https://www.googleapis.com/storage/v1/bucket/foo",
		},
		{
			"www.googleapis.com/storage/v1/bucket/foo/bar",
			"gcs::https://www.googleapis.com/storage/v1/bucket/foo/bar",
		},
		{
			"www.googleapis.com/storage/v1/foo/bar.baz",
			"gcs::https://www.googleapis.com/storage/v1/foo/bar.baz",
		},
	})
}
