// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type SavedPlanBookmark struct {
	RemotePlanFormat int    `json:"remote_plan_format"`
	RunID            string `json:"run_id"`
	Hostname         string `json:"hostname"`
}

func LoadSavedPlanBookmark(filepath string) (SavedPlanBookmark, error) {
	fmt.Println("Are we only accepting a filepath?")

	bookmark := SavedPlanBookmark{}

	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return bookmark, err
	}

	err = json.Unmarshal([]byte(data), &bookmark)
	return bookmark, err
}

func (s *SavedPlanBookmark) save() error {
	fmt.Println("this verifies we can save what to what?")

	return nil
}
