// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package regsrc

import (
	"testing"
)

func TestModule(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		wantString  string
		wantDisplay string
		wantNorm    string
		wantErr     bool
	}{
		{
			name:        "public registry",
			source:      "hashicorp/consul/aws",
			wantString:  "hashicorp/consul/aws",
			wantDisplay: "hashicorp/consul/aws",
			wantNorm:    "hashicorp/consul/aws",
			wantErr:     false,
		},
		{
			name:        "public registry, submodule",
			source:      "hashicorp/consul/aws//foo",
			wantString:  "hashicorp/consul/aws//foo",
			wantDisplay: "hashicorp/consul/aws//foo",
			wantNorm:    "hashicorp/consul/aws//foo",
			wantErr:     false,
		},
		{
			name:        "public registry, explicit host",
			source:      "registry.terraform.io/hashicorp/consul/aws",
			wantString:  "registry.terraform.io/hashicorp/consul/aws",
			wantDisplay: "hashicorp/consul/aws",
			wantNorm:    "hashicorp/consul/aws",
			wantErr:     false,
		},
		{
			name:        "public registry, mixed case",
			source:      "HashiCorp/Consul/aws",
			wantString:  "HashiCorp/Consul/aws",
			wantDisplay: "hashicorp/consul/aws",
			wantNorm:    "hashicorp/consul/aws",
			wantErr:     false,
		},
		{
			name:        "private registry, custom port",
			source:      "Example.com:1234/HashiCorp/Consul/aws",
			wantString:  "Example.com:1234/HashiCorp/Consul/aws",
			wantDisplay: "example.com:1234/hashicorp/consul/aws",
			wantNorm:    "example.com:1234/hashicorp/consul/aws",
			wantErr:     false,
		},
		{
			name:        "IDN registry",
			source:      "Испытание.com/HashiCorp/Consul/aws",
			wantString:  "Испытание.com/HashiCorp/Consul/aws",
			wantDisplay: "испытание.com/hashicorp/consul/aws",
			wantNorm:    "xn--80akhbyknj4f.com/hashicorp/consul/aws",
			wantErr:     false,
		},
		{
			name:       "IDN registry, submodule, custom port",
			source:     "Испытание.com:1234/HashiCorp/Consul/aws//Foo",
			wantString: "Испытание.com:1234/HashiCorp/Consul/aws//Foo",
			// Note we DO lowercase submodule names. This might causes issues on
			// some filesystems (e.g. HFS+) that are case-sensitive where
			// //modules/Foo and //modules/foo describe different paths, but
			// it's less confusing in general just to not support that. Any user
			// with a module with submodules in both cases is already asking for
			// portability issues, and terraform can ensure it does
			// case-insensitive search for the dir in those cases.
			wantDisplay: "испытание.com:1234/hashicorp/consul/aws//foo",
			wantNorm:    "xn--80akhbyknj4f.com:1234/hashicorp/consul/aws//foo",
			wantErr:     false,
		},
		{
			name:    "invalid host",
			source:  "---.com/HashiCorp/Consul/aws",
			wantErr: true,
		},
		{
			name:    "invalid format",
			source:  "foo/var/baz/qux",
			wantErr: true,
		},
		{
			name:    "invalid suffix",
			source:  "foo/var/baz?otherthing",
			wantErr: true,
		},
		{
			name:    "valid host, invalid format",
			source:  "foo.com/var/baz?otherthing",
			wantErr: true,
		},
		{
			name:    "disallow github",
			source:  "github.com/HashiCorp/Consul/aws",
			wantErr: true,
		},
		{
			name:    "disallow bitbucket",
			source:  "bitbucket.org/HashiCorp/Consul/aws",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseModuleSource(tt.source)

			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseModuleSource() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				return
			}

			if v := got.String(); v != tt.wantString {
				t.Fatalf("String() = %v, want %v", v, tt.wantString)
			}
			if v := got.Display(); v != tt.wantDisplay {
				t.Fatalf("Display() = %v, want %v", v, tt.wantDisplay)
			}
			if v := got.Normalized(); v != tt.wantNorm {
				t.Fatalf("Normalized() = %v, want %v", v, tt.wantNorm)
			}

			gotDisplay, err := ParseModuleSource(tt.wantDisplay)
			if err != nil {
				t.Fatalf("ParseModuleSource(wantDisplay) error = %v", err)
			}
			if !got.Equal(gotDisplay) {
				t.Fatalf("Equal() failed for %s and %s", tt.source, tt.wantDisplay)
			}
		})
	}
}
