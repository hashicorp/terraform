// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestVersionHuman_LogVersion(t *testing.T) {
	testCases := map[string]struct {
		version            string
		platform           string
		providerSelections map[addrs.Provider]*depsfile.ProviderLock
		outdated           bool
		latest             string
		diags              tfdiags.Diagnostics

		wantOutput string
	}{
		"basic": {
			version:    "1.0.0",
			platform:   "linux_amd64",
			wantOutput: "Terraform v1.0.0\non linux_amd64\n\n",
		},
		"with providers": {
			version:  "1.0.0",
			platform: "linux_amd64",
			providerSelections: map[addrs.Provider]*depsfile.ProviderLock{
				addrs.NewDefaultProvider("test1"): depsfile.NewProviderLock(
					addrs.NewDefaultProvider("test1"),
					getproviders.MustParseVersion("1.2.3"),
					// No constraint or hashes
					nil,
					nil,
				),
				addrs.NewDefaultProvider("test2"): depsfile.NewProviderLock(
					addrs.NewDefaultProvider("test2"),
					getproviders.MustParseVersion("0.0.0"), // Not sure how this would happen, but the output is special here.
					// No constraint or hashes
					nil,
					nil,
				),
			},
			wantOutput: `Terraform v1.0.0
on linux_amd64
+ provider registry.terraform.io/hashicorp/test1 v1.2.3
+ provider registry.terraform.io/hashicorp/test2 (unversioned)

`,
		},
		"with outdated + latest": {
			version:            "1.0.0",
			platform:           "linux_amd64",
			providerSelections: map[addrs.Provider]*depsfile.ProviderLock{},
			outdated:           true,
			latest:             "1.16.0",
			diags:              nil,
			wantOutput: `Terraform v1.0.0
on linux_amd64

Your version of Terraform is out of date! The latest version
is 1.16.0. You can update by downloading from https://developer.hashicorp.com/terraform/install

`,
		},
		"with warning": {
			version:            "1.0.0",
			platform:           "linux_amd64",
			providerSelections: map[addrs.Provider]*depsfile.ProviderLock{},
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Warning,
					"Your shoelaces are untied",
					"Watch out, or you'll trip!",
				),
			},
			wantOutput: `
Warning: Your shoelaces are untied

Watch out, or you'll trip!
Terraform v1.0.0
on linux_amd64

`,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewVersion(arguments.ViewHuman, view)

			v.LogVersion(
				tc.version,
				tc.platform,
				tc.providerSelections,
				tc.outdated,
				tc.latest,
				tc.diags,
			)

			got := done(t).All()
			if diff := cmp.Diff(tc.wantOutput, got); diff != "" {
				t.Errorf("unexpected output diff:\n%s", diff)
			}
		})
	}
}
