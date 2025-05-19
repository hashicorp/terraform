// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"reflect"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestStacksPluginConfig_ToMetadata(t *testing.T) {
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
