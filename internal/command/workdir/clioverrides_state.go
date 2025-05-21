// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workdir

import (
	"encoding/json"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

type ConfigOverrideState struct {
	Overrides json.RawMessage `json:"config"`
}

func (o *ConfigOverrideState) Config(schema *configschema.Block) (cty.Value, error) {
	ty := schema.ImpliedType()
	if o == nil {
		return cty.NullVal(ty), nil
	}
	return ctyjson.Unmarshal(o.Overrides, ty)
}

// OverrideConfig returns an hcl.Body, produced by overriding the supplied config with the available overrides
// It doesn't matter if the config describes a `backend` or `state_storage` block
func (o *ConfigOverrideState) OverrideConfig(schema *configschema.Block, baseConfig hcl.Body) (hcl.Body, error) {
	c, err := o.Config(schema)
	if err != nil {
		return nil, err
	}
	overrideBody := configs.SynthBody("<backend state file>", c.AsValueMap())

	return configs.MergeBodies(baseConfig, overrideBody), nil
}
