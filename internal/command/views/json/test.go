// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package json

import (
	"strings"

	"github.com/hashicorp/terraform/internal/moduletest"
)

type TestSuiteAbstract map[string][]string

type TestStatus string

type TestProgress string

type TestFileStatus struct {
	Path     string       `json:"path"`
	Progress TestProgress `json:"progress"`
	Status   TestStatus   `json:"status,omitempty"`
}

type TestRunStatus struct {
	Path     string       `json:"path"`
	Run      string       `json:"run"`
	Progress TestProgress `json:"progress"`
	Elapsed  *int64       `json:"elapsed,omitempty"`
	Status   TestStatus   `json:"status,omitempty"`
}

type TestSuiteSummary struct {
	Status  TestStatus `json:"status"`
	Passed  int        `json:"passed"`
	Failed  int        `json:"failed"`
	Errored int        `json:"errored"`
	Skipped int        `json:"skipped"`
}

type TestFileCleanup struct {
	FailedResources []TestFailedResource `json:"failed_resources"`
}

type TestFailedResource struct {
	Instance   string `json:"instance"`
	DeposedKey string `json:"deposed_key,omitempty"`
}

type TestFatalInterrupt struct {
	State   []TestFailedResource            `json:"state,omitempty"`
	States  map[string][]TestFailedResource `json:"states,omitempty"`
	Planned []string                        `json:"planned,omitempty"`
}

type TestStatusUpdate struct {
	Status   string  `json:"status"`
	Duration float64 `json:"duration"`
}

func ToTestStatus(status moduletest.Status) TestStatus {
	return TestStatus(strings.ToLower(status.String()))
}

func ToTestProgress(progress moduletest.Progress) TestProgress {
	return TestProgress(strings.ToLower(progress.String()))
}
