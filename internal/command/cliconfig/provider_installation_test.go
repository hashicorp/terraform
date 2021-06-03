package cliconfig

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
)

func TestLoadConfig_providerInstallation(t *testing.T) {
	for _, configFile := range []string{"provider-installation", "provider-installation.json"} {
		t.Run(configFile, func(t *testing.T) {
			got, diags := loadConfigFile(filepath.Join(fixtureDir, configFile))
			if diags.HasErrors() {
				t.Errorf("unexpected diagnostics: %s", diags.Err().Error())
			}

			want := &Config{
				ProviderInstallation: []*ProviderInstallation{
					{
						Methods: []*ProviderInstallationMethod{
							{
								Location: ProviderInstallationFilesystemMirror("/tmp/example1"),
								Include:  []string{"example.com/*/*"},
							},
							{
								Location: ProviderInstallationNetworkMirror("https://tf-Mirror.example.com/"),
								Include:  []string{"registry.terraform.io/*/*"},
								Exclude:  []string{"registry.Terraform.io/foobar/*"},
							},
							{
								Location: ProviderInstallationFilesystemMirror("/tmp/example2"),
							},
							{
								Location: ProviderInstallationDirect,
								Exclude:  []string{"example.com/*/*"},
							},
						},

						DevOverrides: map[addrs.Provider]getproviders.PackageLocalDir{
							addrs.MustParseProviderSourceString("hashicorp/boop"):  getproviders.PackageLocalDir(filepath.FromSlash("/tmp/boop")),
							addrs.MustParseProviderSourceString("hashicorp/blorp"): getproviders.PackageLocalDir(filepath.FromSlash("/tmp/blorp")),
						},
					},
				},
			}

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}

func TestLoadConfig_providerInstallationErrors(t *testing.T) {
	_, diags := loadConfigFile(filepath.Join(fixtureDir, "provider-installation-errors"))
	want := `7 problems:

- Invalid provider_installation method block: Unknown provider installation method "not_a_thing" at 2:3.
- Invalid provider_installation method block: Invalid filesystem_mirror block at 1:1: "path" argument is required.
- Invalid provider_installation method block: Invalid network_mirror block at 1:1: "url" argument is required.
- Invalid provider_installation method block: The items inside the provider_installation block at 1:1 must all be blocks.
- Invalid provider_installation method block: The blocks inside the provider_installation block at 1:1 may not have any labels.
- Invalid provider_installation block: The provider_installation block at 9:1 must not have any labels.
- Invalid provider_installation block: The provider_installation block at 11:1 must not be introduced with an equals sign.`

	// The above error messages include only line/column location information
	// and not file location information because HCL 1 does not store
	// information about the filename a location belongs to. (There is a field
	// for it in token.Pos but it's always an empty string in practice.)

	if got := diags.Err().Error(); got != want {
		t.Errorf("wrong diagnostics\ngot:\n%s\nwant:\n%s", got, want)
	}
}
