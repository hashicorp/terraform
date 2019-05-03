package remote

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/svchost"
	"github.com/hashicorp/terraform/svchost/disco"
	"github.com/hashicorp/terraform/terraform"
	tfversion "github.com/hashicorp/terraform/version"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"

	backendLocal "github.com/hashicorp/terraform/backend/local"
)

const (
	defaultHostname    = "app.terraform.io"
	defaultModuleDepth = -1
	defaultParallelism = 10
	tfeServiceID       = "tfe.v2.1"
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

	// lastRetry is set to the last time a request was retried
	lastRetry time.Time

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

	// local, if non-nil, will be used for all enhanced behavior. This
	// allows local behavior with the remote backend functioning as remote
	// state storage backend.
	local backend.Enhanced

	// forceLocal, if true, will force the use of the local backend.
	forceLocal bool

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

	// The backend schema for the workspaces attribute changed in terraform
	// 0.12, and causes a panic when encountered by 0.11. While we don't support
	// downgrading, we do want to protect users from that panic.
	wsConfig := d.Get("workspaces").(*schema.Set).List()
	if len(wsConfig) == 0 {
		return fmt.Errorf("the cached backend configuration is not valid for this version of terraform")
	}

	// Get and assert the workspaces configuration block.
	workspace := wsConfig[0].(map[string]interface{})

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
	// a remote backend API and to get the version constraints.
	service, constraints, err := b.discover()

	// First check any contraints we might have received.<Paste>
	if constraints != nil {
		if err := b.checkConstraints(constraints); err != nil {
			return err
		}
	}

	// When we don't have any constraints errors, also check for discovery
	// errors before we continue.
	if err != nil {
		return err
	}

	// Retrieve the token for this host as configured in the credentials
	// section of the CLI Config File.
	token, err := b.token()
	if err != nil {
		return err
	}

	// Get the token from the config if no token was configured for this
	// host in credentials section of the CLI Config File.
	if token == "" {
		token = d.Get("token").(string)
	}

	// Return an error if we still don't have a token at this point.
	if token == "" {
		return fmt.Errorf("required token could not be found")
	}

	cfg := &tfe.Config{
		Address:      service.String(),
		BasePath:     service.Path,
		Token:        token,
		Headers:      make(http.Header),
		RetryLogHook: b.retryLogHook,
	}

	// Set the version header to the current version.
	cfg.Headers.Set(tfversion.Header, tfversion.Version)

	// Create the remote backend API client.
	b.client, err = tfe.NewClient(cfg)
	if err != nil {
		return err
	}

	// Check if the organization exists by reading its entitlements.
	entitlements, err := b.client.Organizations.Entitlements(context.Background(), b.organization)
	if err != nil {
		if err == tfe.ErrResourceNotFound {
			return fmt.Errorf("organization %s does not exist", b.organization)
		}
		return fmt.Errorf("failed to read organization entitlements: %v", err)
	}

	// Configure a local backend for when we need to run operations locally.
	b.local = backendLocal.NewWithBackend(b)
	b.forceLocal = !entitlements.Operations || os.Getenv("TF_FORCE_LOCAL_BACKEND") != ""

	// Enable retries for server errors as the backend is now fully configured.
	b.client.RetryServerErrors(true)

	return nil
}

// discover the remote backend API service URL and version constraints.
func (b *Remote) discover() (*url.URL, *disco.Constraints, error) {
	hostname, err := svchost.ForComparison(b.hostname)
	if err != nil {
		return nil, nil, err
	}

	host, err := b.services.Discover(hostname)
	if err != nil {
		return nil, nil, err
	}

	service, err := host.ServiceURL(tfeServiceID)
	// Return the error, unless its a disco.ErrVersionNotSupported error.
	if _, ok := err.(*disco.ErrVersionNotSupported); !ok && err != nil {
		return nil, nil, err
	}

	// We purposefully ignore the error and return the previous error, as
	// checking for version constraints is considered optional.
	constraints, _ := host.VersionConstraints(tfeServiceID, "terraform")

	return service, constraints, err
}

// checkConstraints checks service version constrains against our own
// version and returns rich and informational diagnostics in case any
// incompatibilities are detected.
func (b *Remote) checkConstraints(c *disco.Constraints) error {
	if c == nil || c.Minimum == "" || c.Maximum == "" {
		return nil
	}

	// Generate a parsable constraints string.
	excluding := ""
	if len(c.Excluding) > 0 {
		excluding = fmt.Sprintf(", != %s", strings.Join(c.Excluding, ", != "))
	}
	constStr := fmt.Sprintf(">= %s%s, <= %s", c.Minimum, excluding, c.Maximum)

	// Create the constraints to check against.
	constraints, err := version.NewConstraint(constStr)
	if err != nil {
		return checkConstraintsWarning(err)
	}

	// Create the version to check.
	v, err := version.NewVersion(tfversion.Version)
	if err != nil {
		return checkConstraintsWarning(err)
	}

	// Return if we satisfy all constraints.
	if constraints.Check(v) {
		return nil
	}

	// Find out what action (upgrade/downgrade) we should advice.
	minimum, err := version.NewVersion(c.Minimum)
	if err != nil {
		return checkConstraintsWarning(err)
	}

	maximum, err := version.NewVersion(c.Maximum)
	if err != nil {
		return checkConstraintsWarning(err)
	}

	var excludes []*version.Version
	for _, exclude := range c.Excluding {
		v, err := version.NewVersion(exclude)
		if err != nil {
			return checkConstraintsWarning(err)
		}
		excludes = append(excludes, v)
	}

	// Sort all the excludes.
	sort.Sort(version.Collection(excludes))

	var action, toVersion string
	switch {
	case minimum.GreaterThan(v):
		action = "upgrade"
		toVersion = ">= " + minimum.String()
	case maximum.LessThan(v):
		action = "downgrade"
		toVersion = "<= " + maximum.String()
	case len(excludes) > 0:
		// Get the latest excluded version.
		action = "upgrade"
		toVersion = "> " + excludes[len(excludes)-1].String()
	}

	switch {
	case len(excludes) == 1:
		excluding = fmt.Sprintf(", excluding version %s", excludes[0].String())
	case len(excludes) > 1:
		var vs []string
		for _, v := range excludes {
			vs = append(vs, v.String())
		}
		excluding = fmt.Sprintf(", excluding versions %s", strings.Join(vs, ", "))
	default:
		excluding = ""
	}

	summary := fmt.Sprintf("Incompatible Terraform version v%s", v.String())
	details := fmt.Sprintf(
		"The configured Terraform Enterprise backend is compatible with Terraform\n"+
			"versions >= %s, <= %s%s.", c.Minimum, c.Maximum, excluding,
	)

	if action != "" && toVersion != "" {
		summary = fmt.Sprintf("Please %s Terraform to %s", action, toVersion)
	}

	// Return the customized and informational error message.
	return fmt.Errorf("%s\n\n%s", summary, details)
}

// token returns the token for this host as configured in the credentials
// section of the CLI Config File. If no token was configured, an empty
// string will be returned instead.
func (b *Remote) token() (string, error) {
	hostname, err := svchost.ForComparison(b.hostname)
	if err != nil {
		return "", err
	}
	creds, err := b.services.CredentialsForHost(hostname)
	if err != nil {
		log.Printf("[WARN] Failed to get credentials for %s: %s (ignoring)", b.hostname, err)
		return "", nil
	}
	if creds != nil {
		return creds.Token(), nil
	}
	return "", nil
}

// retryLogHook is invoked each time a request is retried allowing the
// backend to log any connection issues to prevent data loss.
func (b *Remote) retryLogHook(attemptNum int, resp *http.Response) {
	if b.CLI != nil {
		// Ignore the first retry to make sure any delayed output will
		// be written to the console before we start logging retries.
		//
		// The retry logic in the TFE client will retry both rate limited
		// requests and server errors, but in the remote backend we only
		// care about server errors so we ignore rate limit (429) errors.
		if attemptNum == 0 || resp.StatusCode == 429 {
			// Reset the last retry time.
			b.lastRetry = time.Now()
			return
		}

		if attemptNum == 1 {
			b.CLI.Output(b.Colorize().Color(strings.TrimSpace(initialRetryError)))
		} else {
			b.CLI.Output(b.Colorize().Color(strings.TrimSpace(
				fmt.Sprintf(repeatedRetryError, time.Since(b.lastRetry).Round(time.Second)))))
		}
	}
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
func (b *Remote) State(name string) (state.State, error) {
	if b.workspace == "" && name == backend.DefaultStateName {
		return nil, backend.ErrDefaultStateNotSupported
	}
	if b.prefix == "" && name != backend.DefaultStateName {
		return nil, backend.ErrNamedStatesNotSupported
	}

	// Configure the remote workspace name.
	switch {
	case name == backend.DefaultStateName:
		name = b.workspace
	case b.prefix != "" && !strings.HasPrefix(name, b.prefix):
		name = b.prefix + name
	}

	workspace, err := b.client.Workspaces.Read(context.Background(), b.organization, name)
	if err != nil && err != tfe.ErrResourceNotFound {
		return nil, fmt.Errorf("Failed to retrieve workspace %s: %v", name, err)
	}

	if err == tfe.ErrResourceNotFound {
		options := tfe.WorkspaceCreateOptions{
			Name: tfe.String(name),
		}

		// We only set the Terraform Version for the new workspace if this is
		// a release candidate or a final release.
		if tfversion.Prerelease == "" || strings.HasPrefix(tfversion.Prerelease, "rc") {
			options.TerraformVersion = tfe.String(tfversion.String())
		}

		workspace, err = b.client.Workspaces.Create(context.Background(), b.organization, options)
		if err != nil {
			return nil, fmt.Errorf("Error creating workspace %s: %v", name, err)
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
func (b *Remote) DeleteState(name string) error {
	if b.workspace == "" && name == backend.DefaultStateName {
		return backend.ErrDefaultStateNotSupported
	}
	if b.prefix == "" && name != backend.DefaultStateName {
		return backend.ErrNamedStatesNotSupported
	}

	// Configure the remote workspace name.
	switch {
	case name == backend.DefaultStateName:
		name = b.workspace
	case b.prefix != "" && !strings.HasPrefix(name, b.prefix):
		name = b.prefix + name
	}

	client := &remoteClient{
		client:       b.client,
		organization: b.organization,
		workspace: &tfe.Workspace{
			Name: name,
		},
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
	name := op.Workspace
	switch {
	case op.Workspace == backend.DefaultStateName:
		name = b.workspace
	case b.prefix != "" && !strings.HasPrefix(op.Workspace, b.prefix):
		name = b.prefix + op.Workspace
	}

	// Retrieve the workspace for this operation.
	w, err := b.client.Workspaces.Read(ctx, b.organization, name)
	if err != nil {
		switch err {
		case context.Canceled:
			return nil, err
		case tfe.ErrResourceNotFound:
			return nil, fmt.Errorf(
				"workspace %s not found\n\n"+
					"The configured \"remote\" backend returns '404 Not Found' errors for resources\n"+
					"that do not exist, as well as for resources that a user doesn't have access\n"+
					"to. If the resource does exists, please check the rights for the used token.",
				name,
			)
		default:
			return nil, fmt.Errorf(
				"%s\n\n"+
					"The configured \"remote\" backend encountered an unexpected error. Sometimes\n"+
					"this is caused by network connection problems, in which case you could retry\n"+
					"the command. If the issue persists please open a support ticket to get help\n"+
					"resolving the problem.",
				err,
			)
		}
	}

	// Check if we need to use the local backend to run the operation.
	if b.forceLocal || !w.Operations {
		return b.local.Operation(ctx, op)
	}

	// Set the remote workspace name.
	op.Workspace = w.Name

	// Determine the function to call for our operation
	var f func(context.Context, context.Context, *backend.Operation, *tfe.Workspace) (*tfe.Run, error)
	switch op.Type {
	case backend.OperationTypePlan:
		f = b.opPlan
	case backend.OperationTypeApply:
		f = b.opApply
	default:
		return nil, fmt.Errorf(
			"\n\nThe \"remote\" backend does not support the %q operation.", op.Type)
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

		r, opErr := f(stopCtx, cancelCtx, op, w)
		if opErr != nil && opErr != context.Canceled {
			runningOp.Err = opErr
			return
		}

		if r == nil && opErr == context.Canceled {
			runningOp.ExitCode = 1
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

			if runningOp.Err == nil && (r.Status == tfe.RunCanceled || r.Status == tfe.RunErrored) {
				runningOp.ExitCode = 1
			}
		}
	}()

	// Return the running operation.
	return runningOp, nil
}

func (b *Remote) cancel(cancelCtx context.Context, op *backend.Operation, r *tfe.Run) error {
	if r.Actions.IsCancelable {
		// Only ask if the remote operation should be canceled
		// if the auto approve flag is not set.
		if !op.AutoApprove {
			v, err := op.UIIn.Input(cancelCtx, &terraform.InputOpts{
				Id:          "cancel",
				Query:       "\nDo you want to cancel the remote operation?",
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
//
// TODO SvH: Rename this back to Colorize as soon as we can pass -no-color.
func (b *Remote) cliColorize() *colorstring.Colorize {
	if b.CLIColor != nil {
		return b.CLIColor
	}

	return &colorstring.Colorize{
		Colors:  colorstring.DefaultColors,
		Disable: true,
	}
}

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

func checkConstraintsWarning(err error) error {
	return fmt.Errorf(
		"Failed to check version constraints: %v\n\n"+
			"Checking version constraints is considered optional, but this is an\n"+
			"unexpected error which should be reported.",
		err,
	)
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

// The newline in this error is to make it look good in the CLI!
const initialRetryError = `
[reset][yellow]There was an error connecting to the remote backend. Please do not exit
Terraform to prevent data loss! Trying to restore the connection...
[reset]
`

const repeatedRetryError = `
[reset][yellow]Still trying to restore the connection... (%s elapsed)[reset]
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
