// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package migrate

import (
	"bytes"
	"os"
	"path/filepath"
)

// FileResult holds the before/after content for a single file.
type FileResult struct {
	Filename string
	Before   []byte
	After    []byte
}

// SubMigrationResult holds the outcome of applying one sub-migration.
type SubMigrationResult struct {
	SubMigration SubMigration
	Files        []FileResult // only files that actually changed
}

// Apply runs all sub-migrations against .tf files in dir (recursively).
// It does NOT write files — returns results for the caller to inspect/write.
// Sub-migrations chain: each sees the output of the previous one.
func Apply(dir string, m Migration) ([]SubMigrationResult, error) {
	// Collect .tf files recursively
	var tfFiles []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".tf" {
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			tfFiles = append(tfFiles, rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(tfFiles) == 0 {
		return nil, nil
	}

	// Read initial file contents
	current := make(map[string][]byte, len(tfFiles))
	for _, name := range tfFiles {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		current[name] = data
	}

	var results []SubMigrationResult

	for _, sub := range m.SubMigrations {
		var files []FileResult

		for _, name := range tfFiles {
			before := current[name]
			after, err := sub.Apply(name, before)
			if err != nil {
				return nil, err
			}

			if !bytes.Equal(before, after) {
				files = append(files, FileResult{
					Filename: name,
					Before:   before,
					After:    after,
				})
			}

			// Chain: update current state for next sub-migration
			current[name] = after
		}

		if len(files) > 0 {
			results = append(results, SubMigrationResult{
				SubMigration: sub,
				Files:        files,
			})
		}
	}

	return results, nil
}

// WriteResults writes all changed files to disk using the final state
// from the results. For each file that appears in multiple sub-migration
// results, only the last (final) state is written.
func WriteResults(dir string, results []SubMigrationResult) error {
	// Collect the final state of each file across all sub-migration results.
	final := make(map[string][]byte)
	for _, r := range results {
		for _, f := range r.Files {
			final[f.Filename] = f.After
		}
	}

	for name, data := range final {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			return err
		}
	}

	return nil
}
