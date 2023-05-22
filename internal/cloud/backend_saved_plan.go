// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"encoding/json"
	"os"
)

type SavedPlanBookmark struct {
	RemotePlanFormat int    `json:"remote_plan_format"`
	RunID            string `json:"run_id"`
	Hostname         string `json:"hostname"`
}

func LoadSavedPlanBookmark(filepath string) (SavedPlanBookmark, error) {
	bookmark := SavedPlanBookmark{}
	data, err := os.ReadFile(filepath)

	if err != nil {
		return bookmark, err
	}

	err = json.Unmarshal([]byte(data), &bookmark)
	return bookmark, err

}

func (s *SavedPlanBookmark) Save(filepath string) error {
	// this verifies we can save json to a provided file path
	// json.Marshal, then os.WriteFile

	return nil
}
