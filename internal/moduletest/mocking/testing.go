// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package mocking

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

type InitProviderOverrides func(map[addrs.RootProviderConfig]addrs.Map[addrs.TargetPattern, *configs.Override])
type InitLocalOverrides func(addrs.Map[addrs.TargetPattern, *configs.Override])

func OverridesForTesting(providers InitProviderOverrides, locals InitLocalOverrides) *Overrides {
	overrides := &Overrides{
		providerOverrides: make(map[addrs.RootProviderConfig]addrs.Map[addrs.TargetPattern, *configs.Override]),
		localOverrides:    addrs.MakeMap[addrs.TargetPattern, *configs.Override](),
	}

	if providers != nil {
		providers(overrides.providerOverrides)
	}

	if locals != nil {
		locals(overrides.localOverrides)
	}

	return overrides
}
