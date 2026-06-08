// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"os"
	"path/filepath"
	"testing"
)

func TestZshInstaller_InstallUninstall(t *testing.T) {
	// Create a temporary .zshrc file
	td := t.TempDir()
	zshrcPath := filepath.Join(td, ".zshrc")
	if err := os.WriteFile(zshrcPath, []byte("# test zshrc\n"), 0644); err != nil {
		t.Fatalf("failed to create test .zshrc: %v", err)
	}

	installer := &zshInstaller{rc: zshrcPath}

	// Install
	if err := installer.Install("terraform", "/usr/local/bin/terraform"); err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Verify installation
	if !installer.IsInstalled("terraform", "/usr/local/bin/terraform") {
		t.Fatal("Expected terraform to be installed")
	}

	// Verify double-install is rejected
	if err := installer.Install("terraform", "/usr/local/bin/terraform"); err == nil {
		t.Fatal("Expected double-install to fail")
	}

	// Uninstall
	if err := installer.Uninstall("terraform", "/usr/local/bin/terraform"); err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	// Verify uninstallation
	if installer.IsInstalled("terraform", "/usr/local/bin/terraform") {
		t.Fatal("Expected terraform to be uninstalled")
	}
}

func TestGetZshrcPath(t *testing.T) {
	// Create a temporary ZDOTDIR
	zdotdir := t.TempDir()
	zshrcPath := filepath.Join(zdotdir, ".zshrc")
	if err := os.WriteFile(zshrcPath, []byte("# test zshrc\n"), 0644); err != nil {
		t.Fatalf("failed to create test .zshrc: %v", err)
	}

	// Save original ZDOTDIR
	origZdotdir := os.Getenv("ZDOTDIR")
	defer os.Setenv("ZDOTDIR", origZdotdir)

	// Test with ZDOTDIR set
	os.Setenv("ZDOTDIR", zdotdir)
	got := getZshrcPath()
	if got != zshrcPath {
		t.Fatalf("Expected %q, got %q", zshrcPath, got)
	}

	// Test without ZDOTDIR (should return empty since no ~/.zshrc in test env)
	os.Unsetenv("ZDOTDIR")
	got = getZshrcPath()
	// Should return empty or home path if ~/.zshrc exists
	_ = got
}

func TestLineInFile(t *testing.T) {
	td := t.TempDir()
	f := filepath.Join(td, "test.txt")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if !lineInFile(f, "line2") {
		t.Fatal("Expected line2 to be found")
	}
	if lineInFile(f, "line4") {
		t.Fatal("Expected line4 to not be found")
	}
}

func TestAppendToFile(t *testing.T) {
	td := t.TempDir()
	f := filepath.Join(td, "test.txt")
	if err := os.WriteFile(f, []byte("existing\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if err := appendToFile(f, "new"); err != nil {
		t.Fatalf("appendToFile failed: %v", err)
	}

	data, err := os.ReadFile(f)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	expected := "existing\n\nnew\n"
	if string(data) != expected {
		t.Fatalf("Expected %q, got %q", expected, string(data))
	}
}

func TestRemoveFromFile(t *testing.T) {
	td := t.TempDir()
	f := filepath.Join(td, "test.txt")
	if err := os.WriteFile(f, []byte("line1\nline2\nline3\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if err := removeFromFile(f, "line2"); err != nil {
		t.Fatalf("removeFromFile failed: %v", err)
	}

	data, err := os.ReadFile(f)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	expected := "line1\nline3\n"
	if string(data) != expected {
		t.Fatalf("Expected %q, got %q", expected, string(data))
	}
}

func TestIsZsh(t *testing.T) {
	origShell := os.Getenv("SHELL")
	defer os.Setenv("SHELL", origShell)

	os.Setenv("SHELL", "/bin/zsh")
	if !isZsh() {
		t.Fatal("Expected isZsh() to be true for /bin/zsh")
	}

	os.Setenv("SHELL", "/bin/bash")
	if isZsh() {
		t.Fatal("Expected isZsh() to be false for /bin/bash")
	}

	os.Setenv("SHELL", "")
	if isZsh() {
		t.Fatal("Expected isZsh() to be false for empty SHELL")
	}
}

func TestGetFishConfigDir(t *testing.T) {
	// Create a temporary fish config directory
	td := t.TempDir()
	fishDir := filepath.Join(td, "fish")
	if err := os.MkdirAll(fishDir, 0755); err != nil {
		t.Fatalf("failed to create fish dir: %v", err)
	}

	// Save original XDG_CONFIG_HOME
	origXdg := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origXdg)

	os.Setenv("XDG_CONFIG_HOME", td)
	got := getFishConfigDir()
	if got != fishDir {
		t.Fatalf("Expected %q, got %q", fishDir, got)
	}
}

func TestGetBashrcPaths(t *testing.T) {
	// Create a temporary home directory
	td := t.TempDir()
	bashProfilePath := filepath.Join(td, ".bash_profile")
	if err := os.WriteFile(bashProfilePath, []byte("# test bash_profile\n"), 0644); err != nil {
		t.Fatalf("failed to create test .bash_profile: %v", err)
	}

	// Save original HOME
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	os.Setenv("HOME", td)

	paths := getBashrcPaths()
	if len(paths) != 1 || paths[0] != bashProfilePath {
		t.Fatalf("Expected [%q], got %v", bashProfilePath, paths)
	}
}
