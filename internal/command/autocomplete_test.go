// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/cli"
	"github.com/posener/complete"
)

func TestMetaCompletePredictWorkspaceName(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	t.Chdir(td)

	// make sure a vars file doesn't interfere
	err := os.WriteFile(DefaultVarsFilename, nil, 0644)
	if err != nil {
		t.Fatal(err)
	}

	ui := new(cli.MockUi)
	meta := &Meta{Ui: ui}

	predictor := meta.completePredictWorkspaceName()

	got := predictor.Predict(complete.Args{
		Last: "",
	})
	want := []string{"default"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
	}
}
