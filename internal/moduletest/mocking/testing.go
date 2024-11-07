// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mocking

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

type InitProviderOverrides func(map[string]addrs.Map[addrs.Targetable, *configs.Override])
type InitLocalOverrides func(addrs.Map[addrs.Targetable, *configs.Override])

func OverridesForTesting(providers InitProviderOverrides, locals InitLocalOverrides) *Overrides {
	overrides := &Overrides{
		providerOverrides: make(map[string]addrs.Map[addrs.Targetable, *configs.Override]),
		localOverrides:    addrs.MakeMap[addrs.Targetable, *configs.Override](),
	}

	if providers != nil {
		providers(overrides.providerOverrides)
	}

	if locals != nil {
		locals(overrides.localOverrides)
	}

	return overrides
}
