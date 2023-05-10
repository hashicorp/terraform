// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"fmt"
	"testing"
)

func TestMetaSHA256(t *testing.T) {
	m := PluginMeta{
		Path: "testdata/current-style-plugins/mockos_mockarch/terraform-foo-bar_v0.0.1",
	}
	hash, err := m.SHA256()
	if err != nil {
		t.Fatalf("failed: %s", err)
	}

	got := fmt.Sprintf("%x", hash)
	want := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" // (hash of empty file)
	if got != want {
		t.Errorf("incorrect hash %s; want %s", got, want)
	}
}
