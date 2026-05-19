// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	defectdetector "github.com/hashicorp/terraform/tools/defect-detector"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(defectdetector.DiagsAppendAnalyzer)
}
