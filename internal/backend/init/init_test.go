// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package init

import (
	"reflect"
	"testing"
)

func TestInit_backend(t *testing.T) {
	// Initialize the backends map
	Init(nil)

	backends := []struct {
		Name string
		Type string
	}{
		{"local", "*local.Local"},
		{"remote", "*remote.Remote"},
		{"azurerm", "*azure.Backend"},
		{"consul", "*consul.Backend"},
		{"cos", "*cos.Backend"},
		{"gcs", "*gcs.Backend"},
		{"inmem", "*inmem.Backend"},
		{"pg", "*pg.Backend"},
		{"s3", "*s3.Backend"},
	}

	// Make sure we get the requested backend
	for _, b := range backends {
		t.Run(b.Name, func(t *testing.T) {
			f := Backend(b.Name)
			if f == nil {
				t.Fatalf("backend %q is not present; should be", b.Name)
			}
			bType := reflect.TypeOf(f()).String()
			if bType != b.Type {
				t.Fatalf("expected backend %q to be %q, got: %q", b.Name, b.Type, bType)
			}
		})
	}
}
