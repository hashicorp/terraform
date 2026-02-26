// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"archive/zip"
	"bytes"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/plans/planfile"
)

func TestConfigSnapshotRoundtrip(t *testing.T) {
	fixtureDir := filepath.Join("testdata", "planfile", "test-config")
	loader, err := configload.NewLoader(&configload.Config{
		ModulesDir: filepath.Join(fixtureDir, ".terraform", "modules"),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, snapIn, diags := testLoadWithSnapshot(fixtureDir, loader, nil)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	err = planfile.WriteConfigSnapshot(snapIn, zw)
	if err != nil {
		t.Fatalf("failed to write snapshot: %s", err)
	}
	zw.Close()

	raw := buf.Bytes()
	r := bytes.NewReader(raw)
	zr, err := zip.NewReader(r, int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}

	snapOut, err := planfile.ReadConfigSnapshot(zr)
	if err != nil {
		t.Fatalf("failed to read snapshot: %s", err)
	}

	if !reflect.DeepEqual(snapIn, snapOut) {
		t.Errorf("result does not match input\nresult: %sinput: %s", spew.Sdump(snapOut), spew.Sdump(snapIn))
	}
}
