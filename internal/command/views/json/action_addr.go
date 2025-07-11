// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package json

import (
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/addrs"
)

type ActionAddr struct {
	Addr            string                  `json:"addr"`
	Module          string                  `json:"module"`
	Action          string                  `json:"resource"`
	ImpliedProvider string                  `json:"implied_provider"`
	ActionType      string                  `json:"resource_type"`
	ActionName      string                  `json:"resource_name"`
	ActionKey       ctyjson.SimpleJSONValue `json:"resource_key"`
}

func newActionAddr(addr addrs.AbsActionInstance) ActionAddr {
	actionKey := ctyjson.SimpleJSONValue{Value: cty.NilVal}
	if addr.Action.Key != nil {
		actionKey.Value = addr.Action.Key.Value()
	}
	return ActionAddr{
		Addr:            addr.String(),
		Module:          addr.Module.String(),
		Action:          addr.Action.String(),
		ImpliedProvider: addr.Action.Action.ImpliedProvider(),
		ActionType:      addr.Action.Action.Type,
		ActionName:      addr.Action.Action.Name,
		ActionKey:       actionKey,
	}
}
