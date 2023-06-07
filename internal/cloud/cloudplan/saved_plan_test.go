// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloudplan

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
)

func TestCloud_loadBasic(t *testing.T) {
	bookmark := SavedPlanBookmark{
		RemotePlanFormat: 1,
		RunID:            "run-GXfuHMkbyHccAGUg",
		Hostname:         "app.terraform.io",
	}

	file := "./testdata/plan-bookmark/bookmark.json"
	result, err := LoadSavedPlanBookmark(file)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(bookmark, result, cmp.Comparer(cty.Value.RawEquals)); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}
}

func TestCloud_loadCheckRunID(t *testing.T) {
	// Run ID must never be empty
	file := "./testdata/plan-bookmark/empty_run_id.json"
	_, err := LoadSavedPlanBookmark(file)
	if !errors.Is(err, ErrInvalidRunID) {
		t.Fatalf("expected %s but got %s", ErrInvalidRunID, err)
	}
}

func TestCloud_loadCheckHostname(t *testing.T) {
	// Hostname must never be empty
	file := "./testdata/plan-bookmark/empty_hostname.json"
	_, err := LoadSavedPlanBookmark(file)
	if !errors.Is(err, ErrInvalidHostname) {
		t.Fatalf("expected %s but got %s", ErrInvalidHostname, err)
	}
}

func TestCloud_loadCheckVersionNumberBasic(t *testing.T) {
	// remote_plan_format must be set to 1
	// remote_plan_format and format version number are used interchangeably
	file := "./testdata/plan-bookmark/invalid_version.json"
	_, err := LoadSavedPlanBookmark(file)
	if !errors.Is(err, ErrInvalidRemotePlanFormat) {
		t.Fatalf("expected %s but got %s", ErrInvalidRemotePlanFormat, err)
	}
}

func TestCloud_saveWhenFileExistsBasic(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile, err := os.Create(filepath.Join(tmpDir, "saved-bookmark.json"))
	if err != nil {
		t.Fatal("File could not be created.", err)
	}
	defer tmpFile.Close()

	// verify the created path exists
	// os.Stat() wants path to file
	_, error := os.Stat(tmpFile.Name())
	if error != nil {
		t.Fatal("Path to file does not exist.", error)
	} else {
		b := &SavedPlanBookmark{
			RemotePlanFormat: 1,
			RunID:            "run-GXfuHMkbyHccAGUg",
			Hostname:         "app.terraform.io",
		}
		err := b.Save(tmpFile.Name())
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestCloud_saveWhenFileDoesNotExistBasic(t *testing.T) {
	tmpDir := t.TempDir()
	b := &SavedPlanBookmark{
		RemotePlanFormat: 1,
		RunID:            "run-GXfuHMkbyHccAGUg",
		Hostname:         "app.terraform.io",
	}
	err := b.Save(filepath.Join(tmpDir, "create-new-file.txt"))
	if err != nil {
		t.Fatal(err)
	}
}
