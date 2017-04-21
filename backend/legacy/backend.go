package legacy

import (
	"fmt"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
)

// Backend is an implementation of backend.Backend for legacy remote state
// clients.
type Backend struct {
	// Type is the type of remote state client to support
	Type string

	// client is set after Configure is called and client is initialized.
	client remote.Client
}

func (b *Backend) Input(
	ui terraform.UIInput, c *terraform.ResourceConfig) (*terraform.ResourceConfig, error) {
	// Return the config as-is, legacy doesn't support input
	return c, nil
}

func (b *Backend) Validate(*terraform.ResourceConfig) ([]string, []error) {
	// No validation was supported for old clients
	return nil, nil
}

func (b *Backend) Configure(c *terraform.ResourceConfig) error {
	// Legacy remote state was only map[string]string config
	var conf map[string]string
	if err := mapstructure.Decode(c.Raw, &conf); err != nil {
		return fmt.Errorf(
			"Failed to decode %q configuration: %s\n\n"+
				"This backend expects all configuration keys and values to be\n"+
				"strings. Please verify your configuration and try again.",
			b.Type, err)
	}

	client, err := remote.NewClient(b.Type, conf)
	if err != nil {
		return fmt.Errorf(
			"Failed to configure remote backend %q: %s",
			b.Type, err)
	}

	// Set our client
	b.client = client
	return nil
}

func (b *Backend) State(name string) (state.State, error) {
	if name != backend.DefaultStateName {
		return nil, backend.ErrNamedStatesNotSupported
	}

	if b.client == nil {
		panic("State called with nil remote state client")
	}

	return &remote.State{Client: b.client}, nil
}

func (b *Backend) States() ([]string, error) {
	return nil, backend.ErrNamedStatesNotSupported
}

func (b *Backend) DeleteState(string) error {
	return backend.ErrNamedStatesNotSupported
}
