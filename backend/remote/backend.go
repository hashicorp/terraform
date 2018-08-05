package remote

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"sort"
	"strings"
	"sync"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/svchost"
	"github.com/hashicorp/terraform/svchost/disco"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/version"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
)

const (
	defaultHostname = "app.terraform.io"
	serviceID       = "tfe.v2"
)

// Remote is an implementation of EnhancedBackend that performs all
// operations in a remote backend.
type Remote struct {
	// CLI and Colorize control the CLI output. If CLI is nil then no CLI
	// output will be done. If CLIColor is nil then no coloring will be done.
	CLI      cli.Ui
	CLIColor *colorstring.Colorize

	// ContextOpts are the base context options to set when initializing a
	// new Terraform context. Many of these will be overridden or merged by
	// Operation. See Operation for more details.
	ContextOpts *terraform.ContextOpts

	// client is the remote backend API client
	client *tfe.Client

	// hostname of the remote backend server
	hostname string

	// organization is the organization that contains the target workspaces
	organization string

	// workspace is used to map the default workspace to a remote workspace
	workspace string

	// prefix is used to filter down a set of workspaces that use a single
	// configuration
	prefix string

	// schema defines the configuration for the backend
	schema *schema.Backend

	// services is used for service discovery
	services *disco.Disco

	// opLock locks operations
	opLock sync.Mutex
}

// New creates a new initialized remote backend.
func New(services *disco.Disco) *Remote {
	b := &Remote{
		services: services,
	}

	b.schema = &schema.Backend{
		Schema: map[string]*schema.Schema{
			"hostname": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: schemaDescriptions["hostname"],
				Default:     defaultHostname,
			},

			"organization": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: schemaDescriptions["organization"],
			},

			"token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: schemaDescriptions["token"],
				DefaultFunc: schema.EnvDefaultFunc("TFE_TOKEN", ""),
			},

			"workspaces": &schema.Schema{
				Type:        schema.TypeSet,
				Required:    true,
				Description: schemaDescriptions["workspaces"],
				MinItems:    1,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: schemaDescriptions["name"],
						},

						"prefix": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: schemaDescriptions["prefix"],
						},
					},
				},
			},
		},

		ConfigureFunc: b.configure,
	}

	return b
}

func (b *Remote) configure(ctx context.Context) error {
	d := schema.FromContextBackendConfig(ctx)

	// Get the hostname and organization.
	b.hostname = d.Get("hostname").(string)
	b.organization = d.Get("organization").(string)

	// Get and assert the workspaces configuration block.
	workspace := d.Get("workspaces").(*schema.Set).List()[0].(map[string]interface{})

	// Get the default workspace name and prefix.
	b.workspace = workspace["name"].(string)
	b.prefix = workspace["prefix"].(string)

	// Make sure that we have either a workspace name or a prefix.
	if b.workspace == "" && b.prefix == "" {
		return fmt.Errorf("either workspace 'name' or 'prefix' is required")
	}

	// Make sure that only one of workspace name or a prefix is configured.
	if b.workspace != "" && b.prefix != "" {
		return fmt.Errorf("only one of workspace 'name' or 'prefix' is allowed")
	}

	// Discover the service URL for this host to confirm that it provides
	// a remote backend API and to discover the required base path.
	service, err := b.discover(b.hostname)
	if err != nil {
		return err
	}

	// Retrieve the token for this host as configured in the credentials
	// section of the CLI Config File.
	token, err := b.token(b.hostname)
	if err != nil {
		return err
	}
	if token == "" {
		token = d.Get("token").(string)
	}

	cfg := &tfe.Config{
		Address:  service.String(),
		BasePath: service.Path,
		Token:    token,
	}

	// Create the remote backend API client.
	b.client, err = tfe.NewClient(cfg)
	if err != nil {
		return err
	}

	return nil
}

// discover the remote backend API service URL and token.
func (b *Remote) discover(hostname string) (*url.URL, error) {
	host, err := svchost.ForComparison(hostname)
	if err != nil {
		return nil, err
	}
	service := b.services.DiscoverServiceURL(host, serviceID)
	if service == nil {
		return nil, fmt.Errorf("host %s does not provide a remote backend API", host)
	}
	return service, nil
}

// token returns the token for this host as configured in the credentials
// section of the CLI Config File. If no token was configured, an empty
// string will be returned instead.
func (b *Remote) token(hostname string) (string, error) {
	host, err := svchost.ForComparison(hostname)
	if err != nil {
		return "", err
	}
	creds, err := b.services.CredentialsForHost(host)
	if err != nil {
		log.Printf("[WARN] Failed to get credentials for %s: %s (ignoring)", host, err)
		return "", nil
	}
	if creds != nil {
		return creds.Token(), nil
	}
	return "", nil
}

// Input is called to ask the user for input for completing the configuration.
func (b *Remote) Input(ui terraform.UIInput, c *terraform.ResourceConfig) (*terraform.ResourceConfig, error) {
	return b.schema.Input(ui, c)
}

// Validate is called once at the beginning with the raw configuration and
// can return a list of warnings and/or errors.
func (b *Remote) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	return b.schema.Validate(c)
}

// Configure configures the backend itself with the configuration given.
func (b *Remote) Configure(c *terraform.ResourceConfig) error {
	return b.schema.Configure(c)
}

// State returns the latest state of the given remote workspace. The workspace
// will be created if it doesn't exist.
func (b *Remote) State(workspace string) (state.State, error) {
	if b.workspace == "" && workspace == backend.DefaultStateName {
		return nil, backend.ErrDefaultStateNotSupported
	}
	if b.prefix == "" && workspace != backend.DefaultStateName {
		return nil, backend.ErrNamedStatesNotSupported
	}

	workspaces, err := b.states()
	if err != nil {
		return nil, fmt.Errorf("Error retrieving workspaces: %v", err)
	}

	exists := false
	for _, name := range workspaces {
		if workspace == name {
			exists = true
			break
		}
	}

	// Configure the remote workspace name.
	if workspace == backend.DefaultStateName {
		workspace = b.workspace
	} else if b.prefix != "" && !strings.HasPrefix(workspace, b.prefix) {
		workspace = b.prefix + workspace
	}

	if !exists {
		options := tfe.WorkspaceCreateOptions{
			Name:             tfe.String(workspace),
			TerraformVersion: tfe.String(version.Version),
		}
		_, err = b.client.Workspaces.Create(context.Background(), b.organization, options)
		if err != nil {
			return nil, fmt.Errorf("Error creating workspace %s: %v", workspace, err)
		}
	}

	client := &remoteClient{
		client:       b.client,
		organization: b.organization,
		workspace:    workspace,
	}

	return &remote.State{Client: client}, nil
}

// DeleteState removes the remote workspace if it exists.
func (b *Remote) DeleteState(workspace string) error {
	if b.workspace == "" && workspace == backend.DefaultStateName {
		return backend.ErrDefaultStateNotSupported
	}
	if b.prefix == "" && workspace != backend.DefaultStateName {
		return backend.ErrNamedStatesNotSupported
	}

	// Configure the remote workspace name.
	if workspace == backend.DefaultStateName {
		workspace = b.workspace
	} else if b.prefix != "" && !strings.HasPrefix(workspace, b.prefix) {
		workspace = b.prefix + workspace
	}

	// Check if the configured organization exists.
	_, err := b.client.Organizations.Read(context.Background(), b.organization)
	if err != nil {
		if err == tfe.ErrResourceNotFound {
			return fmt.Errorf("organization %s does not exist", b.organization)
		}
		return err
	}

	client := &remoteClient{
		client:       b.client,
		organization: b.organization,
		workspace:    workspace,
	}

	return client.Delete()
}

// States returns a filtered list of remote workspace names.
func (b *Remote) States() ([]string, error) {
	if b.prefix == "" {
		return nil, backend.ErrNamedStatesNotSupported
	}
	return b.states()
}

func (b *Remote) states() ([]string, error) {
	// Check if the configured organization exists.
	_, err := b.client.Organizations.Read(context.Background(), b.organization)
	if err != nil {
		if err == tfe.ErrResourceNotFound {
			return nil, fmt.Errorf("organization %s does not exist", b.organization)
		}
		return nil, err
	}

	options := tfe.WorkspaceListOptions{}
	ws, err := b.client.Workspaces.List(context.Background(), b.organization, options)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, w := range ws {
		if b.workspace != "" && w.Name == b.workspace {
			names = append(names, backend.DefaultStateName)
			continue
		}
		if b.prefix != "" && strings.HasPrefix(w.Name, b.prefix) {
			names = append(names, strings.TrimPrefix(w.Name, b.prefix))
		}
	}

	// Sort the result so we have consistent output.
	sort.StringSlice(names).Sort()

	return names, nil
}

// Operation implements backend.Enhanced
func (b *Remote) Operation(ctx context.Context, op *backend.Operation) (*backend.RunningOperation, error) {
	// Configure the remote workspace name.
	if op.Workspace == backend.DefaultStateName {
		op.Workspace = b.workspace
	} else if b.prefix != "" && !strings.HasPrefix(op.Workspace, b.prefix) {
		op.Workspace = b.prefix + op.Workspace
	}

	// Determine the function to call for our operation
	var f func(context.Context, context.Context, *backend.Operation, *backend.RunningOperation)
	switch op.Type {
	case backend.OperationTypePlan:
		f = b.opPlan
	default:
		return nil, fmt.Errorf(
			"\n\nThe \"remote\" backend currently only supports the \"plan\" operation.\n"+
				"Please use the remote backend web UI for all other operations:\n"+
				"https://%s/app/%s/%s", b.hostname, b.organization, op.Workspace)
		// return nil, backend.ErrOperationNotSupported
	}

	// Lock
	b.opLock.Lock()

	// Build our running operation
	// the runninCtx is only used to block until the operation returns.
	runningCtx, done := context.WithCancel(context.Background())
	runningOp := &backend.RunningOperation{
		Context: runningCtx,
	}

	// stopCtx wraps the context passed in, and is used to signal a graceful Stop.
	stopCtx, stop := context.WithCancel(ctx)
	runningOp.Stop = stop

	// cancelCtx is used to cancel the operation immediately, usually
	// indicating that the process is exiting.
	cancelCtx, cancel := context.WithCancel(context.Background())
	runningOp.Cancel = cancel

	// Do it
	go func() {
		defer done()
		defer stop()
		defer cancel()

		defer b.opLock.Unlock()
		f(stopCtx, cancelCtx, op, runningOp)
	}()

	// Return
	return runningOp, nil
}

// Colorize returns the Colorize structure that can be used for colorizing
// output. This is gauranteed to always return a non-nil value and so is useful
// as a helper to wrap any potentially colored strings.
func (b *Remote) Colorize() *colorstring.Colorize {
	if b.CLIColor != nil {
		return b.CLIColor
	}

	return &colorstring.Colorize{
		Colors:  colorstring.DefaultColors,
		Disable: true,
	}
}

const generalErr = `
%s: %v

The "remote" backend encountered an unexpected error while communicating
with remote backend. In some cases this could be caused by a network
connection problem, in which case you could retry the command. If the issue
persists please open a support ticket to get help resolving the problem.
`

var schemaDescriptions = map[string]string{
	"hostname":     "The remote backend hostname to connect to (defaults to app.terraform.io).",
	"organization": "The name of the organization containing the targeted workspace(s).",
	"token": "The token used to authenticate with the remote backend. If TFE_TOKEN is set\n" +
		"or credentials for the host are configured in the CLI Config File, then this\n" +
		"this will override any saved value for this.",
	"workspaces": "Workspaces contains arguments used to filter down to a set of workspaces\n" +
		"to work on.",
	"name": "A workspace name used to map the default workspace to a named remote workspace.\n" +
		"When configured only the default workspace can be used. This option conflicts\n" +
		"with \"prefix\"",
	"prefix": "A prefix used to filter workspaces using a single configuration. New workspaces\n" +
		"will automatically be prefixed with this prefix. If omitted only the default\n" +
		"workspace can be used. This option conflicts with \"name\"",
}
