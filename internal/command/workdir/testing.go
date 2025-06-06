// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workdir

import (
	"testing"

	version "github.com/hashicorp/go-version"
)

// getTestProviderState is a test helper that returns a state representation
// of a provider used for managing state via pluggable state storage.
func getTestProviderState(t *testing.T, semVer, fqn string) *Provider {
	t.Helper()

	ver, err := version.NewSemver(semVer)
	if err != nil {
		t.Fatalf("test setup failed when creating version.Version: %s", err)
	}

	source := &Source{}
	err = source.UnmarshalText([]byte(fqn))
	if err != nil {
		t.Fatalf("test setup failed when creating ProviderSource: %s", err)
	}

	return &Provider{
		Version: ver,
		Source:  source,
	}
}
