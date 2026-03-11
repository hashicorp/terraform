// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduleaddrs

import (
	"testing"
)

func TestDetectS3(t *testing.T) {
	tableTestDetectorFuncs(t, []struct {
		Input  string
		Output string
	}{
		// Virtual hosted style
		{
			"bucket.s3.amazonaws.com/foo",
			"s3::https://s3.amazonaws.com/bucket/foo",
		},
		{
			"bucket.s3.amazonaws.com/foo/bar",
			"s3::https://s3.amazonaws.com/bucket/foo/bar",
		},
		{
			"bucket.s3.amazonaws.com/foo/bar.baz",
			"s3::https://s3.amazonaws.com/bucket/foo/bar.baz",
		},
		{
			"bucket.s3-eu-west-1.amazonaws.com/foo",
			"s3::https://s3-eu-west-1.amazonaws.com/bucket/foo",
		},
		{
			"bucket.s3-eu-west-1.amazonaws.com/foo/bar",
			"s3::https://s3-eu-west-1.amazonaws.com/bucket/foo/bar",
		},
		{
			"bucket.s3-eu-west-1.amazonaws.com/foo/bar.baz",
			"s3::https://s3-eu-west-1.amazonaws.com/bucket/foo/bar.baz",
		},
		// 5 parts Virtual hosted-style
		{
			"bucket.s3.eu-west-1.amazonaws.com/foo/bar.baz",
			"s3::https://s3.eu-west-1.amazonaws.com/bucket/foo/bar.baz",
		},
		// Path style
		{
			"s3.amazonaws.com/bucket/foo",
			"s3::https://s3.amazonaws.com/bucket/foo",
		},
		{
			"s3.amazonaws.com/bucket/foo/bar",
			"s3::https://s3.amazonaws.com/bucket/foo/bar",
		},
		{
			"s3.amazonaws.com/bucket/foo/bar.baz",
			"s3::https://s3.amazonaws.com/bucket/foo/bar.baz",
		},
		{
			"s3-eu-west-1.amazonaws.com/bucket/foo",
			"s3::https://s3-eu-west-1.amazonaws.com/bucket/foo",
		},
		{
			"s3-eu-west-1.amazonaws.com/bucket/foo/bar",
			"s3::https://s3-eu-west-1.amazonaws.com/bucket/foo/bar",
		},
		{
			"s3-eu-west-1.amazonaws.com/bucket/foo/bar.baz",
			"s3::https://s3-eu-west-1.amazonaws.com/bucket/foo/bar.baz",
		},
		// Misc tests
		{
			"s3-eu-west-1.amazonaws.com/bucket/foo/bar.baz?version=1234",
			"s3::https://s3-eu-west-1.amazonaws.com/bucket/foo/bar.baz?version=1234",
		},
	})
}
