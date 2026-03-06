// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package migrate

import (
	"testing"
)

func TestRegistryAll(t *testing.T) {
	r := NewRegistry()
	all := r.All()
	if len(all) == 0 {
		t.Fatal("expected registry to contain at least one migration, but it was empty")
	}
}

func TestRegistryFind(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		r := NewRegistry()
		m, err := r.Find("hashicorp/aws/v3-to-v4")
		if err != nil {
			t.Fatalf("expected to find migration, got error: %s", err)
		}
		if m.ID() != "hashicorp/aws/v3-to-v4" {
			t.Fatalf("expected ID %q, got %q", "hashicorp/aws/v3-to-v4", m.ID())
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := NewRegistry()
		_, err := r.Find("nonexistent/provider/migration")
		if err == nil {
			t.Fatal("expected error for nonexistent migration, got nil")
		}
	})
}
