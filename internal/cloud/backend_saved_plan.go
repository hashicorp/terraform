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

func (s *SavedPlanBookmark) load(filepath string) (map[string]interface{}, error) {
	fmt.Println("Are we only accepting a filepath?")

	path := filepath
	bookmark := SavedPlanBookmark{}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(data), &bookmark); err != nil {
		panic(err)
	}
	return bookmark, err
}

func (s *SavedPlanBookmark) save() error {
	fmt.Println("this verifies we can save what to what?")

	return nil
}
