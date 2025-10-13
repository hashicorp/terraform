// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package e2etest

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/apparentlymart/go-versions/versions"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/e2e"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
)

func TestMetaBackend_GetStateStoreProviderFactory(t *testing.T) {
	t.Run("gets the matching factory from local provider cache", func(t *testing.T) {
		if !canRunGoBuild {
			// We're running in a separate-build-then-run context, so we can't
			// currently execute this test which depends on being able to build
			// new executable at runtime.
			//
			// (See the comment on canRunGoBuild's declaration for more information.)
			t.Skip("can't run without building a new provider executable")
		}

		// Set up locks
		locks := depsfile.NewLocks()
		providerAddr := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/simple")
		constraint, err := providerreqs.ParseVersionConstraints(">1.0.0")
		if err != nil {
			t.Fatalf("test setup failed when making constraint: %s", err)
		}
		locks.SetProvider(
			providerAddr,
			versions.MustParseVersion("9.9.9"),
			constraint,
			[]providerreqs.Hash{""},
		)

		// Set up a local provider cache for the test to use
		// 1. Build a binary for the current platform
		simple6Provider := filepath.Join(".", "terraform-provider-simple6")
		simple6ProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)
		// 2. Create a working directory with .terraform/providers directory
		td := t.TempDir()
		t.Chdir(td)
		providerPath := fmt.Sprintf(".terraform/providers/registry.terraform.io/hashicorp/simple/9.9.9/%s", getproviders.CurrentPlatform.String())
		err = os.MkdirAll(providerPath, os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}
		// 3. Move the binary into the cache folder created above.
		os.Rename(simple6ProviderExe, fmt.Sprintf("%s/%s/terraform-provider-simple", td, providerPath))

		config := &configs.StateStore{
			ProviderAddr: tfaddr.MustParseProviderSource("registry.terraform.io/hashicorp/simple"),
			// No other fields necessary for test.
		}

		// Setup the meta and test GetStateStoreProviderFactory
		m := command.Meta{}
		factory, diags := m.GetStateStoreProviderFactory(config, locks)
		if diags.HasErrors() {
			t.Fatalf("unexpected error : %s", err)
		}

		p, _ := factory()
		defer p.Close()
		s := p.GetProviderSchema()
		expectedProviderDescription := "This is terraform-provider-simple v6"
		if s.Provider.Body.Description != expectedProviderDescription {
			t.Fatalf("expected description to be %q, but got %q", expectedProviderDescription, s.Provider.Body.Description)
		}
	})

	// See command/meta_backend_test.go for other test cases
}
