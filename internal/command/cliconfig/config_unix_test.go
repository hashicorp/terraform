// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows
// +build !windows

package cliconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func Test_configDir(t *testing.T) {
	const (
		defaultDirPattern = "go_test_terraform_configdir"

		envHome      = "HOME"
		envXdgConfig = "XDG_CONFIG_HOME"
	)

	t.Run("empty home of user", func(t *testing.T) {
		tmpdir, err := os.MkdirTemp("", defaultDirPattern)
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpdir)

		t.Setenv(envHome, tmpdir)
		got, err := configDir()
		if err != nil {
			t.Fatal(err)
		}

		want := filepath.Join(tmpdir, ".terraform.d")

		if got != want {
			t.Errorf("configDir() = %v, want %v", got, want)
		}
	})

	t.Run("has xdg env var, but no actual dir", func(t *testing.T) {
		tmpdir, err := os.MkdirTemp("", defaultDirPattern)
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpdir)

		t.Setenv(envHome, tmpdir)
		t.Setenv(envXdgConfig, filepath.Join(tmpdir, ".config"))

		got, err := configDir()
		if err != nil {
			t.Fatal(err)
		}

		want := filepath.Join(tmpdir, ".terraform.d")

		if got != want {
			t.Errorf("configDir() = %v, want %v", got, want)
		}
	})

	t.Run("terraform config dir exists in home directory", func(t *testing.T) {
		tmpdir, err := os.MkdirTemp("", defaultDirPattern)
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpdir)

		terraformConfigDir := filepath.Join(tmpdir, ".terraform.d")
		err = os.MkdirAll(terraformConfigDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		t.Setenv(envHome, tmpdir)

		got, err := configDir()
		if err != nil {
			t.Fatal(err)
		}

		want := terraformConfigDir
		if got != want {
			t.Errorf("configDir() = %v, want %v", got, want)
		}
	})

	t.Run("has xdg config dir with terraform config", func(t *testing.T) {
		tmpdir, err := os.MkdirTemp("", defaultDirPattern)
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpdir)

		xdgConfigDir := filepath.Join(tmpdir, ".config")
		terraformConfigDir := filepath.Join(xdgConfigDir, "terraform", "terraform.d")
		if err := os.MkdirAll(terraformConfigDir, 0755); err != nil {
			t.Fatal(err)
		}

		t.Setenv(envHome, tmpdir)
		t.Setenv(envXdgConfig, xdgConfigDir)

		got, err := configDir()
		if err != nil {
			t.Fatal(err)
		}

		want := terraformConfigDir

		if got != want {
			t.Errorf("configDir() = %v, want %v", got, want)
		}
	})
}

func Test_configFile(t *testing.T) {
	const (
		defaultDirPattern = "go_test_terraform_configfile"

		envHome      = "HOME"
		envXdgConfig = "XDG_CONFIG_HOME"
	)

	t.Run("empty home of user", func(t *testing.T) {
		tmpdir, err := os.MkdirTemp("", defaultDirPattern)
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpdir)

		t.Setenv(envHome, tmpdir)
		got, err := configFile()
		if err != nil {
			t.Fatal(err)
		}

		want := filepath.Join(tmpdir, ".terraformrc")

		if got != want {
			t.Errorf("configFile() = %v, want %v", got, want)
		}
	})

	t.Run("has xdg env var, but no actual dir", func(t *testing.T) {
		tmpdir, err := os.MkdirTemp("", defaultDirPattern)
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpdir)

		t.Setenv(envHome, tmpdir)
		t.Setenv(envXdgConfig, filepath.Join(tmpdir, ".config"))

		got, err := configFile()
		if err != nil {
			t.Fatal(err)
		}

		want := filepath.Join(tmpdir, ".terraformrc")

		if got != want {
			t.Errorf("configFile() = %v, want %v", got, want)
		}
	})

	t.Run("terraform config file exists in home directory", func(t *testing.T) {
		tmpdir, err := os.MkdirTemp("", defaultDirPattern)
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpdir)

		terraformConfigDir := filepath.Join(tmpdir, ".terraformrc")
		f, err := os.Create(terraformConfigDir)
		if err != nil {
			t.Fatal(err)
		}
		f.Close()

		t.Setenv(envHome, tmpdir)

		got, err := configFile()
		if err != nil {
			t.Fatal(err)
		}

		want := terraformConfigDir
		if got != want {
			t.Errorf("configFile() = %v, want %v", got, want)
		}
	})

	t.Run("has xdg config file with terraform config", func(t *testing.T) {
		tmpdir, err := os.MkdirTemp("", defaultDirPattern)
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpdir)

		xdgConfigDir := filepath.Join(tmpdir, ".config")
		terraformConfigDir := filepath.Join(xdgConfigDir, "terraform")
		terraformConfigFile := filepath.Join(terraformConfigDir, "terraformrc")

		if err := os.MkdirAll(terraformConfigDir, 0755); err != nil {
			t.Fatal(err)
		}

		f, err := os.Create(terraformConfigFile)
		if err != nil {
			t.Fatal(err)
		}
		f.Close()

		t.Setenv(envHome, tmpdir)
		t.Setenv(envXdgConfig, xdgConfigDir)

		got, err := configFile()
		if err != nil {
			t.Fatal(err)
		}

		want := terraformConfigFile

		if got != want {
			t.Errorf("configFile() = %v, want %v", got, want)
		}
	})
}

func Test_configDirXDG(t *testing.T) {
	const (
		defaultDirPattern = "go_test_terraform_configdir_xdg"
	)

	t.Run("terraform config dir exists", func(t *testing.T) {
		tmpdir, err := os.MkdirTemp("", defaultDirPattern)
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpdir)

		// Create the expected terraform config directory structure
		terraformConfigDir := filepath.Join(tmpdir, "terraform")
		if err := os.MkdirAll(terraformConfigDir, 0755); err != nil {
			t.Fatal(err)
		}

		t.Setenv("XDG_CONFIG_HOME", tmpdir)

		got, err := configDirXDG()
		if err != nil {
			t.Fatal(err)
		}

		want := terraformConfigDir
		if got != want {
			t.Errorf("configDirXDG() = %v, want %v", got, want)
		}
	})

	t.Run("terraform config dir does not exist", func(t *testing.T) {
		tmpdir, err := os.MkdirTemp("", defaultDirPattern)
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpdir)

		// Don't create the expected terraform config directory structure, to get expected error

		t.Setenv("XDG_CONFIG_HOME", tmpdir)

		_, err = configDirXDG()
		if err == nil {
			t.Error("configDirXDG() expected error when terraform config dir does not exist, got nil")
		}
	})

	t.Run("no HOME or XDG_CONFIG_HOME defined", func(t *testing.T) {
		t.Setenv("HOME", "")
		t.Setenv("XDG_CONFIG_HOME", "")

		_, err := configDirXDG()
		if err == nil {
			t.Error("configDirXDG() expected error when HOME or XDG_CONFIG_HOME not defined, got nil")
		}
	})
}
