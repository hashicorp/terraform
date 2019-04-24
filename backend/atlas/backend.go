package atlas

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
)

const EnvVarToken = "ATLAS_TOKEN"
const EnvVarAddress = "ATLAS_ADDRESS"

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

	// opLock locks operations
	opLock sync.Mutex
}

var _ backend.Backend = (*Backend)(nil)

// New returns a new initialized Atlas backend.
func New() *Backend {
	return &Backend{}
}

func (b *Backend) ConfigSchema() *configschema.Block {
	return &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"name": {
				Type:        cty.String,
				Required:    true,
				Description: "Full name of the environment in Terraform Enterprise, such as 'myorg/myenv'",
			},
			"access_token": {
				Type:        cty.String,
				Optional:    true,
				Description: "Access token to use to access Terraform Enterprise; the ATLAS_TOKEN environment variable is used if this argument is not set",
			},
			"address": {
				Type:        cty.String,
				Optional:    true,
				Description: "Base URL for your Terraform Enterprise installation; the ATLAS_ADDRESS environment variable is used if this argument is not set, finally falling back to a default of 'https://atlas.hashicorp.com/' if neither are set.",
			},
		},
	}
}

func (b *Backend) PrepareConfig(obj cty.Value) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	name := obj.GetAttr("name").AsString()
	if ct := strings.Count(name, "/"); ct != 1 {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Invalid workspace selector",
			`The "name" argument must be an organization name and a workspace name separated by a slash, such as "acme/network-production".`,
			cty.Path{cty.GetAttrStep{Name: "name"}},
		))
	}

	if v := obj.GetAttr("address"); !v.IsNull() {
		addr := v.AsString()
		_, err := url.Parse(addr)
		if err != nil {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid Terraform Enterprise URL",
				fmt.Sprintf(`The "address" argument must be a valid URL: %s.`, err),
				cty.Path{cty.GetAttrStep{Name: "address"}},
			))
		}
	}

	return obj, diags
}

func (b *Backend) Configure(obj cty.Value) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	client := &stateClient{
		// This is optionally set during Atlas Terraform runs.
		RunId: os.Getenv("ATLAS_RUN_ID"),
	}

	name := obj.GetAttr("name").AsString() // assumed valid due to PrepareConfig method
	slashIdx := strings.Index(name, "/")
	client.User = name[:slashIdx]
	client.Name = name[slashIdx+1:]

	if v := obj.GetAttr("access_token"); !v.IsNull() {
		client.AccessToken = v.AsString()
	} else {
		client.AccessToken = os.Getenv(EnvVarToken)
		if client.AccessToken == "" {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Missing Terraform Enterprise access token",
				`The "access_token" argument must be set unless the ATLAS_TOKEN environment variable is set to provide the authentication token for Terraform Enterprise.`,
				cty.Path{cty.GetAttrStep{Name: "access_token"}},
			))
		}
	}

	if v := obj.GetAttr("address"); !v.IsNull() {
		addr := v.AsString()
		addrURL, err := url.Parse(addr)
		if err != nil {
			// We already validated the URL in PrepareConfig, so this shouldn't happen
			panic(err)
		}
		client.Server = addr
		client.ServerURL = addrURL
	} else {
		addr := os.Getenv(EnvVarAddress)
		if addr == "" {
			addr = defaultAtlasServer
		}
		addrURL, err := url.Parse(addr)
		if err != nil {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid Terraform Enterprise URL",
				fmt.Sprintf(`The ATLAS_ADDRESS environment variable must contain a valid URL: %s.`, err),
				cty.Path{cty.GetAttrStep{Name: "address"}},
			))
		}
		client.Server = addr
		client.ServerURL = addrURL
	}

	b.stateClient = client

	return diags
}

func (b *Backend) Workspaces() ([]string, error) {
	return nil, backend.ErrWorkspacesNotSupported
}

func (b *Backend) DeleteWorkspace(name string) error {
	return backend.ErrWorkspacesNotSupported
}

func (b *Backend) StateMgr(name string) (state.State, error) {
	if name != backend.DefaultStateName {
		return nil, backend.ErrWorkspacesNotSupported
	}

	return &remote.State{Client: b.stateClient}, nil
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
