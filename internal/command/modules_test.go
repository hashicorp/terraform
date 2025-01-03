// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"encoding/json"
	"os"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/terraform/internal/moduleref"
)

func TestModules_noJsonFlag(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(dir, 0755)
	testCopyDir(t, testFixturePath("modules-nested-dependencies"), dir)
	ui := new(cli.MockUi)
	view, done := testView(t)
	defer testChdir(t, dir)()

	cmd := &ModulesCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{}
	code := cmd.Run(args)
	if code != 0 {
		t.Fatalf("Got a non-zero exit code: %d\n", code)
	}

	actual := done(t).All()

	expectedOutputHuman := `
Modules declared by configuration:
.
├── "other"[./mods/other]
└── "test"[./mods/test]
    └── "test2"[./test2]
        └── "test3"[./test3]

`
	if runtime.GOOS == "windows" {
		expectedOutputHuman = `
Modules declared by configuration:
.
├── "other"[.\mods\other]
└── "test"[.\mods\test]
	└── "test2"[.\test2]
		└── "test3"[.\test3]

`
	}

	if diff := cmp.Diff(expectedOutputHuman, actual); diff != "" {
		t.Fatalf("unexpected output:\n%s\n", diff)
	}
}

func TestModules_noJsonFlag_noModules(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(dir, 0755)
	ui := new(cli.MockUi)
	view, done := testView(t)
	defer testChdir(t, dir)()

	cmd := &ModulesCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{}
	code := cmd.Run(args)
	if code != 0 {
		t.Fatalf("Got a non-zero exit code: %d\n", code)
	}

	actual := done(t).All()

	if diff := cmp.Diff("No modules found in configuration.\n", actual); diff != "" {
		t.Fatalf("unexpected output (-want +got):\n%s", diff)
	}
}

func TestModules_fullCmd(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(dir, 0755)
	testCopyDir(t, testFixturePath("modules-nested-dependencies"), dir)

	ui := new(cli.MockUi)
	view, done := testView(t)
	defer testChdir(t, dir)()

	cmd := &ModulesCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{"-json"}
	code := cmd.Run(args)
	if code != 0 {
		t.Fatalf("Got a non-zero exit code: %d\n", code)
	}

	output := done(t).All()
	compareJSONOutput(t, output, expectedOutputJSON)
}

func TestModules_fullCmd_unreferencedEntries(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(dir, 0755)
	testCopyDir(t, testFixturePath("modules-unreferenced-entries"), dir)

	ui := new(cli.MockUi)
	view, done := testView(t)
	defer testChdir(t, dir)()

	cmd := &ModulesCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{"-json"}
	code := cmd.Run(args)
	if code != 0 {
		t.Fatalf("Got a non-zero exit code: %d\n", code)
	}
	output := done(t).All()
	compareJSONOutput(t, output, expectedOutputJSON)
}

func TestModules_uninstalledModules(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(dir, 0755)
	testCopyDir(t, testFixturePath("modules-uninstalled-entries"), dir)

	ui := new(cli.MockUi)
	view, done := testView(t)
	defer testChdir(t, dir)()

	cmd := &ModulesCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{"-json"}
	code := cmd.Run(args)
	if code == 0 {
		t.Fatal("Expected a non-zero exit code\n")
	}
	output := done(t).All()
	if !strings.Contains(output, "Module not installed") {
		t.Fatalf("expected to see a `not installed` error message: %s\n", output)
	}

	if !strings.Contains(output, `Run "terraform init"`) {
		t.Fatalf("expected error message to ask user to run terraform init: %s\n", output)
	}
}

func compareJSONOutput(t *testing.T, got string, want string) {
	var expected, actual moduleref.Manifest

	if err := json.Unmarshal([]byte(got), &actual); err != nil {
		t.Fatalf("Failed to unmarshal actual JSON: %v", err)
	}

	if err := json.Unmarshal([]byte(want), &expected); err != nil {
		t.Fatalf("Failed to unmarshal expected JSON: %v", err)
	}

	sort.Slice(actual.Records, func(i, j int) bool {
		return actual.Records[i].Key < actual.Records[j].Key
	})
	sort.Slice(expected.Records, func(i, j int) bool {
		return expected.Records[i].Key < expected.Records[j].Key
	})

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("unexpected output, got: %s\n, want:%s\n", got, want)
	}
}

var expectedOutputJSON = `{"format_version":"1.0","modules":[{"key":"test","source":"./mods/test","version":""},{"key":"test2","source":"./test2","version":""},{"key":"test3","source":"./test3","version":""},{"key":"other","source":"./mods/other","version":""}]}`
