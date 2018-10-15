package remote

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

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
	defaultHostname    = "app.terraform.io"
	defaultModuleDepth = -1
	defaultParallelism = 10
	serviceID          = "tfe.v2"
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
		Headers:  make(http.Header),
	}

	// Set the version header to the current version.
	cfg.Headers.Set(version.Header, version.Version)

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
	switch {
	case workspace == backend.DefaultStateName:
		workspace = b.workspace
	case b.prefix != "" && !strings.HasPrefix(workspace, b.prefix):
		workspace = b.prefix + workspace
	}

	if !exists {
		options := tfe.WorkspaceCreateOptions{
			Name: tfe.String(workspace),
		}

		// We only set the Terraform Version for the new workspace if this is
		// a release candidate or a final release.
		if version.Prerelease == "" || strings.HasPrefix(version.Prerelease, "rc") {
			options.TerraformVersion = tfe.String(version.String())
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

		// This is optionally set during Terraform Enterprise runs.
		runID: os.Getenv("TFE_RUN_ID"),
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
	switch {
	case workspace == backend.DefaultStateName:
		workspace = b.workspace
	case b.prefix != "" && !strings.HasPrefix(workspace, b.prefix):
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
	switch {
	case b.workspace != "":
		options.Search = tfe.String(b.workspace)
	case b.prefix != "":
		options.Search = tfe.String(b.prefix)
	}

	// Create a slice to contain all the names.
	var names []string

	for {
		wl, err := b.client.Workspaces.List(context.Background(), b.organization, options)
		if err != nil {
			return nil, err
		}

		for _, w := range wl.Items {
			if b.workspace != "" && w.Name == b.workspace {
				names = append(names, backend.DefaultStateName)
				continue
			}
			if b.prefix != "" && strings.HasPrefix(w.Name, b.prefix) {
				names = append(names, strings.TrimPrefix(w.Name, b.prefix))
			}
		}

		// Exit the loop when we've seen all pages.
		if wl.CurrentPage >= wl.TotalPages {
			break
		}

		// Update the page number to get the next page.
		options.PageNumber = wl.NextPage
	}

	// Sort the result so we have consistent output.
	sort.StringSlice(names).Sort()

	return names, nil
}

// Operation implements backend.Enhanced
func (b *Remote) Operation(ctx context.Context, op *backend.Operation) (*backend.RunningOperation, error) {
	// Configure the remote workspace name.
	switch {
	case op.Workspace == backend.DefaultStateName:
		op.Workspace = b.workspace
	case b.prefix != "" && !strings.HasPrefix(op.Workspace, b.prefix):
		op.Workspace = b.prefix + op.Workspace
	}

	// Determine the function to call for our operation
	var f func(context.Context, context.Context, *backend.Operation) (*tfe.Run, error)
	switch op.Type {
	case backend.OperationTypePlan:
		f = b.opPlan
	case backend.OperationTypeApply:
		f = b.opApply
	default:
		return nil, fmt.Errorf(
			"\n\nThe \"remote\" backend does not support the %q operation.\n"+
				"Please use the remote backend web UI for running this operation:\n"+
				"https://%s/app/%s/%s", op.Type, b.hostname, b.organization, op.Workspace)
	}

	// Lock
	b.opLock.Lock()

	// Build our running operation
	// the runninCtx is only used to block until the operation returns.
	runningCtx, done := context.WithCancel(context.Background())
	runningOp := &backend.RunningOperation{
		Context:   runningCtx,
		PlanEmpty: true,
	}

	// stopCtx wraps the context passed in, and is used to signal a graceful Stop.
	stopCtx, stop := context.WithCancel(ctx)
	runningOp.Stop = stop

	// cancelCtx is used to cancel the operation immediately, usually
	// indicating that the process is exiting.
	cancelCtx, cancel := context.WithCancel(context.Background())
	runningOp.Cancel = cancel

	// Do it.
	go func() {
		defer done()
		defer stop()
		defer cancel()

		defer b.opLock.Unlock()

		r, opErr := f(stopCtx, cancelCtx, op)
		if opErr != nil && opErr != context.Canceled {
			runningOp.Err = opErr
			return
		}

		if r != nil {
			// Retrieve the run to get its current status.
			r, err := b.client.Runs.Read(cancelCtx, r.ID)
			if err != nil {
				runningOp.Err = generalError("error retrieving run", err)
				return
			}

			// Record if there are any changes.
			runningOp.PlanEmpty = !r.HasChanges

			if opErr == context.Canceled {
				runningOp.Err = b.cancel(cancelCtx, op, r)
			}

			if runningOp.Err == nil && r.Status == tfe.RunErrored {
				runningOp.ExitCode = 1
			}
		}
	}()

	// Return the running operation.
	return runningOp, nil
}

// backoff will perform exponential backoff based on the iteration and
// limited by the provided min and max (in milliseconds) durations.
func backoff(min, max float64, iter int) time.Duration {
	backoff := math.Pow(2, float64(iter)/5) * min
	if backoff > max {
		backoff = max
	}
	return time.Duration(backoff) * time.Millisecond
}

func (b *Remote) waitForRun(stopCtx, cancelCtx context.Context, op *backend.Operation, opType string, r *tfe.Run, w *tfe.Workspace) (*tfe.Run, error) {
	started := time.Now()
	updated := started
	for i := 0; ; i++ {
		select {
		case <-stopCtx.Done():
			return r, stopCtx.Err()
		case <-cancelCtx.Done():
			return r, cancelCtx.Err()
		case <-time.After(backoff(1000, 3000, i)):
			// Timer up, show status
		}

		// Retrieve the run to get its current status.
		r, err := b.client.Runs.Read(stopCtx, r.ID)
		if err != nil {
			return r, generalError("error retrieving run", err)
		}

		// Return if the run is no longer pending.
		if r.Status != tfe.RunPending && r.Status != tfe.RunConfirmed {
			if i == 0 && opType == "plan" && b.CLI != nil {
				b.CLI.Output(b.Colorize().Color(fmt.Sprintf("Waiting for the %s to start...\n", opType)))
			}
			if i > 0 && b.CLI != nil {
				// Insert a blank line to separate the ouputs.
				b.CLI.Output("")
			}
			return r, nil
		}

		// Check if 30 seconds have passed since the last update.
		current := time.Now()
		if b.CLI != nil && (i == 0 || current.Sub(updated).Seconds() > 30) {
			updated = current
			position := 0
			elapsed := ""

			// Calculate and set the elapsed time.
			if i > 0 {
				elapsed = fmt.Sprintf(
					" (%s elapsed)", current.Sub(started).Truncate(30*time.Second))
			}

			// Retrieve the workspace used to run this operation in.
			w, err = b.client.Workspaces.Read(stopCtx, b.organization, w.Name)
			if err != nil {
				return nil, generalError("error retrieving workspace", err)
			}

			// If the workspace is locked the run will not be queued and we can
			// update the status without making any expensive calls.
			if w.Locked && w.CurrentRun != nil {
				cr, err := b.client.Runs.Read(stopCtx, w.CurrentRun.ID)
				if err != nil {
					return r, generalError("error retrieving current run", err)
				}
				if cr.Status == tfe.RunPending {
					b.CLI.Output(b.Colorize().Color(
						"Waiting for the manually locked workspace to be unlocked..." + elapsed))
					continue
				}
			}

			// Skip checking the workspace queue when we are the current run.
			if w.CurrentRun == nil || w.CurrentRun.ID != r.ID {
				found := false
				options := tfe.RunListOptions{}
			runlist:
				for {
					rl, err := b.client.Runs.List(stopCtx, w.ID, options)
					if err != nil {
						return r, generalError("error retrieving run list", err)
					}

					// Loop through all runs to calculate the workspace queue position.
					for _, item := range rl.Items {
						if !found {
							if r.ID == item.ID {
								found = true
							}
							continue
						}

						// If the run is in a final state, ignore it and continue.
						switch item.Status {
						case tfe.RunApplied, tfe.RunCanceled, tfe.RunDiscarded, tfe.RunErrored:
							continue
						case tfe.RunPlanned:
							if op.Type == backend.OperationTypePlan {
								continue
							}
						}

						// Increase the workspace queue position.
						position++

						// Stop searching when we reached the current run.
						if w.CurrentRun != nil && w.CurrentRun.ID == item.ID {
							break runlist
						}
					}

					// Exit the loop when we've seen all pages.
					if rl.CurrentPage >= rl.TotalPages {
						break
					}

					// Update the page number to get the next page.
					options.PageNumber = rl.NextPage
				}

				if position > 0 {
					b.CLI.Output(b.Colorize().Color(fmt.Sprintf(
						"Waiting for %d run(s) to finish before being queued...%s",
						position,
						elapsed,
					)))
					continue
				}
			}

			options := tfe.RunQueueOptions{}
		search:
			for {
				rq, err := b.client.Organizations.RunQueue(stopCtx, b.organization, options)
				if err != nil {
					return r, generalError("error retrieving queue", err)
				}

				// Search through all queued items to find our run.
				for _, item := range rq.Items {
					if r.ID == item.ID {
						position = item.PositionInQueue
						break search
					}
				}

				// Exit the loop when we've seen all pages.
				if rq.CurrentPage >= rq.TotalPages {
					break
				}

				// Update the page number to get the next page.
				options.PageNumber = rq.NextPage
			}

			if position > 0 {
				c, err := b.client.Organizations.Capacity(stopCtx, b.organization)
				if err != nil {
					return r, generalError("error retrieving capacity", err)
				}
				b.CLI.Output(b.Colorize().Color(fmt.Sprintf(
					"Waiting for %d queued run(s) to finish before starting...%s",
					position-c.Running,
					elapsed,
				)))
				continue
			}

			b.CLI.Output(b.Colorize().Color(fmt.Sprintf(
				"Waiting for the %s to start...%s", opType, elapsed)))
		}
	}
}

func (b *Remote) cancel(cancelCtx context.Context, op *backend.Operation, r *tfe.Run) error {
	if r.Status == tfe.RunPending && r.Actions.IsCancelable {
		// Only ask if the remote operation should be canceled
		// if the auto approve flag is not set.
		if !op.AutoApprove {
			v, err := op.UIIn.Input(&terraform.InputOpts{
				Id:          "cancel",
				Query:       "\nDo you want to cancel the pending remote operation?",
				Description: "Only 'yes' will be accepted to cancel.",
			})
			if err != nil {
				return generalError("error asking to cancel", err)
			}
			if v != "yes" {
				if b.CLI != nil {
					b.CLI.Output(b.Colorize().Color(strings.TrimSpace(operationNotCanceled)))
				}
				return nil
			}
		} else {
			if b.CLI != nil {
				// Insert a blank line to separate the ouputs.
				b.CLI.Output("")
			}
		}

		// Try to cancel the remote operation.
		err := b.client.Runs.Cancel(cancelCtx, r.ID, tfe.RunCancelOptions{})
		if err != nil {
			return generalError("error cancelling run", err)
		}
		if b.CLI != nil {
			b.CLI.Output(b.Colorize().Color(strings.TrimSpace(operationCanceled)))
		}
	}

	return nil
}

// Colorize returns the Colorize structure that can be used for colorizing
// output. This is guaranteed to always return a non-nil value and so useful
// as a helper to wrap any potentially colored strings.
// func (b *Remote) Colorize() *colorstring.Colorize {
// 	if b.CLIColor != nil {
// 		return b.CLIColor
// 	}

// 	return &colorstring.Colorize{
// 		Colors:  colorstring.DefaultColors,
// 		Disable: true,
// 	}
// }

func generalError(msg string, err error) error {
	if urlErr, ok := err.(*url.Error); ok {
		err = urlErr.Err
	}
	switch err {
	case context.Canceled:
		return err
	case tfe.ErrResourceNotFound:
		return fmt.Errorf(strings.TrimSpace(fmt.Sprintf(notFoundErr, msg, err)))
	default:
		return fmt.Errorf(strings.TrimSpace(fmt.Sprintf(generalErr, msg, err)))
	}
}

const generalErr = `
%s: %v

The configured "remote" backend encountered an unexpected error. Sometimes
this is caused by network connection problems, in which case you could retry
the command. If the issue persists please open a support ticket to get help
resolving the problem.
`

const notFoundErr = `
%s: %v

The configured "remote" backend returns '404 Not Found' errors for resources
that do not exist, as well as for resources that a user doesn't have access
to. When the resource does exists, please check the rights for the used token.
`

const operationCanceled = `
[reset][red]The remote operation was successfully cancelled.[reset]
`

const operationNotCanceled = `
[reset][red]The remote operation was not cancelled.[reset]
`

var schemaDescriptions = map[string]string{
	"hostname":     "The remote backend hostname to connect to (defaults to app.terraform.io).",
	"organization": "The name of the organization containing the targeted workspace(s).",
	"token": "The token used to authenticate with the remote backend. If credentials for the\n" +
		"host are configured in the CLI Config File, then those will be used instead.",
	"workspaces": "Workspaces contains arguments used to filter down to a set of workspaces\n" +
		"to work on.",
	"name": "A workspace name used to map the default workspace to a named remote workspace.\n" +
		"When configured only the default workspace can be used. This option conflicts\n" +
		"with \"prefix\"",
	"prefix": "A prefix used to filter workspaces using a single configuration. New workspaces\n" +
		"will automatically be prefixed with this prefix. If omitted only the default\n" +
		"workspace can be used. This option conflicts with \"name\"",
}
