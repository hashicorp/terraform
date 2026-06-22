// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package remote

import (
	"testing"

	"github.com/hashicorp/terraform/internal/policy"
)

func TestRemotePolicyEntitlement(t *testing.T) {
	tests := []struct {
		name string
		b    *Remote
		want *policy.Entitlement
	}{
		{
			name: "nil backend",
			b:    nil,
			want: nil,
		},
		{
			name: "complete triple",
			b:    &Remote{hostname: "app.terraform.io", organization: "hashicorp", resolvedToken: "secret"},
			want: &policy.Entitlement{Host: "app.terraform.io", Token: "secret", Org: "hashicorp"},
		},
		{
			name: "missing host",
			b:    &Remote{organization: "hashicorp", resolvedToken: "secret"},
			want: nil,
		},
		{
			name: "missing organization",
			b:    &Remote{hostname: "app.terraform.io", resolvedToken: "secret"},
			want: nil,
		},
		{
			name: "missing token",
			b:    &Remote{hostname: "app.terraform.io", organization: "hashicorp"},
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
			if got == nil {
				t.Fatal("expected entitlement, got nil")
			}
			if *got != *tt.want {
				t.Fatalf("unexpected entitlement: got %+v, want %+v", *got, *tt.want)
			}
		})
	}
}
