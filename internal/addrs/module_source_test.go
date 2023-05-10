// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	svchost "github.com/hashicorp/terraform-svchost"
)

func TestParseModuleSource(t *testing.T) {
	tests := map[string]struct {
		input   string
		want    ModuleSource
		wantErr string
	}{
		// Local paths
		"local in subdirectory": {
			input: "./child",
			want:  ModuleSourceLocal("./child"),
		},
		"local in subdirectory non-normalized": {
			input: "./nope/../child",
			want:  ModuleSourceLocal("./child"),
		},
		"local in sibling directory": {
			input: "../sibling",
			want:  ModuleSourceLocal("../sibling"),
		},
		"local in sibling directory non-normalized": {
			input: "./nope/../../sibling",
			want:  ModuleSourceLocal("../sibling"),
		},
		"Windows-style local in subdirectory": {
			input: `.\child`,
			want:  ModuleSourceLocal("./child"),
		},
		"Windows-style local in subdirectory non-normalized": {
			input: `.\nope\..\child`,
			want:  ModuleSourceLocal("./child"),
		},
		"Windows-style local in sibling directory": {
			input: `..\sibling`,
			want:  ModuleSourceLocal("../sibling"),
		},
		"Windows-style local in sibling directory non-normalized": {
			input: `.\nope\..\..\sibling`,
			want:  ModuleSourceLocal("../sibling"),
		},
		"an abominable mix of different slashes": {
			input: `./nope\nope/why\./please\don't`,
			want:  ModuleSourceLocal("./nope/nope/why/please/don't"),
		},

		// Registry addresses
		// (NOTE: There is another test function TestParseModuleSourceRegistry
		// which tests this situation more exhaustively, so this is just a
		// token set of cases to see that we are indeed calling into the
		// registry address parser when appropriate.)
		"main registry implied": {
			input: "hashicorp/subnets/cidr",
			want: ModuleSourceRegistry{
				Package: ModuleRegistryPackage{
					Host:         svchost.Hostname("registry.terraform.io"),
					Namespace:    "hashicorp",
					Name:         "subnets",
					TargetSystem: "cidr",
				},
				Subdir: "",
			},
		},
		"main registry implied, subdir": {
			input: "hashicorp/subnets/cidr//examples/foo",
			want: ModuleSourceRegistry{
				Package: ModuleRegistryPackage{
					Host:         svchost.Hostname("registry.terraform.io"),
					Namespace:    "hashicorp",
					Name:         "subnets",
					TargetSystem: "cidr",
				},
				Subdir: "examples/foo",
			},
		},
		"main registry implied, escaping subdir": {
			input: "hashicorp/subnets/cidr//../nope",
			// NOTE: This error is actually being caught by the _remote package_
			// address parser, because any registry parsing failure falls back
			// to that but both of them have the same subdir validation. This
			// case is here to make sure that stays true, so we keep reporting
			// a suitable error when the user writes a registry-looking thing.
			wantErr: `subdirectory path "../nope" leads outside of the module package`,
		},
		"custom registry": {
			input: "example.com/awesomecorp/network/happycloud",
			want: ModuleSourceRegistry{
				Package: ModuleRegistryPackage{
					Host:         svchost.Hostname("example.com"),
					Namespace:    "awesomecorp",
					Name:         "network",
					TargetSystem: "happycloud",
				},
				Subdir: "",
			},
		},
		"custom registry, subdir": {
			input: "example.com/awesomecorp/network/happycloud//examples/foo",
			want: ModuleSourceRegistry{
				Package: ModuleRegistryPackage{
					Host:         svchost.Hostname("example.com"),
					Namespace:    "awesomecorp",
					Name:         "network",
					TargetSystem: "happycloud",
				},
				Subdir: "examples/foo",
			},
		},

		// Remote package addresses
		"github.com shorthand": {
			input: "github.com/hashicorp/terraform-cidr-subnets",
			want: ModuleSourceRemote{
				Package: ModulePackage("git::https://github.com/hashicorp/terraform-cidr-subnets.git"),
			},
		},
		"github.com shorthand, subdir": {
			input: "github.com/hashicorp/terraform-cidr-subnets//example/foo",
			want: ModuleSourceRemote{
				Package: ModulePackage("git::https://github.com/hashicorp/terraform-cidr-subnets.git"),
				Subdir:  "example/foo",
			},
		},
		"git protocol, URL-style": {
			input: "git://example.com/code/baz.git",
			want: ModuleSourceRemote{
				Package: ModulePackage("git://example.com/code/baz.git"),
			},
		},
		"git protocol, URL-style, subdir": {
			input: "git://example.com/code/baz.git//bleep/bloop",
			want: ModuleSourceRemote{
				Package: ModulePackage("git://example.com/code/baz.git"),
				Subdir:  "bleep/bloop",
			},
		},
		"git over HTTPS, URL-style": {
			input: "git::https://example.com/code/baz.git",
			want: ModuleSourceRemote{
				Package: ModulePackage("git::https://example.com/code/baz.git"),
			},
		},
		"git over HTTPS, URL-style, subdir": {
			input: "git::https://example.com/code/baz.git//bleep/bloop",
			want: ModuleSourceRemote{
				Package: ModulePackage("git::https://example.com/code/baz.git"),
				Subdir:  "bleep/bloop",
			},
		},
		"git over HTTPS, URL-style, subdir, query parameters": {
			input: "git::https://example.com/code/baz.git//bleep/bloop?otherthing=blah",
			want: ModuleSourceRemote{
				Package: ModulePackage("git::https://example.com/code/baz.git?otherthing=blah"),
				Subdir:  "bleep/bloop",
			},
		},
		"git over SSH, URL-style": {
			input: "git::ssh://git@example.com/code/baz.git",
			want: ModuleSourceRemote{
				Package: ModulePackage("git::ssh://git@example.com/code/baz.git"),
			},
		},
		"git over SSH, URL-style, subdir": {
			input: "git::ssh://git@example.com/code/baz.git//bleep/bloop",
			want: ModuleSourceRemote{
				Package: ModulePackage("git::ssh://git@example.com/code/baz.git"),
				Subdir:  "bleep/bloop",
			},
		},
		"git over SSH, scp-style": {
			input: "git::git@example.com:code/baz.git",
			want: ModuleSourceRemote{
				// Normalized to URL-style
				Package: ModulePackage("git::ssh://git@example.com/code/baz.git"),
			},
		},
		"git over SSH, scp-style, subdir": {
			input: "git::git@example.com:code/baz.git//bleep/bloop",
			want: ModuleSourceRemote{
				// Normalized to URL-style
				Package: ModulePackage("git::ssh://git@example.com/code/baz.git"),
				Subdir:  "bleep/bloop",
			},
		},

		// NOTE: We intentionally don't test the bitbucket.org shorthands
		// here, because that detector makes direct HTTP tequests to the
		// Bitbucket API and thus isn't appropriate for unit testing.

		"Google Cloud Storage bucket implied, path prefix": {
			input: "www.googleapis.com/storage/v1/BUCKET_NAME/PATH_TO_MODULE",
			want: ModuleSourceRemote{
				Package: ModulePackage("gcs::https://www.googleapis.com/storage/v1/BUCKET_NAME/PATH_TO_MODULE"),
			},
		},
		"Google Cloud Storage bucket, path prefix": {
			input: "gcs::https://www.googleapis.com/storage/v1/BUCKET_NAME/PATH_TO_MODULE",
			want: ModuleSourceRemote{
				Package: ModulePackage("gcs::https://www.googleapis.com/storage/v1/BUCKET_NAME/PATH_TO_MODULE"),
			},
		},
		"Google Cloud Storage bucket implied, archive object": {
			input: "www.googleapis.com/storage/v1/BUCKET_NAME/PATH/TO/module.zip",
			want: ModuleSourceRemote{
				Package: ModulePackage("gcs::https://www.googleapis.com/storage/v1/BUCKET_NAME/PATH/TO/module.zip"),
			},
		},
		"Google Cloud Storage bucket, archive object": {
			input: "gcs::https://www.googleapis.com/storage/v1/BUCKET_NAME/PATH/TO/module.zip",
			want: ModuleSourceRemote{
				Package: ModulePackage("gcs::https://www.googleapis.com/storage/v1/BUCKET_NAME/PATH/TO/module.zip"),
			},
		},

		"Amazon S3 bucket implied, archive object": {
			input: "s3-eu-west-1.amazonaws.com/examplecorp-terraform-modules/vpc.zip",
			want: ModuleSourceRemote{
				Package: ModulePackage("s3::https://s3-eu-west-1.amazonaws.com/examplecorp-terraform-modules/vpc.zip"),
			},
		},
		"Amazon S3 bucket, archive object": {
			input: "s3::https://s3-eu-west-1.amazonaws.com/examplecorp-terraform-modules/vpc.zip",
			want: ModuleSourceRemote{
				Package: ModulePackage("s3::https://s3-eu-west-1.amazonaws.com/examplecorp-terraform-modules/vpc.zip"),
			},
		},

		"HTTP URL": {
			input: "http://example.com/module",
			want: ModuleSourceRemote{
				Package: ModulePackage("http://example.com/module"),
			},
		},
		"HTTPS URL": {
			input: "https://example.com/module",
			want: ModuleSourceRemote{
				Package: ModulePackage("https://example.com/module"),
			},
		},
		"HTTPS URL, archive file": {
			input: "https://example.com/module.zip",
			want: ModuleSourceRemote{
				Package: ModulePackage("https://example.com/module.zip"),
			},
		},
		"HTTPS URL, forced archive file": {
			input: "https://example.com/module?archive=tar",
			want: ModuleSourceRemote{
				Package: ModulePackage("https://example.com/module?archive=tar"),
			},
		},
		"HTTPS URL, forced archive file and checksum": {
			input: "https://example.com/module?archive=tar&checksum=blah",
			want: ModuleSourceRemote{
				// The query string only actually gets processed when we finally
				// do the get, so "checksum=blah" is accepted as valid up
				// at this parsing layer.
				Package: ModulePackage("https://example.com/module?archive=tar&checksum=blah"),
			},
		},

		"absolute filesystem path": {
			// Although a local directory isn't really "remote", we do
			// treat it as such because we still need to do all of the same
			// high-level steps to work with these, even though "downloading"
			// is replaced by a deep filesystem copy instead.
			input: "/tmp/foo/example",
			want: ModuleSourceRemote{
				Package: ModulePackage("file:///tmp/foo/example"),
			},
		},
		"absolute filesystem path, subdir": {
			// This is a funny situation where the user wants to use a
			// directory elsewhere on their system as a package containing
			// multiple modules, but the entry point is not at the root
			// of that subtree, and so they can use the usual subdir
			// syntax to move the package root higher in the real filesystem.
			input: "/tmp/foo//example",
			want: ModuleSourceRemote{
				Package: ModulePackage("file:///tmp/foo"),
				Subdir:  "example",
			},
		},

		"subdir escaping out of package": {
			// This is general logic for all subdir regardless of installation
			// protocol, but we're using a filesystem path here just as an
			// easy placeholder/
			input:   "/tmp/foo//example/../../invalid",
			wantErr: `subdirectory path "../invalid" leads outside of the module package`,
		},

		"relative path without the needed prefix": {
			input: "boop/bloop",
			// For this case we return a generic error message from the addrs
			// layer, but using a specialized error type which our module
			// installer checks for and produces an extra hint for users who
			// were intending to write a local path which then got
			// misinterpreted as a remote source due to the missing prefix.
			// However, the main message is generic here because this is really
			// just a general "this string doesn't match any of our source
			// address patterns" situation, not _necessarily_ about relative
			// local paths.
			wantErr: `Terraform cannot detect a supported external module source type for boop/bloop`,
		},

		"go-getter will accept all sorts of garbage": {
			input: "dfgdfgsd:dgfhdfghdfghdfg/dfghdfghdfg",
			want: ModuleSourceRemote{
				// Unfortunately go-getter doesn't actually reject a totally
				// invalid address like this until getting time, as long as
				// it looks somewhat like a URL.
				Package: ModulePackage("dfgdfgsd:dgfhdfghdfghdfg/dfghdfghdfg"),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			addr, err := ParseModuleSource(test.input)

			if test.wantErr != "" {
				switch {
				case err == nil:
					t.Errorf("unexpected success\nwant error: %s", test.wantErr)
				case err.Error() != test.wantErr:
					t.Errorf("wrong error messages\ngot:  %s\nwant: %s", err.Error(), test.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if diff := cmp.Diff(addr, test.want); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}

}

func TestModuleSourceRemoteFromRegistry(t *testing.T) {
	t.Run("both have subdir", func(t *testing.T) {
		remote := ModuleSourceRemote{
			Package: ModulePackage("boop"),
			Subdir:  "foo",
		}
		registry := ModuleSourceRegistry{
			Subdir: "bar",
		}
		gotAddr := remote.FromRegistry(registry)
		if remote.Subdir != "foo" {
			t.Errorf("FromRegistry modified the reciever; should be pure function")
		}
		if registry.Subdir != "bar" {
			t.Errorf("FromRegistry modified the given address; should be pure function")
		}
		if got, want := gotAddr.Subdir, "foo/bar"; got != want {
			t.Errorf("wrong resolved subdir\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run("only remote has subdir", func(t *testing.T) {
		remote := ModuleSourceRemote{
			Package: ModulePackage("boop"),
			Subdir:  "foo",
		}
		registry := ModuleSourceRegistry{
			Subdir: "",
		}
		gotAddr := remote.FromRegistry(registry)
		if remote.Subdir != "foo" {
			t.Errorf("FromRegistry modified the reciever; should be pure function")
		}
		if registry.Subdir != "" {
			t.Errorf("FromRegistry modified the given address; should be pure function")
		}
		if got, want := gotAddr.Subdir, "foo"; got != want {
			t.Errorf("wrong resolved subdir\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run("only registry has subdir", func(t *testing.T) {
		remote := ModuleSourceRemote{
			Package: ModulePackage("boop"),
			Subdir:  "",
		}
		registry := ModuleSourceRegistry{
			Subdir: "bar",
		}
		gotAddr := remote.FromRegistry(registry)
		if remote.Subdir != "" {
			t.Errorf("FromRegistry modified the reciever; should be pure function")
		}
		if registry.Subdir != "bar" {
			t.Errorf("FromRegistry modified the given address; should be pure function")
		}
		if got, want := gotAddr.Subdir, "bar"; got != want {
			t.Errorf("wrong resolved subdir\ngot:  %s\nwant: %s", got, want)
		}
	})
}

func TestParseModuleSourceRemote(t *testing.T) {

	tests := map[string]struct {
		input          string
		wantString     string
		wantForDisplay string
		wantErr        string
	}{
		"git over HTTPS, URL-style, query parameters": {
			// Query parameters should be correctly appended after the Package
			input:          `git::https://example.com/code/baz.git?otherthing=blah`,
			wantString:     `git::https://example.com/code/baz.git?otherthing=blah`,
			wantForDisplay: `git::https://example.com/code/baz.git?otherthing=blah`,
		},
		"git over HTTPS, URL-style, subdir, query parameters": {
			// Query parameters should be correctly appended after the Package and Subdir
			input:          `git::https://example.com/code/baz.git//bleep/bloop?otherthing=blah`,
			wantString:     `git::https://example.com/code/baz.git//bleep/bloop?otherthing=blah`,
			wantForDisplay: `git::https://example.com/code/baz.git//bleep/bloop?otherthing=blah`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			remote, err := parseModuleSourceRemote(test.input)

			if test.wantErr != "" {
				switch {
				case err == nil:
					t.Errorf("unexpected success\nwant error: %s", test.wantErr)
				case err.Error() != test.wantErr:
					t.Errorf("wrong error messages\ngot:  %s\nwant: %s", err.Error(), test.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if got, want := remote.String(), test.wantString; got != want {
				t.Errorf("wrong String() result\ngot:  %s\nwant: %s", got, want)
			}
			if got, want := remote.ForDisplay(), test.wantForDisplay; got != want {
				t.Errorf("wrong ForDisplay() result\ngot:  %s\nwant: %s", got, want)
			}
		})
	}
}

func TestParseModuleSourceRegistry(t *testing.T) {
	// We test parseModuleSourceRegistry alone here, in addition to testing
	// it indirectly as part of TestParseModuleSource, because general
	// module parsing unfortunately eats all of the error situations from
	// registry passing by falling back to trying for a direct remote package
	// address.

	// Historical note: These test cases were originally derived from the
	// ones in the old internal/registry/regsrc package that the
	// ModuleSourceRegistry type is replacing. That package had the notion
	// of "normalized" addresses as separate from the original user input,
	// but this new implementation doesn't try to preserve the original
	// user input at all, and so the main string output is always normalized.
	//
	// That package also had some behaviors to turn the namespace, name, and
	// remote system portions into lowercase, but apparently we didn't
	// actually make use of that in the end and were preserving the case
	// the user provided in the input, and so for backward compatibility
	// we're continuing to do that here, at the expense of now making the
	// "ForDisplay" output case-preserving where its predecessor in the
	// old package wasn't. The main Terraform Registry at registry.terraform.io
	// is itself case-insensitive anyway, so our case-preserving here is
	// entirely for the benefit of existing third-party registry
	// implementations that might be case-sensitive, which we must remain
	// compatible with now.

	tests := map[string]struct {
		input           string
		wantString      string
		wantForDisplay  string
		wantForProtocol string
		wantErr         string
	}{
		"public registry": {
			input:           `hashicorp/consul/aws`,
			wantString:      `registry.terraform.io/hashicorp/consul/aws`,
			wantForDisplay:  `hashicorp/consul/aws`,
			wantForProtocol: `hashicorp/consul/aws`,
		},
		"public registry with subdir": {
			input:           `hashicorp/consul/aws//foo`,
			wantString:      `registry.terraform.io/hashicorp/consul/aws//foo`,
			wantForDisplay:  `hashicorp/consul/aws//foo`,
			wantForProtocol: `hashicorp/consul/aws`,
		},
		"public registry using explicit hostname": {
			input:           `registry.terraform.io/hashicorp/consul/aws`,
			wantString:      `registry.terraform.io/hashicorp/consul/aws`,
			wantForDisplay:  `hashicorp/consul/aws`,
			wantForProtocol: `hashicorp/consul/aws`,
		},
		"public registry with mixed case names": {
			input:           `HashiCorp/Consul/aws`,
			wantString:      `registry.terraform.io/HashiCorp/Consul/aws`,
			wantForDisplay:  `HashiCorp/Consul/aws`,
			wantForProtocol: `HashiCorp/Consul/aws`,
		},
		"private registry with non-standard port": {
			input:           `Example.com:1234/HashiCorp/Consul/aws`,
			wantString:      `example.com:1234/HashiCorp/Consul/aws`,
			wantForDisplay:  `example.com:1234/HashiCorp/Consul/aws`,
			wantForProtocol: `HashiCorp/Consul/aws`,
		},
		"private registry with IDN hostname": {
			input:           `Испытание.com/HashiCorp/Consul/aws`,
			wantString:      `испытание.com/HashiCorp/Consul/aws`,
			wantForDisplay:  `испытание.com/HashiCorp/Consul/aws`,
			wantForProtocol: `HashiCorp/Consul/aws`,
		},
		"private registry with IDN hostname and non-standard port": {
			input:           `Испытание.com:1234/HashiCorp/Consul/aws//Foo`,
			wantString:      `испытание.com:1234/HashiCorp/Consul/aws//Foo`,
			wantForDisplay:  `испытание.com:1234/HashiCorp/Consul/aws//Foo`,
			wantForProtocol: `HashiCorp/Consul/aws`,
		},
		"invalid hostname": {
			input:   `---.com/HashiCorp/Consul/aws`,
			wantErr: `invalid module registry hostname "---.com"; internationalized domain names must be given as direct unicode characters, not in punycode`,
		},
		"hostname with only one label": {
			// This was historically forbidden in our initial implementation,
			// so we keep it forbidden to avoid newly interpreting such
			// addresses as registry addresses rather than remote source
			// addresses.
			input:   `foo/var/baz/qux`,
			wantErr: `invalid module registry hostname: must contain at least one dot`,
		},
		"invalid target system characters": {
			input:   `foo/var/no-no-no`,
			wantErr: `invalid target system "no-no-no": must be between one and 64 ASCII letters or digits`,
		},
		"invalid target system length": {
			input:   `foo/var/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaah`,
			wantErr: `invalid target system "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaah": must be between one and 64 ASCII letters or digits`,
		},
		"invalid namespace": {
			input:   `boop!/var/baz`,
			wantErr: `invalid namespace "boop!": must be between one and 64 characters, including ASCII letters, digits, dashes, and underscores, where dashes and underscores may not be the prefix or suffix`,
		},
		"missing part with explicit hostname": {
			input:   `foo.com/var/baz`,
			wantErr: `source address must have three more components after the hostname: the namespace, the name, and the target system`,
		},
		"errant query string": {
			input:   `foo/var/baz?otherthing`,
			wantErr: `module registry addresses may not include a query string portion`,
		},
		"github.com": {
			// We don't allow using github.com like a module registry because
			// that conflicts with the historically-supported shorthand for
			// installing directly from GitHub-hosted git repositories.
			input:   `github.com/HashiCorp/Consul/aws`,
			wantErr: `can't use "github.com" as a module registry host, because it's reserved for installing directly from version control repositories`,
		},
		"bitbucket.org": {
			// We don't allow using bitbucket.org like a module registry because
			// that conflicts with the historically-supported shorthand for
			// installing directly from BitBucket-hosted git repositories.
			input:   `bitbucket.org/HashiCorp/Consul/aws`,
			wantErr: `can't use "bitbucket.org" as a module registry host, because it's reserved for installing directly from version control repositories`,
		},
		"local path from current dir": {
			// Can't use a local path when we're specifically trying to parse
			// a _registry_ source address.
			input:   `./boop`,
			wantErr: `can't use local directory "./boop" as a module registry address`,
		},
		"local path from parent dir": {
			// Can't use a local path when we're specifically trying to parse
			// a _registry_ source address.
			input:   `../boop`,
			wantErr: `can't use local directory "../boop" as a module registry address`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			addrI, err := ParseModuleSourceRegistry(test.input)

			if test.wantErr != "" {
				switch {
				case err == nil:
					t.Errorf("unexpected success\nwant error: %s", test.wantErr)
				case err.Error() != test.wantErr:
					t.Errorf("wrong error messages\ngot:  %s\nwant: %s", err.Error(), test.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			addr, ok := addrI.(ModuleSourceRegistry)
			if !ok {
				t.Fatalf("wrong address type %T; want %T", addrI, addr)
			}

			if got, want := addr.String(), test.wantString; got != want {
				t.Errorf("wrong String() result\ngot:  %s\nwant: %s", got, want)
			}
			if got, want := addr.ForDisplay(), test.wantForDisplay; got != want {
				t.Errorf("wrong ForDisplay() result\ngot:  %s\nwant: %s", got, want)
			}
			if got, want := addr.Package.ForRegistryProtocol(), test.wantForProtocol; got != want {
				t.Errorf("wrong ForRegistryProtocol() result\ngot:  %s\nwant: %s", got, want)
			}
		})
	}
}
