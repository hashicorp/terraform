// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package differ

import (
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/structured"
)

// asDiff is a helper function to abstract away some simple and common
// functionality when converting a renderer into a concrete diff.
func asDiff(change structured.Change, renderer computed.DiffRenderer) computed.Diff {
	return computed.NewDiff(renderer, change.CalculateAction(), change.ReplacePaths.Matches())
}
