package command

import (
	"path/filepath"
	"sort"
	"testing"
)

// testStateBackups returns the list of backups in order of creation
// (oldest first) in the given directory.
func testStateBackups(t *testing.T, dir string) []string {
	// Find all the backups
	list, err := filepath.Glob(filepath.Join(dir, "*"+DefaultBackupExtension))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Sort them which will put them naturally in the right order
	sort.Strings(list)

	return list
}
