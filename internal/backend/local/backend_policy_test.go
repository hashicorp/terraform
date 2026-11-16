// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"testing"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/policy"
)

// entitlementBackend is a backend.Backend that also implements
// policy.EntitlementProvider, for testing Local's delegation.
type entitlementBackend struct {
	backend.Backend
	ent *policy.Entitlement
}

func (b *entitlementBackend) PolicyEntitlement() *policy.Entitlement { return b.ent }

// plainBackend is a backend.Backend that does not implement
// policy.EntitlementProvider.
type plainBackend struct {
	backend.Backend
}

func TestLocalPolicyEntitlement(t *testing.T) {
	ent := &policy.Entitlement{Host: "app.terraform.io", Token: "secret", Org: "hashicorp"}

	tests := []struct {
		name string
		b    *Local
		want *policy.Entitlement
	}{
		{
			name: "nil backend",
			b:    nil,
			want: nil,
		},
		{
			name: "nil wrapped backend",
			b:    &Local{},
			want: nil,
		},
		{
			name: "wrapped backend provides an entitlement",
			b:    &Local{Backend: &entitlementBackend{ent: ent}},
			want: ent,
		},
		{
			name: "wrapped backend provides nil entitlement",
			b:    &Local{Backend: &entitlementBackend{ent: nil}},
			want: nil,
		},
		{
			name: "wrapped backend is not an EntitlementProvider",
			b:    &Local{Backend: &plainBackend{}},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.b.PolicyEntitlement()
			if tt.want == nil {
				if got != nil {
					t.Fatalf("expected nil entitlement, got %+v", got)
				}
				return
			}
			if got != tt.want {
				t.Fatalf("unexpected entitlement: got %+v, want %+v", got, tt.want)
			}
		})
	}
}
