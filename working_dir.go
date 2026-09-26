// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package main

import "github.com/hashicorp/terraform/internal/command/workdir"

// WorkingDir configures the working directory environment for Terraform.
//
// originalDir is the absolute path where the process started (before any -chdir).
// overrideDataDir is an optional path to override the default data directory
// (typically .terraform), usually set via TF_DATA_DIR.
func WorkingDir(originalDir string, overrideDataDir string) *workdir.Dir {
	ret := workdir.NewDir(".") // caller should already have used os.Chdir in "-chdir=..." mode
	ret.OverrideOriginalWorkingDir(originalDir)
	if overrideDataDir != "" {
		ret.OverrideDataDir(overrideDataDir)
	}
	return ret
}
