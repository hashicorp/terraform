package terraform

import (
	"sync"

	"github.com/hashicorp/terraform/config"
)

// EvalContext is the interface that is given to eval nodes to execute.
type EvalContext interface {
	// Path is the current module path.
	Path() []string

	// Hook is used to call hook methods. The callback is called for each
	// hook and should return the hook action to take and the error.
	Hook(func(Hook) (HookAction, error)) error

	// Input is the UIInput object for interacting with the UI.
	Input() UIInput

	// InitProvider initializes the provider with the given name and
	// returns the implementation of the resource provider or an error.
	//
	// It is an error to initialize the same provider more than once.
	InitProvider(string) (ResourceProvider, error)

	// Provider gets the provider instance with the given name (already
	// initialized) or returns nil if the provider isn't initialized.
	Provider(string) ResourceProvider

	// CloseProvider closes provider connections that aren't needed anymore.
	CloseProvider(string) error

	// ConfigureProvider configures the provider with the given
	// configuration. This is a separate context call because this call
	// is used to store the provider configuration for inheritance lookups
	// with ParentProviderConfig().
	ConfigureProvider(string, *ResourceConfig) error
	SetProviderConfig(string, *ResourceConfig) error
	ParentProviderConfig(string) *ResourceConfig

	// ProviderInput and SetProviderInput are used to configure providers
	// from user input.
	ProviderInput(string) map[string]interface{}
	SetProviderInput(string, map[string]interface{})

	// InitProvisioner initializes the provisioner with the given name and
	// returns the implementation of the resource provisioner or an error.
	//
	// It is an error to initialize the same provisioner more than once.
	InitProvisioner(string) (ResourceProvisioner, error)

	// Provisioner gets the provisioner instance with the given name (already
	// initialized) or returns nil if the provisioner isn't initialized.
	Provisioner(string) ResourceProvisioner

	// CloseProvisioner closes provisioner connections that aren't needed
	// anymore.
	CloseProvisioner(string) error

	// Interpolate takes the given raw configuration and completes
	// the interpolations, returning the processed ResourceConfig.
	//
	// The resource argument is optional. If given, it is the resource
	// that is currently being acted upon.
	Interpolate(*config.RawConfig, *Resource) (*ResourceConfig, error)

	// SetVariables sets the variables for the module within
	// this context with the name n. This function call is additive:
	// the second parameter is merged with any previous call.
	SetVariables(string, map[string]string)

	// Diff returns the global diff as well as the lock that should
	// be used to modify that diff.
	Diff() (*Diff, *sync.RWMutex)

	// State returns the global state as well as the lock that should
	// be used to modify that state.
	State() (*State, *sync.RWMutex)
}
