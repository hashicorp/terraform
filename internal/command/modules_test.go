// Copyright IBM Corp. 2014, 2026
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
	"github.com/hashicorp/terraform/internal/backend"
	backendInit "github.com/hashicorp/terraform/internal/backend/init"
	backendCloud "github.com/hashicorp/terraform/internal/cloud"
	"github.com/hashicorp/terraform/internal/moduleref"
)

func TestModules_noJsonFlag(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(dir, 0755)
	testCopyDir(t, testFixturePath("modules-nested-dependencies"), dir)
	t.Chdir(dir)

	ui := new(cli.MockUi)
	view, done := testView(t)

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
	t.Chdir(dir)

	ui := new(cli.MockUi)
	view, done := testView(t)

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
	t.Chdir(dir)

	ui := new(cli.MockUi)
	view, done := testView(t)

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
	t.Chdir(dir)

	ui := new(cli.MockUi)
	view, done := testView(t)

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
	t.Chdir(dir)

	ui := new(cli.MockUi)
	view, done := testView(t)

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

func TestModules_constVariable(t *testing.T) {
	t.Run("missing value", func(t *testing.T) {
		wd := tempWorkingDirFixture(t, "dynamic-module-sources/command-with-const-var")
		t.Chdir(wd.RootModuleDir())

		ui := cli.NewMockUi()
		view, done := testView(t)

		cmd := &ModulesCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				Ui:               ui,
				View:             view,
				WorkingDir:       wd,
			},
		}

		args := []string{}
		code := cmd.Run(args)
		if code == 0 {
			t.Fatalf("expected error, got 0")
		}

		output := done(t).All()
		if !strings.Contains(output, "No value for required variable") {
			t.Fatalf("expected missing variable error, got: %s", output)
		}
	})

	t.Run("value via cli", func(t *testing.T) {
		wd := tempWorkingDirFixture(t, "dynamic-module-sources/command-with-const-var")
		t.Chdir(wd.RootModuleDir())

		ui := cli.NewMockUi()
		view, done := testView(t)

		cmd := &ModulesCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				Ui:               ui,
				View:             view,
				WorkingDir:       wd,
			},
		}

		args := []string{}
		code := cmd.Run(append(args, "-var", "module_name=child"))
		if code != 0 {
			t.Fatalf("Got a non-zero exit code: %d\n%s", code, done(t).All())
		}

		actual := done(t).All()

		expectedOutputHuman := `
Modules declared by configuration:
.
└── "child"[./modules/child]

`
		if runtime.GOOS == "windows" {
			expectedOutputHuman = `
Modules declared by configuration:
.
└── "child"[.\modules\child]

`
		}

		if diff := cmp.Diff(expectedOutputHuman, actual); diff != "" {
			t.Fatalf("unexpected output:\n%s\n", diff)
		}
	})

	t.Run("value via backend", func(t *testing.T) {
		server := cloudTestServerWithVars(t)
		defer server.Close()
		d := testDisco(server)

		previousBackend := backendInit.Backend("cloud")
		backendInit.Set("cloud", func() backend.Backend { return backendCloud.New(d) })
		defer backendInit.Set("cloud", previousBackend)

		wd := tempWorkingDirFixture(t, "dynamic-module-sources/command-with-const-var-cloud-backend")
		t.Chdir(wd.RootModuleDir())

		ui := cli.NewMockUi()
		view, done := testView(t)

		cmd := &ModulesCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				Ui:               ui,
				View:             view,
				WorkingDir:       wd,
				Services:         d,
			},
		}

		args := []string{}
		code := cmd.Run(args)
		if code != 0 {
			t.Fatalf("Got a non-zero exit code: %d\n%s", code, done(t).All())
		}

		actual := done(t).All()

		expectedOutputHuman := `
Modules declared by configuration:
.
└── "child"[./modules/example]

`
		if runtime.GOOS == "windows" {
			expectedOutputHuman = `
Modules declared by configuration:
.
└── "child"[.\modules\example]

`
		}

		if diff := cmp.Diff(expectedOutputHuman, actual); diff != "" {
			t.Fatalf("unexpected output:\n%s\n", diff)
		}
	})
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
