// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestStacksPluginConfig_ToMetadata(t *testing.T) {
	t.Parallel()
	expected := metadata.Pairs(
		"tfc-address", "https://app.staging.terraform.io",
		"tfc-base-path", "/api/v2/",
		"tfc-display-hostname", "app.staging.terraform.io",
		"tfc-token", "not-a-legit-token",
		"tfc-organization", "example-corp",
		"tfc-project", "example-project",
		"tfc-stack", "example-stack",
		"terraform-binary-path", "",
		"terminal-width", "78",
	)
	inputStruct := StacksPluginConfig{
		Address:             "https://app.staging.terraform.io",
		BasePath:            "/api/v2/",
		DisplayHostname:     "app.staging.terraform.io",
		Token:               "not-a-legit-token",
		OrganizationName:    "example-corp",
		ProjectName:         "example-project",
		StackName:           "example-stack",
		TerraformBinaryPath: "",
		TerminalWidth:       78,
	}
	result := inputStruct.ToMetadata()
	if !reflect.DeepEqual(expected, result) {
		t.Fatalf("Expected: %#v\nGot: %#v\n", expected, result)
	}
}

func TestStacks_resolveDisplayHostname(t *testing.T) {
	tests := []struct {
		name             string
		tfStacksHostname string
		tfCloudHostname  string
		credentialsJSON  string
		wantHostname     string
		wantErrContains  string
		wantWarnContains string
	}{
		{
			name:             "uses TF_STACKS_HOSTNAME first",
			tfStacksHostname: "stacks.example.com",
			tfCloudHostname:  "cloud.example.com",
			credentialsJSON:  `{"credentials":{"cred.example.com":{"token":"x"}}}`,
			wantHostname:     "stacks.example.com",
		},
		{
			name:            "uses TF_CLOUD_HOSTNAME when stacks hostname missing",
			tfCloudHostname: "cloud.example.com",
			credentialsJSON: `{"credentials":{"cred.example.com":{"token":"x"}}}`,
			wantHostname:    "cloud.example.com",
		},
		{
			name:            "uses single credentials hostname",
			credentialsJSON: `{"credentials":{"tfe.company.com":{"token":"x"}}}`,
			wantHostname:    "tfe.company.com",
		},
		{
			name:            "multiple credentials hostnames returns error",
			credentialsJSON: `{"credentials":{"app.terraform.io":{"token":"x"},"tfe.company.com":{"token":"y"}}}`,
			wantErrContains: "Multiple hostnames found in credentials file",
		},
		{
			name:         "missing credentials file falls back to default",
			wantHostname: defaultHostname,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			// Override HOME so cliconfig.ConfigDir() resolves to a temp dir
			// we control, avoiding interference from real credentials on the
			// developer's machine.
			home := t.TempDir()
			t.Setenv("HOME", home)

			if test.credentialsJSON != "" {
				// CredentialsConfigFile() returns $HOME/.terraform.d/credentials.tfrc.json
				configDir := filepath.Join(home, ".terraform.d")
				if err := os.MkdirAll(configDir, 0700); err != nil {
					t.Fatalf("failed to create config dir: %s", err)
				}
				filename := filepath.Join(configDir, "credentials.tfrc.json")
				if err := os.WriteFile(filename, []byte(test.credentialsJSON), 0600); err != nil {
					t.Fatalf("failed to write credentials file: %s", err)
				}
			}

			t.Setenv("TF_STACKS_HOSTNAME", test.tfStacksHostname)
			t.Setenv("TF_CLOUD_HOSTNAME", test.tfCloudHostname)

			c := &StacksCommand{Meta: Meta{}}
			hostname, diags := c.resolveDisplayHostname()

			if test.wantErrContains != "" {
				if !diags.HasErrors() {
					t.Fatalf("expected error diagnostics")
				}
				if got := diags.Err().Error(); !strings.Contains(got, test.wantErrContains) {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, test.wantErrContains)
				}
				return
			}

			if diags.HasErrors() {
				t.Fatalf("unexpected error diagnostics: %s", diags.Err().Error())
			}

			if hostname != test.wantHostname {
				t.Fatalf("wrong hostname\ngot:  %q\nwant: %q", hostname, test.wantHostname)
			}

			if test.wantWarnContains == "" {
				if diags.HasWarnings() {
					t.Fatalf("unexpected warnings: %s", diags.ErrWithWarnings().Error())
				}
				return
			}

			if !diags.HasWarnings() {
				t.Fatalf("expected warning diagnostics")
			}
			if got := diags.ErrWithWarnings().Error(); !strings.Contains(got, test.wantWarnContains) {
				t.Fatalf("wrong warning\ngot:  %s\nwant: %s", got, test.wantWarnContains)
			}
		})
	}
}
