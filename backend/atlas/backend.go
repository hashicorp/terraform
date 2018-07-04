package atlas

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
)

// Backend is an implementation of EnhancedBackend that performs all operations
// in Atlas. State must currently also be stored in Atlas, although it is worth
// investigating in the future if state storage can be external as well.
type Backend struct {
	// CLI and Colorize control the CLI output. If CLI is nil then no CLI
	// output will be done. If CLIColor is nil then no coloring will be done.
	CLI      cli.Ui
	CLIColor *colorstring.Colorize

	// ContextOpts are the base context options to set when initializing a
	// Terraform context. Many of these will be overridden or merged by
	// Operation. See Operation for more details.
	ContextOpts *terraform.ContextOpts

	//---------------------------------------------------------------
	// Internal fields, do not set
	//---------------------------------------------------------------
	// stateClient is the legacy state client, setup in Configure
	stateClient *stateClient

	// schema is the schema for configuration, set by init
	schema *schema.Backend

	// opLock locks operations
	opLock sync.Mutex
}

// New returns a new initialized Atlas backend.
func New() *Backend {
	b := &Backend{}
	b.schema = &schema.Backend{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: schemaDescriptions["name"],
			},

			"access_token": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: schemaDescriptions["access_token"],
				DefaultFunc: schema.EnvDefaultFunc("ATLAS_TOKEN", nil),
			},

			"address": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: schemaDescriptions["address"],
				DefaultFunc: schema.EnvDefaultFunc("ATLAS_ADDRESS", defaultAtlasServer),
			},
		},

		ConfigureFunc: b.configure,
	}

	return b
}

func (b *Backend) configure(ctx context.Context) error {
	d := schema.FromContextBackendConfig(ctx)

	// Parse the address
	addr := d.Get("address").(string)
	addrUrl, err := url.Parse(addr)
	if err != nil {
		return fmt.Errorf("Error parsing 'address': %s", err)
	}

	// Parse the org/env
	name := d.Get("name").(string)
	parts := strings.Split(name, "/")
	if len(parts) != 2 {
		return fmt.Errorf("malformed name '%s', expected format '<org>/<name>'", name)
	}
	org := parts[0]
	env := parts[1]

	// Setup the client
	b.stateClient = &stateClient{
		Server:      addr,
		ServerURL:   addrUrl,
		AccessToken: d.Get("access_token").(string),
		User:        org,
		Name:        env,

		// This is optionally set during Atlas Terraform runs.
		RunId: os.Getenv("ATLAS_RUN_ID"),
	}

	return nil
}

func (b *Backend) Input(ui terraform.UIInput, c *terraform.ResourceConfig) (*terraform.ResourceConfig, error) {
	return b.schema.Input(ui, c)
}

func (b *Backend) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	return b.schema.Validate(c)
}

func (b *Backend) Configure(c *terraform.ResourceConfig) error {
	return b.schema.Configure(c)
}

func (b *Backend) State(name string) (state.State, error) {
	if name != backend.DefaultStateName {
		return nil, backend.ErrNamedStatesNotSupported
	}
	return &remote.State{Client: b.stateClient}, nil
}

func (b *Backend) DeleteState(name string) error {
	return backend.ErrNamedStatesNotSupported
}

func (b *Backend) States() ([]string, error) {
	return nil, backend.ErrNamedStatesNotSupported
}

// Colorize returns the Colorize structure that can be used for colorizing
// output. This is gauranteed to always return a non-nil value and so is useful
// as a helper to wrap any potentially colored strings.
func (b *Backend) Colorize() *colorstring.Colorize {
	if b.CLIColor != nil {
		return b.CLIColor
	}

	return &colorstring.Colorize{
		Colors:  colorstring.DefaultColors,
		Disable: true,
	}
}

var schemaDescriptions = map[string]string{
	"name": "Full name of the environment in Atlas, such as 'hashicorp/myenv'",
	"access_token": "Access token to use to access Atlas. If ATLAS_TOKEN is set then\n" +
		"this will override any saved value for this.",
	"address": "Address to your Atlas installation. This defaults to the publicly\n" +
		"hosted version at 'https://atlas.hashicorp.com/'. This address\n" +
		"should contain the full HTTP scheme to use.",
}
