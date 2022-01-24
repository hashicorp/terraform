// Package localrun deals with the concerns of taking Terraform workflow
// actions only on the local system, without any help from a remote service.
//
// The intent is that local runs follow a similar workflow as remote runs, but
// will achieve it using approaches that make more sense for a local system,
// such as files on disk instead of remote API objects, and will omit features
// that _rely_ on a remote service, such as centralized management of input
// variables, credentials, etc.
package localrun

import (
	"github.com/hashicorp/terraform/internal/terraform"
)

func nothing() {
	// This is literally just here to see what happens when we import the
	// Terraform package into this nested module.
	var thing terraform.Context
}
