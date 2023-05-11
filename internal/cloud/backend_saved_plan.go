// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

type SavedPlanBookmark struct {
	RemotePlanFormat int    `json:"remote_plan_format"`
	RunID            string `json:"run_id"`
	Hostname         string `json:"hostname"`
}
