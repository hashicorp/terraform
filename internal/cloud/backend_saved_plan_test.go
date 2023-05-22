// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestCloud_loadAndSaveBasic(t *testing.T) {
	bookmark := &SavedPlanBookmark{
		RemotePlanFormat: 1,
		RunID:            "run-GXfuHMkbyHccAGUg",
		Hostname:         "app.terraform.io",
	}

	data, _ := json.Marshal(bookmark)
	err := os.WriteFile("/tmp/test-load-bookmark", data, 0644)
	if err != nil {
		t.Fatal(err)
	}

	readBookmark, err := os.ReadFile("/tmp/test-load-bookmark")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Read Ok!", readBookmark)
}
