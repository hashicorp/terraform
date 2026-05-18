// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package tfdiagsappendcheck

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	// Note, assertions are made in comments in the testdata files.
	// See comments in ./testpkg/testpkg.go
	// See docs for analysistest.Run for more info: https://pkg.go.dev/golang.org/x/tools/go/analysis/analysistest#Run
	analysistest.Run(t, testdata, DiagsAppendAnalyzer, "github.com/hashicorp/terraform/tools/tfdiagsappendcheck/testpkg")
}
