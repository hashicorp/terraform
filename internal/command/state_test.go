// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/states/statemgr"
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

func TestStateCommand_docs(t *testing.T) {
	c := &StateCommand{}
	help := c.Help()

	if got, want := help, "Usage: terraform [global options] state <subcommand>"; !strings.Contains(got, want) {
		t.Fatalf("unexpected help text\nwant: %s\nfull output:\n%s", want, got)
	}

	if strings.Contains(help, "specifically tailored to work") || strings.Contains(help, "grep, awk") {
		t.Fatalf("help text still claims default state command output is for machines:\n%s", help)
	}

	if got, want := c.Synopsis(), "Advanced state management"; got != want {
		t.Fatalf("unexpected synopsis\nwant: %s\ngot: %s", want, got)
	}
}

func TestStateDefaultBackupExtension(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	view := arguments.ViewHuman
	s, err := (&StateMeta{}).State(view)
	if err != nil {
		t.Fatal(err)
	}

	backupPath := s.(*statemgr.Filesystem).BackupPath()
	match := regexp.MustCompile(`terraform\.tfstate\.\d+\.backup$`).MatchString
	if !match(backupPath) {
		t.Fatal("Bad backup path:", backupPath)
	}
}
