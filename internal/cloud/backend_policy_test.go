// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"testing"

	"github.com/hashicorp/terraform/internal/policy"
)

func TestCloudPolicyEntitlement(t *testing.T) {
	tests := []struct {
		name string
		b    *Cloud
		want *policy.Entitlement
	}{
		{
			name: "nil backend",
			b:    nil,
			want: nil,
		},
		{
			name: "complete triple",
			b:    &Cloud{Hostname: "app.terraform.io", Organization: "hashicorp", Token: "secret"},
			want: &policy.Entitlement{Host: "app.terraform.io", Token: "secret", Org: "hashicorp"},
		},
		{
			name: "missing host",
			b:    &Cloud{Organization: "hashicorp", Token: "secret"},
			want: nil,
		},
		{
			name: "missing organization",
			b:    &Cloud{Hostname: "app.terraform.io", Token: "secret"},
			want: nil,
		},
		{
			name: "missing token",
			b:    &Cloud{Hostname: "app.terraform.io", Organization: "hashicorp"},
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
