// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloudplan

import (
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

func TestCloud_loadIsSavedPlanBasic(t *testing.T) {
	// JSON must include hostname, run ID, version
}

func TestCloud_loadErrorWhenJSONEmptyBasic(t *testing.T) {
	// loaded file should never be empty
}

func TestCloud_loadErrorWhenJSONValsEmptyBasic(t *testing.T) {
	// JSON values should never be empty
}

func TestCloud_loadCheckVersionNumberBasic(t *testing.T) {
	// remote_plan_format must be set to 1
	// remote_plan_format and format version number are used interchangeably
	validVersion := 1
	file := "./testdata/plan-bookmark/wrong_version.json"
	result, err := LoadSavedPlanBookmark(file)
	if err != nil {
		t.Fatal(err)
	}
	if result.RemotePlanFormat != validVersion {
		t.Fatal("invalid format version: ", result.RemotePlanFormat)
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
