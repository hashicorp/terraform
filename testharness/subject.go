package testharness

import (
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/terraform"
	"github.com/zclconf/go-cty/cty"
)

// Subject represents the overall situation being tested. It contains a
// configuration, a state, and a set of top-level variable values that tests
// can then be applied to.
type Subject struct {
	config    *module.Tree
	state     *terraform.State
	variables map[string]cty.Value
}
