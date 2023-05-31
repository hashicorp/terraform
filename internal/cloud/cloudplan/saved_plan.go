// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0
package cloudplan

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type SavedPlanBookmark struct {
	RemotePlanFormat int    `json:"remote_plan_format"`
	RunID            string `json:"run_id"`
	Hostname         string `json:"hostname"`
}

func LoadSavedPlanBookmark(filepath string) (SavedPlanBookmark, error) {
	bookmark := SavedPlanBookmark{}

	file, err := os.Open(filepath)
	if err != nil {
		fmt.Println("error opening file")
		return bookmark, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("error reading file")
		return bookmark, err
	}

	e := json.Unmarshal([]byte(data), &bookmark)
	if e != nil {
		fmt.Println("could not unmarshal")
		return bookmark, e
	}

	if bookmark.RemotePlanFormat != 1 {
		return bookmark, err
	} else if bookmark.Hostname == "" {
		return bookmark, err
	} else if bookmark.RunID == "" {
		return bookmark, err
	}

	return bookmark, err
}

func (s *SavedPlanBookmark) Save(filepath string) error {
	data, _ := json.Marshal(s)

	err := os.WriteFile(filepath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}
