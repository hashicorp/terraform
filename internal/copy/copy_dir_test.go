// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package copy

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// TestCopyDir_symlinks sets up a directory with two submodules,
// one being a symlink to the other
//
// The resultant file structure is as follows:
// 	├── modules
//  │   ├── symlink-module -> test-module
//  │   └── test-module
//  │       └── main.tf
//  └── target
//     ├── symlink-module -> test-module
//     └── test-module
//         └── main.tf

func TestCopyDir_symlinks(t *testing.T) {
	tmpdir := t.TempDir()

	moduleDir := filepath.Join(tmpdir, "modules")
	err := os.Mkdir(moduleDir, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	subModuleDir := filepath.Join(moduleDir, "test-module")
	err = os.Mkdir(subModuleDir, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile(filepath.Join(subModuleDir, "main.tf"), []byte("hello"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.Symlink("test-module", filepath.Join(moduleDir, "symlink-module"))
	if err != nil {
		t.Fatal(err)
	}

	targetDir := filepath.Join(tmpdir, "target")
	os.Mkdir(targetDir, os.ModePerm)

	err = CopyDir(targetDir, moduleDir)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = os.Lstat(filepath.Join(targetDir, "test-module", "main.tf")); os.IsNotExist(err) {
		t.Fatal("target test-module/main.tf was not created")
	}

	if _, err = os.Lstat(filepath.Join(targetDir, "symlink-module", "main.tf")); os.IsNotExist(err) {
		t.Fatal("target symlink-module/main.tf was not created")
	}
}

func TestCopyDir_symlink_file(t *testing.T) {
	tmpdir := t.TempDir()

	moduleDir := filepath.Join(tmpdir, "modules")
	err := os.Mkdir(moduleDir, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile(filepath.Join(moduleDir, "main.tf"), []byte("hello"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.Symlink("main.tf", filepath.Join(moduleDir, "symlink.tf"))
	if err != nil {
		t.Fatal(err)
	}

	targetDir := filepath.Join(tmpdir, "target")
	os.Mkdir(targetDir, os.ModePerm)

	err = CopyDir(targetDir, moduleDir)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = os.Lstat(filepath.Join(targetDir, "main.tf")); os.IsNotExist(err) {
		t.Fatal("target/main.tf was not created")
	}

	if _, err = os.Lstat(filepath.Join(targetDir, "symlink.tf")); os.IsNotExist(err) {
		t.Fatal("target/symlink.tf was not created")
	}
}
