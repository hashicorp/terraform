// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package initwd

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// includedExtensions contains file extensions that should be included in module hashing.
// Only files that affect Terraform behavior are included.
var includedExtensions = []string{
	".tf",          // Terraform configuration files
	".tf.json",     // JSON-based Terraform configuration
	".tfvars",      // Terraform variable files
	".tfvars.json", // JSON-based variable files
	".tftest.hcl",  // Terraform test files
	".tftest.json", // JSON-based test files
	".sh",          // Shell scripts (often used in provisioners)
	".py",          // Python scripts (often used in provisioners)
	".ps1",         // PowerShell scripts (often used in provisioners)
}

// includedFiles contains specific filenames that should always be included if present
var includedFiles = []string{
	"README.md", // Documentation is important for module users
	"README.txt",
	"README",
	"LICENSE", // License information
	"LICENSE.md",
	"LICENSE.txt",
	"CHANGELOG.md", // Version history
	"CHANGELOG.txt",
	"CHANGELOG",
}

// shouldIncludePath checks if a given path should be included in hashing
// based on explicit inclusion rules
func shouldIncludePath(relPath string) bool {
	// Normalize path separators
	normalized := filepath.ToSlash(relPath)

	// Skip hidden directories (except .terraform which we'd skip anyway)
	parts := strings.Split(normalized, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") {
			return false
		}
	}

	// Check if it's a file we always include
	baseName := filepath.Base(relPath)
	for _, included := range includedFiles {
		if strings.EqualFold(baseName, included) {
			return true
		}
	}

	// Check file extensions
	ext := strings.ToLower(filepath.Ext(relPath))
	for _, includedExt := range includedExtensions {
		if ext == includedExt {
			return true
		}
	}

	// Don't include anything else
	return false
}

// hashModuleContent computes a content-based hash of a module directory,
// including only files that affect Terraform's behavior. This ensures
// consistent hashes across different download methods while maintaining
// security by only validating files that actually matter.
func hashModuleContent(dir string) (string, error) {
	// Collect all files that should be included in the hash
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path from module root
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		// Check if this file should be included
		if shouldIncludePath(relPath) {
			// Convert to forward slashes for consistent hashing
			files = append(files, filepath.ToSlash(relPath))
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to walk directory: %w", err)
	}

	// If no relevant files found, that's an error
	if len(files) == 0 {
		return "", fmt.Errorf("no Terraform configuration files found in module directory")
	}

	// Sort files for deterministic output
	slices.Sort(files)

	// Hash each file's contents
	h := sha256.New()
	for _, file := range files {
		if strings.Contains(file, "\n") {
			return "", fmt.Errorf("filenames with newlines are not supported")
		}

		fullPath := filepath.Join(dir, filepath.FromSlash(file))
		f, err := os.Open(fullPath)
		if err != nil {
			return "", fmt.Errorf("failed to open %s: %w", file, err)
		}

		// Hash the file contents
		hf := sha256.New()
		_, err = io.Copy(hf, f)
		f.Close()
		if err != nil {
			return "", fmt.Errorf("failed to hash %s: %w", file, err)
		}

		// Add to summary: "hash  filename\n" (matching dirhash format)
		fmt.Fprintf(h, "%x  %s\n", hf.Sum(nil), file)
	}

	// Return in the same h1: format as dirhash for compatibility
	return "h1:" + base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}
