// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package testpkg

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func myFunc() tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.Sourceless(tfdiags.Warning, "summary", "detail"))
	return diags
}

func goodHandleReturns() {
	diags := myFunc()
	if len(diags) > 0 {
		fmt.Printf("have diags: %v", diags)
	}
}

func goodIgnoreReturns() {
	_ = myFunc()
	fmt.Print("explicitly not using diags value here")
}

func badNotHandleReturns() {
	myFunc() // want "ignored return value with type tfdiags.Diagnostics"
	fmt.Print("no diags value handling here")
}
