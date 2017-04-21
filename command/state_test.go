package command

import (
	"path/filepath"
	"regexp"
	"sort"
	"testing"

	"github.com/hashicorp/terraform/state"
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

func TestStateDefaultBackupExtension(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	s, err := (&StateMeta{}).State(&Meta{})
	if err != nil {
		t.Fatal(err)
	}

	backupPath := s.(*state.BackupState).Path
	match := regexp.MustCompile(`terraform\.tfstate\.\d+\.backup$`).MatchString
	if !match(backupPath) {
		t.Fatal("Bad backup path:", backupPath)
	}
}
