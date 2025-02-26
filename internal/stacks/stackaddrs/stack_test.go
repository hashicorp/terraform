// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackaddrs

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
)

func TestStackInstance_Contains(t *testing.T) {
	tests := []struct {
		name   string
		parent StackInstance
		arg    StackInstance
		want   bool
	}{
		{
			name:   "root contains root",
			parent: RootStackInstance,
			arg:    RootStackInstance,
			want:   true,
		},
		{
			name:   "root contains child",
			parent: RootStackInstance,
			arg:    RootStackInstance.Child("child", addrs.NoKey),
			want:   true,
		},
		{
			name:   "child does not contain root",
			parent: RootStackInstance.Child("child", addrs.NoKey),
			arg:    RootStackInstance,
			want:   false,
		},
		{
			name:   "child contains itself",
			parent: RootStackInstance.Child("child", addrs.NoKey),
			arg:    RootStackInstance.Child("child", addrs.NoKey),
			want:   true,
		},
		{
			name:   "child contains grandchild",
			parent: RootStackInstance.Child("child", addrs.NoKey),
			arg:    RootStackInstance.Child("child", addrs.NoKey).Child("grandchild", addrs.NoKey),
			want:   true,
		},
		{
			name:   "grandchild does not contain child",
			parent: RootStackInstance.Child("child", addrs.NoKey).Child("grandchild", addrs.NoKey),
			arg:    RootStackInstance.Child("child", addrs.NoKey),
			want:   false,
		},
		{
			name:   "different keys are not contained",
			parent: RootStackInstance.Child("child", addrs.NoKey),
			arg:    RootStackInstance.Child("child", addrs.IntKey(1)),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.parent.Contains(tt.arg); got != tt.want {
				t.Errorf("StackInstance.Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
