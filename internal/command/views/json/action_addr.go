// Copyright IBM Corp. 2014, 2026
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
	Action          string                  `json:"action"`
	ImpliedProvider string                  `json:"implied_provider"`
	ActionType      string                  `json:"action_type"`
	ActionName      string                  `json:"action_name"`
	ActionKey       ctyjson.SimpleJSONValue `json:"action_key"`
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
