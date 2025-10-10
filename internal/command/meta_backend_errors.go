// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import "fmt"

type errBackendLocalRead struct {
	innerError error
}

func (e *errBackendLocalRead) Error() string {
	return fmt.Sprintf(`Error reading local state: %s

Terraform is trying to read your local state to determine if there is
state to migrate to your newly configured backend. Terraform can't continue
without this check because that would risk losing state. Please resolve the
error above and try again.`, e.innerError)
}
