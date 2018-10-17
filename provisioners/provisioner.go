package provisioners

import (
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// Interface is the set of methods required for a resource provisioner plugin.
type Interface interface {
	// GetSchema returns the schema for the provisioner configuration.
	GetSchema() GetSchemaResponse

	// ValidateProvisionerConfig allows the provisioner to validate the
	// configuration values.
	ValidateProvisionerConfig(ValidateProvisionerConfigRequest) ValidateProvisionerConfigResponse

	// ProvisionResource runs the provisioner with provided configuration.
	// ProvisionResource blocks until the execution is complete.
	// If the returned diagnostics contain any errors, the resource will be
	// left in a tainted state.
	ProvisionResource(ProvisionResourceRequest) ProvisionResourceResponse

	// Stop is called to interrupt the provisioner.
	//
	// Stop should not block waiting for in-flight actions to complete. It
	// should take any action it wants and return immediately acknowledging it
	// has received the stop request. Terraform will not make any further API
	// calls to the provisioner after Stop is called.
	//
	// The error returned, if non-nil, is assumed to mean that signaling the
	// stop somehow failed and that the user should expect potentially waiting
	// a longer period of time.
	Stop() error

	// Close shuts down the plugin process if applicable.
	Close() error
}

type GetSchemaResponse struct {
	// Provisioner contains the schema for this provisioner.
	Provisioner *configschema.Block

	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics
}

// UIOutput provides the Output method for resource provisioner
// plugins to write any output to the UI.
//
// Provisioners may call the Output method multiple times while Apply is in
// progress. It is invalid to call Output after Apply returns.
type UIOutput interface {
	Output(string)
}

type ValidateProvisionerConfigRequest struct {
	// Config is the complete configuration to be used for the provisioner.
	Config cty.Value
}

type ValidateProvisionerConfigResponse struct {
	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics
}

type ProvisionResourceRequest struct {
	// Config is the complete provisioner configuration.
	Config cty.Value

	// Connection contains any information required to access the resource
	// instance.
	Connection cty.Value

	// UIOutput is used to return output during the Apply operation.
	UIOutput UIOutput
}

type ProvisionResourceResponse struct {
	// Diagnostics contains any warnings or errors from the method call.
	Diagnostics tfdiags.Diagnostics
}
