// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"testing"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/policy"
)

// entitlementBackend is a backend.Backend that also implements
// policy.EntitlementProvider.
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

func TestBackendPolicyEntitlement(t *testing.T) {
	ent := &policy.Entitlement{Host: "app.terraform.io", Token: "secret", Org: "hashicorp"}

	tests := []struct {
		name string
		be   backend.Backend
		want *policy.Entitlement
	}{
		{
			name: "backend implements EntitlementProvider",
			be:   &entitlementBackend{ent: ent},
			want: ent,
		},
		{
			name: "backend provides nil entitlement",
			be:   &entitlementBackend{ent: nil},
			want: nil,
		},
		{
			name: "backend does not implement EntitlementProvider",
			be:   &plainBackend{},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := backendPolicyEntitlement(tt.be)
			if got != tt.want {
				t.Fatalf("unexpected entitlement: got %+v, want %+v", got, tt.want)
			}
		})
	}
}
