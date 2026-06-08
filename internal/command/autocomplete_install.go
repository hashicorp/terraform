// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/hashicorp/go-multierror"
	"github.com/posener/complete/cmd/install"
)

// AutocompleteInstall installs shell autocompletion for Terraform.
// It wraps posener/complete's install package with support for ZDOTDIR.
func AutocompleteInstall(cmd string) error {
	var err error

	// Install for bash and fish using posener/complete
	// This will skip zsh if ~/.zshrc doesn't exist, which is the case
	// when ZDOTDIR is set to a custom location
	err = install.Install(cmd)

	// Check if ZDOTDIR is set and install zsh completion there
	if zdotdir := os.Getenv("ZDOTDIR"); zdotdir != "" {
		zshrcPath := filepath.Join(zdotdir, ".zshrc")
		if _, statErr := os.Stat(zshrcPath); statErr == nil {
			zshInstaller := &zshInstaller{rc: zshrcPath}
			bin, binErr := getBinaryPath()
			if binErr != nil {
				return multierror.Append(err, binErr)
			}
			zshErr := zshInstaller.Install(cmd, bin)
			if zshErr != nil {
				// Ignore "already installed" errors
				if zshErr.Error() != fmt.Sprintf("already installed in %s", zshrcPath) {
					err = multierror.Append(err, zshErr)
				}
			}
		}
	}

	return err
}

// AutocompleteUninstall uninstalls shell autocompletion for Terraform.
// It wraps posener/complete's uninstall package with support for ZDOTDIR.
func AutocompleteUninstall(cmd string) error {
	var err error

	// Uninstall for bash and fish using posener/complete
	err = install.Uninstall(cmd)

	// Check if ZDOTDIR is set and uninstall zsh completion from there
	if zdotdir := os.Getenv("ZDOTDIR"); zdotdir != "" {
		zshrcPath := filepath.Join(zdotdir, ".zshrc")
		if _, statErr := os.Stat(zshrcPath); statErr == nil {
			zshInstaller := &zshInstaller{rc: zshrcPath}
			bin, binErr := getBinaryPath()
			if binErr != nil {
				return multierror.Append(err, binErr)
			}
			zshErr := zshInstaller.Uninstall(cmd, bin)
			if zshErr != nil {
				// Ignore "not installed" errors
				if zshErr.Error() != fmt.Sprintf("does not installed in %s", zshrcPath) {
					err = multierror.Append(err, zshErr)
				}
			}
		}
	}

	return err
}

// getBinaryPath returns the absolute path of the current binary.
func getBinaryPath() (string, error) {
	bin, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Abs(bin)
}

// zshInstaller implements zsh autocompletion installation.
// This is a copy of the posener/complete zsh installer with ZDOTDIR support.
type zshInstaller struct {
	rc string
}

func (z zshInstaller) IsInstalled(cmd, bin string) bool {
	completeCmd := z.cmd(cmd, bin)
	return lineInFile(z.rc, completeCmd)
}

func (z zshInstaller) Install(cmd, bin string) error {
	if z.IsInstalled(cmd, bin) {
		return fmt.Errorf("already installed in %s", z.rc)
	}

	completeCmd := z.cmd(cmd, bin)
	bashCompInit := "autoload -U +X bashcompinit && bashcompinit"
	if !lineInFile(z.rc, bashCompInit) {
		completeCmd = bashCompInit + "\n" + completeCmd
	}

	return appendToFile(z.rc, completeCmd)
}

func (z zshInstaller) Uninstall(cmd, bin string) error {
	if !z.IsInstalled(cmd, bin) {
		return fmt.Errorf("does not installed in %s", z.rc)
	}

	completeCmd := z.cmd(cmd, bin)
	return removeFromFile(z.rc, completeCmd)
}

func (zshInstaller) cmd(cmd, bin string) string {
	return fmt.Sprintf("complete -o nospace -C %s %s", bin, cmd)
}

// Utility functions copied from posener/complete/cmd/install/utils.go

func lineInFile(name string, lookFor string) bool {
	f, err := os.Open(name)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 1024)
	var line []byte
	for {
		n, err := f.Read(buf)
		if n > 0 {
			for i := 0; i < n; i++ {
				if buf[i] == '\n' {
					if string(line) == lookFor {
						return true
					}
					line = line[:0]
				} else {
					line = append(line, buf[i])
				}
			}
		}
		if err != nil {
			break
		}
	}
	if len(line) > 0 && string(line) == lookFor {
		return true
	}
	return false
}

func appendToFile(name string, content string) error {
	f, err := os.OpenFile(name, os.O_RDWR|os.O_APPEND, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(fmt.Sprintf("\n%s\n", content))
	return err
}

func removeFromFile(name string, content string) error {
	// Read the file
	data, err := os.ReadFile(name)
	if err != nil {
		return err
	}

	// Remove the content
	lines := string(data)
	var result string
	var currentLine string
	for i := 0; i < len(lines); i++ {
		if lines[i] == '\n' {
			if currentLine != content {
				result += currentLine + "\n"
			}
			currentLine = ""
		} else {
			currentLine += string(lines[i])
		}
	}
	if currentLine != "" && currentLine != content {
		result += currentLine + "\n"
	}

	// Write back
	return os.WriteFile(name, []byte(result), 0644)
}

// isZsh returns true if the current shell is zsh.
func isZsh() bool {
	shell := os.Getenv("SHELL")
	return shell != "" && filepath.Base(shell) == "zsh"
}

// getZshrcPath returns the path to the zsh rc file, respecting ZDOTDIR.
func getZshrcPath() string {
	if zdotdir := os.Getenv("ZDOTDIR"); zdotdir != "" {
		path := filepath.Join(zdotdir, ".zshrc")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	// Fall back to home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	path := filepath.Join(home, ".zshrc")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

// getBashrcPaths returns the list of bash rc file candidates.
func getBashrcPaths() []string {
	var bashConfFiles []string
	switch runtime.GOOS {
	case "darwin":
		bashConfFiles = []string{".bash_profile"}
	default:
		bashConfFiles = []string{".bashrc", ".bash_profile", ".bash_login", ".profile"}
	}

	var paths []string
	home, err := os.UserHomeDir()
	if err != nil {
		return paths
	}
	for _, rc := range bashConfFiles {
		path := filepath.Join(home, rc)
		if _, err := os.Stat(path); err == nil {
			paths = append(paths, path)
		}
	}
	return paths
}

// getFishConfigDir returns the fish configuration directory.
func getFishConfigDir() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		configHome = filepath.Join(home, ".config")
	}
	configDir := filepath.Join(configHome, "fish")
	if info, err := os.Stat(configDir); err == nil && info.IsDir() {
		return configDir
	}
	return ""
}

// AutocompleteIsInstalled returns true if autocompletion is installed for any shell.
func AutocompleteIsInstalled(cmd string) bool {
	if install.IsInstalled(cmd) {
		return true
	}
	// Check ZDOTDIR
	if zdotdir := os.Getenv("ZDOTDIR"); zdotdir != "" {
		zshrcPath := filepath.Join(zdotdir, ".zshrc")
		if _, err := os.Stat(zshrcPath); err == nil {
			zshInstaller := &zshInstaller{rc: zshrcPath}
			bin, err := getBinaryPath()
			if err == nil && zshInstaller.IsInstalled(cmd, bin) {
				return true
			}
		}
	}
	return false
}
