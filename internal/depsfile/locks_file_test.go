// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package depsfile

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestLoadLocksFromFile(t *testing.T) {
	// For ease of test maintenance we treat every file under
	// test-data/locks-files as a test case which is subject
	// at least to testing that it produces an expected set
	// of diagnostics represented via specially-formatted comments
	// in the fixture files (which might be the empty set, if
	// there are no such comments).
	//
	// Some of the files also have additional assertions that
	// are encoded in the test code below. These must pass
	// in addition to the standard diagnostics tests, if present.
	files, err := os.ReadDir("testdata/locks-files")
	if err != nil {
		t.Fatal(err.Error())
	}

	for _, info := range files {
		testName := filepath.Base(info.Name())
		filename := filepath.Join("testdata/locks-files", testName)
		t.Run(testName, func(t *testing.T) {
			f, err := os.Open(filename)
			if err != nil {
				t.Fatal(err.Error())
			}
			defer f.Close()
			const errorPrefix = "# ERROR: "
			const warningPrefix = "# WARNING: "
			wantErrors := map[int]string{}
			wantWarnings := map[int]string{}
			sc := bufio.NewScanner(f)
			lineNum := 1
			for sc.Scan() {
				l := sc.Text()
				if pos := strings.Index(l, errorPrefix); pos != -1 {
					wantSummary := l[pos+len(errorPrefix):]
					wantErrors[lineNum] = wantSummary
				}
				if pos := strings.Index(l, warningPrefix); pos != -1 {
					wantSummary := l[pos+len(warningPrefix):]
					wantWarnings[lineNum] = wantSummary
				}
				lineNum++
			}
			if err := sc.Err(); err != nil {
				t.Fatal(err.Error())
			}

			locks, diags := LoadLocksFromFile(filename)
			gotErrors := map[int]string{}
			gotWarnings := map[int]string{}
			for _, diag := range diags {
				summary := diag.Description().Summary
				if diag.Source().Subject == nil {
					// We don't expect any sourceless diagnostics here.
					t.Errorf("unexpected sourceless diagnostic: %s", summary)
					continue
				}
				lineNum := diag.Source().Subject.Start.Line
				switch sev := diag.Severity(); sev {
				case tfdiags.Error:
					gotErrors[lineNum] = summary
				case tfdiags.Warning:
					gotWarnings[lineNum] = summary
				default:
					t.Errorf("unexpected diagnostic severity %s", sev)
				}
			}

			if diff := cmp.Diff(wantErrors, gotErrors); diff != "" {
				t.Errorf("wrong errors\n%s", diff)
			}
			if diff := cmp.Diff(wantWarnings, gotWarnings); diff != "" {
				t.Errorf("wrong warnings\n%s", diff)
			}

			switch testName {
			// These are the file-specific test assertions. Not all files
			// need custom test assertions in addition to the standard
			// diagnostics assertions implemented above, so the cases here
			// don't need to be exhaustive for all files.
			//
			// Please keep these in alphabetical order so the list is easy
			// to scan!

			case "empty.hcl":
				if got, want := len(locks.providers), 0; got != want {
					t.Errorf("wrong number of providers %d; want %d", got, want)
				}

			case "valid-provider-locks.hcl":
				if got, want := len(locks.providers), 3; got != want {
					t.Errorf("wrong number of providers %d; want %d", got, want)
				}

				t.Run("version-only", func(t *testing.T) {
					if lock := locks.Provider(addrs.MustParseProviderSourceString("terraform.io/test/version-only")); lock != nil {
						if got, want := lock.Version().String(), "1.0.0"; got != want {
							t.Errorf("wrong version\ngot:  %s\nwant: %s", got, want)
						}
						if got, want := providerreqs.VersionConstraintsString(lock.VersionConstraints()), ""; got != want {
							t.Errorf("wrong version constraints\ngot:  %s\nwant: %s", got, want)
						}
						if got, want := len(lock.hashes), 0; got != want {
							t.Errorf("wrong number of hashes %d; want %d", got, want)
						}
					}
				})

				t.Run("version-and-constraints", func(t *testing.T) {
					if lock := locks.Provider(addrs.MustParseProviderSourceString("terraform.io/test/version-and-constraints")); lock != nil {
						if got, want := lock.Version().String(), "1.2.0"; got != want {
							t.Errorf("wrong version\ngot:  %s\nwant: %s", got, want)
						}
						if got, want := providerreqs.VersionConstraintsString(lock.VersionConstraints()), "~> 1.2"; got != want {
							t.Errorf("wrong version constraints\ngot:  %s\nwant: %s", got, want)
						}
						if got, want := len(lock.hashes), 0; got != want {
							t.Errorf("wrong number of hashes %d; want %d", got, want)
						}
					}
				})

				t.Run("all-the-things", func(t *testing.T) {
					if lock := locks.Provider(addrs.MustParseProviderSourceString("terraform.io/test/all-the-things")); lock != nil {
						if got, want := lock.Version().String(), "3.0.10"; got != want {
							t.Errorf("wrong version\ngot:  %s\nwant: %s", got, want)
						}
						if got, want := providerreqs.VersionConstraintsString(lock.VersionConstraints()), ">= 3.0.2"; got != want {
							t.Errorf("wrong version constraints\ngot:  %s\nwant: %s", got, want)
						}
						wantHashes := []providerreqs.Hash{
							providerreqs.MustParseHash("test:placeholder-hash-1"),
							providerreqs.MustParseHash("test:placeholder-hash-2"),
							providerreqs.MustParseHash("test:placeholder-hash-3"),
						}
						if diff := cmp.Diff(wantHashes, lock.hashes); diff != "" {
							t.Errorf("wrong hashes\n%s", diff)
						}
					}
				})

			case "mixed-provider-module-locks.hcl":
				if got, want := len(locks.providers), 1; got != want {
					t.Errorf("wrong number of providers %d; want %d", got, want)
				}
				if got, want := len(locks.modules), 2; got != want {
					t.Errorf("wrong number of modules %d; want %d", got, want)
				}

				t.Run("aws-provider", func(t *testing.T) {
					if lock := locks.Provider(addrs.MustParseProviderSourceString("hashicorp/aws")); lock != nil {
						if got, want := lock.Version().String(), "3.74.0"; got != want {
							t.Errorf("wrong version\ngot:  %s\nwant: %s", got, want)
						}
						if got, want := len(lock.hashes), 2; got != want {
							t.Errorf("wrong number of hashes %d; want %d", got, want)
						}
					} else {
						t.Error("AWS provider lock not found")
					}
				})

				t.Run("local-module", func(t *testing.T) {
					moduleCall := addrs.ModuleCall{Name: "local_example"}.Absolute(addrs.RootModuleInstance)
					if lock := locks.Module(moduleCall); lock != nil {
						if got, want := lock.source, "./modules/local-example"; got != want {
							t.Errorf("wrong source\ngot:  %s\nwant: %s", got, want)
						}
						if got, want := len(lock.hashes), 2; got != want {
							t.Errorf("wrong number of hashes %d; want %d", got, want)
						}
					} else {
						t.Error("Local module lock not found")
					}
				})

				t.Run("registry-module", func(t *testing.T) {
					moduleCall := addrs.ModuleCall{Name: "registry_example"}.Absolute(addrs.RootModuleInstance)
					if lock := locks.Module(moduleCall); lock != nil {
						if got, want := lock.source, "registry.terraform.io/terraform-aws-modules/vpc/aws"; got != want {
							t.Errorf("wrong source\ngot:  %s\nwant: %s", got, want)
						}
						if got, want := len(lock.hashes), 1; got != want {
							t.Errorf("wrong number of hashes %d; want %d", got, want)
						}
					} else {
						t.Error("Registry module lock not found")
					}
				})
			}
		})
	}
}

func TestLoadLocksFromFileAbsent(t *testing.T) {
	t.Run("lock file is a directory", func(t *testing.T) {
		// This can never happen when Terraform is the one generating the
		// lock file, but might arise if the user makes a directory with the
		// lock file's name for some reason. (There is no actual reason to do
		// so, so that would always be a mistake.)
		locks, diags := LoadLocksFromFile("testdata")
		if len(locks.providers) != 0 {
			t.Errorf("returned locks has providers; expected empty locks")
		}
		if !diags.HasErrors() {
			t.Fatalf("LoadLocksFromFile succeeded; want error")
		}
		// This is a generic error message from HCL itself, so upgrading HCL
		// in future might cause a different error message here.
		want := `Failed to read file: The configuration file "testdata" could not be read.`
		got := diags.Err().Error()
		if got != want {
			t.Errorf("wrong error message\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run("lock file doesn't exist", func(t *testing.T) {
		locks, diags := LoadLocksFromFile("testdata/nonexist.hcl")
		if len(locks.providers) != 0 {
			t.Errorf("returned locks has providers; expected empty locks")
		}
		if !diags.HasErrors() {
			t.Fatalf("LoadLocksFromFile succeeded; want error")
		}
		// This is a generic error message from HCL itself, so upgrading HCL
		// in future might cause a different error message here.
		want := `Failed to read file: The configuration file "testdata/nonexist.hcl" could not be read.`
		got := diags.Err().Error()
		if got != want {
			t.Errorf("wrong error message\ngot:  %s\nwant: %s", got, want)
		}
	})
}

func TestSaveLocksToFile(t *testing.T) {
	locks := NewLocks()

	fooProvider := addrs.MustParseProviderSourceString("test/foo")
	barProvider := addrs.MustParseProviderSourceString("test/bar")
	bazProvider := addrs.MustParseProviderSourceString("test/baz")
	booProvider := addrs.MustParseProviderSourceString("test/boo")
	oneDotOh := providerreqs.MustParseVersion("1.0.0")
	oneDotTwo := providerreqs.MustParseVersion("1.2.0")
	atLeastOneDotOh := providerreqs.MustParseVersionConstraints(">= 1.0.0")
	pessimisticOneDotOh := providerreqs.MustParseVersionConstraints("~> 1")
	abbreviatedOneDotTwo := providerreqs.MustParseVersionConstraints("1.2")
	hashes := []providerreqs.Hash{
		providerreqs.MustParseHash("test:cccccccccccccccccccccccccccccccccccccccccccccccc"),
		providerreqs.MustParseHash("test:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		providerreqs.MustParseHash("test:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
	}
	locks.SetProvider(fooProvider, oneDotOh, atLeastOneDotOh, hashes)
	locks.SetProvider(barProvider, oneDotTwo, pessimisticOneDotOh, nil)
	locks.SetProvider(bazProvider, oneDotTwo, nil, nil)
	locks.SetProvider(booProvider, oneDotTwo, abbreviatedOneDotTwo, nil)

	dir := t.TempDir()

	filename := filepath.Join(dir, LockFilePath)
	diags := SaveLocksToFile(locks, filename)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	fileInfo, err := os.Stat(filename)
	if err != nil {
		t.Fatal(err.Error())
	}
	if mode := fileInfo.Mode(); mode&0111 != 0 {
		t.Fatalf("Expected lock file to be non-executable: %o", mode)
	}

	gotContentBytes, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err.Error())
	}
	gotContent := string(gotContentBytes)
	wantContent := `# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/test/bar" {
  version     = "1.2.0"
  constraints = "~> 1.0"
}

provider "registry.terraform.io/test/baz" {
  version = "1.2.0"
}

provider "registry.terraform.io/test/boo" {
  version     = "1.2.0"
  constraints = "1.2.0"
}

provider "registry.terraform.io/test/foo" {
  version     = "1.0.0"
  constraints = ">= 1.0.0"
  hashes = [
    "test:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "test:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    "test:cccccccccccccccccccccccccccccccccccccccccccccccc",
  ]
}
`
	if diff := cmp.Diff(wantContent, gotContent); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}
}

func TestSaveLocksToFileWithModules(t *testing.T) {
	locks := NewLocks()

	// Add a provider and a module
	fooProvider := addrs.MustParseProviderSourceString("test/foo")
	oneDotOh := providerreqs.MustParseVersion("1.0.0")
	locks.SetProvider(fooProvider, oneDotOh, nil, nil)

	localModule := addrs.ModuleCall{Name: "example"}.Absolute(addrs.RootModuleInstance)
	moduleHashes := []providerreqs.Hash{
		providerreqs.MustParseHash("h1:cccccccccccccccccccccccccccccccccccccccccccccccc"),
	}
	locks.SetModule(localModule, "./modules/example", "", moduleHashes)

	dir := t.TempDir()
	filename := filepath.Join(dir, LockFilePath)
	diags := SaveLocksToFile(locks, filename)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	gotContentBytes, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err.Error())
	}
	gotContent := string(gotContentBytes)

	// Check that both provider and module blocks are present
	expected := []string{
		`provider "registry.terraform.io/test/foo"`,
		`module "module.example"`,
		`source = "./modules/example"`,
	}

	for _, exp := range expected {
		if !strings.Contains(gotContent, exp) {
			t.Errorf("Expected %q in lock file", exp)
		}
	}
}

func TestLoadModuleLocks(t *testing.T) {
	// Create a test lock file with both providers and modules
	lockContent := `# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/test/foo" {
  version     = "1.0.0"
  constraints = ">= 1.0.0"
  hashes = [
    "test:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  ]
}

module "module.local_example" {
  source = "./modules/local-example"
  hashes = [
    "h1:cccccccccccccccccccccccccccccccccccccccccccccccc",
    "h1:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
  ]
}

module "module.registry_example" {
  source = "registry.terraform.io/terraform-aws-modules/vpc/aws"
  hashes = [
    "h1:dddddddddddddddddddddddddddddddddddddddddddddddd",
  ]
}
`

	dir := t.TempDir()
	filename := filepath.Join(dir, LockFilePath)

	err := os.WriteFile(filename, []byte(lockContent), 0644)
	if err != nil {
		t.Fatal(err.Error())
	}

	locks, diags := LoadLocksFromFile(filename)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}

	// Verify provider was loaded
	fooProvider := addrs.MustParseProviderSourceString("test/foo")
	if providerLock := locks.Provider(fooProvider); providerLock == nil {
		t.Error("Expected provider lock to be loaded")
	} else {
		if providerLock.Version().String() != "1.0.0" {
			t.Errorf("Expected provider version 1.0.0, got %s", providerLock.Version().String())
		}
	}

	// Verify modules were loaded
	localModule := addrs.ModuleCall{Name: "local_example"}.Absolute(addrs.RootModuleInstance)
	registryModule := addrs.ModuleCall{Name: "registry_example"}.Absolute(addrs.RootModuleInstance)

	if moduleLock := locks.Module(localModule); moduleLock == nil {
		t.Error("Expected local module lock to be loaded")
	} else {
		if moduleLock.source != "./modules/local-example" {
			t.Errorf("Expected local module source './modules/local-example', got %s", moduleLock.source)
		}
		if len(moduleLock.hashes) != 2 {
			t.Errorf("Expected 2 module hashes, got %d", len(moduleLock.hashes))
		}
	}

	if moduleLock := locks.Module(registryModule); moduleLock == nil {
		t.Error("Expected registry module lock to be loaded")
	} else {
		if moduleLock.source != "registry.terraform.io/terraform-aws-modules/vpc/aws" {
			t.Errorf("Expected registry module source, got %s", moduleLock.source)
		}
		if len(moduleLock.hashes) != 1 {
			t.Errorf("Expected 1 module hash, got %d", len(moduleLock.hashes))
		}
	}
}

func TestLoadLocksFromFile_corruptedModuleLocks(t *testing.T) {
	testCases := []struct {
		name        string
		lockContent string
		wantError   string
	}{
		{
			name: "invalid module block name",
			lockContent: `# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

module "invalid-module-name" {
  source = "./test-module"
  hashes = [
    "h1:test-hash",
  ]
}
`,
			wantError: "Module address must start with 'module.' prefix",
		},
		{
			name: "missing source attribute",
			lockContent: `# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

module "module.test" {
  hashes = [
    "h1:test-hash",
  ]
}
`,
			wantError: "Missing required argument",
		},
		{
			name: "invalid hash format",
			lockContent: `# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

module "module.test" {
  source = "./test-module"
  hashes = [
    "invalid-hash-format",
  ]
}
`,
			wantError: "hash string must start with a scheme keyword followed by a colon",
		},
		{
			name: "invalid HCL syntax",
			lockContent: `# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

module "module.test" {
  source = "./test-module"
  hashes = [
    "h1:test-hash"
    # Missing comma causes syntax error
    "h1:test-hash-2",
  ]
}
`,
			wantError: "Missing item separator",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			filename := filepath.Join(dir, LockFilePath)

			err := os.WriteFile(filename, []byte(tc.lockContent), 0644)
			if err != nil {
				t.Fatal(err)
			}

			locks, diags := LoadLocksFromFile(filename)

			if !diags.HasErrors() {
				t.Fatalf("expected error for corrupted lock file, but got none. Locks: %+v", locks)
			}

			errorStr := diags.Err().Error()
			if !strings.Contains(errorStr, tc.wantError) {
				t.Errorf("expected error containing %q, got %q", tc.wantError, errorStr)
			}
		})
	}
}
