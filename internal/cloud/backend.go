package cloud

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
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	tfversion "github.com/hashicorp/terraform/version"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"

	backendLocal "github.com/hashicorp/terraform/internal/backend/local"
)

const (
	defaultHostname    = "app.terraform.io"
	defaultParallelism = 10
	tfeServiceID       = "tfe.v2"
	headerSourceKey    = "X-Terraform-Integration"
	headerSourceValue  = "cloud"
)

// Cloud is an implementation of EnhancedBackend in service of the Terraform Cloud/Enterprise
// integration for Terraform CLI. This backend is not intended to be surfaced at the user level and
// is instead an implementation detail of cloud.Cloud.
type Cloud struct {
	// CLI and Colorize control the CLI output. If CLI is nil then no CLI
	// output will be done. If CLIColor is nil then no coloring will be done.
	CLI      cli.Ui
	CLIColor *colorstring.Colorize

	// ContextOpts are the base context options to set when initializing a
	// new Terraform context. Many of these will be overridden or merged by
	// Operation. See Operation for more details.
	ContextOpts *terraform.ContextOpts

	// client is the Terraform Cloud/Enterprise API client.
	client *tfe.Client

	// lastRetry is set to the last time a request was retried.
	lastRetry time.Time

	// hostname of Terraform Cloud or Terraform Enterprise
	hostname string

	// organization is the organization that contains the target workspaces.
	organization string

	// WorkspaceMapping contains strategies for mapping CLI workspaces in the working directory
	// to remote Terraform Cloud workspaces.
	WorkspaceMapping WorkspaceMapping

	// services is used for service discovery
	services *disco.Disco

	// local allows local operations, where Terraform Cloud serves as a state storage backend.
	local backend.Enhanced

	// forceLocal, if true, will force the use of the local backend.
	forceLocal bool

	// opLock locks operations
	opLock sync.Mutex

	// ignoreVersionConflict, if true, will disable the requirement that the
	// local Terraform version matches the remote workspace's configured
	// version. This will also cause VerifyWorkspaceTerraformVersion to return
	// a warning diagnostic instead of an error.
	ignoreVersionConflict bool

	runningInAutomation bool
}

var _ backend.Backend = (*Cloud)(nil)
var _ backend.Enhanced = (*Cloud)(nil)
var _ backend.Local = (*Cloud)(nil)

// New creates a new initialized cloud backend.
func New(services *disco.Disco) *Cloud {
	return &Cloud{
		services: services,
	}
}

// ConfigSchema implements backend.Enhanced.
func (b *Cloud) ConfigSchema() *configschema.Block {
	return &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"hostname": {
				Type:        cty.String,
				Optional:    true,
				Description: schemaDescriptionHostname,
			},
			"organization": {
				Type:        cty.String,
				Required:    true,
				Description: schemaDescriptionOrganization,
			},
			"token": {
				Type:        cty.String,
				Optional:    true,
				Description: schemaDescriptionToken,
			},
		},

		BlockTypes: map[string]*configschema.NestedBlock{
			"workspaces": {
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"name": {
							Type:        cty.String,
							Optional:    true,
							Description: schemaDescriptionName,
						},
						"tags": {
							Type:        cty.Set(cty.String),
							Optional:    true,
							Description: schemaDescriptionTags,
						},
					},
				},
				Nesting: configschema.NestingSingle,
			},
		},
	}
}

// PrepareConfig implements backend.Backend.
func (b *Cloud) PrepareConfig(obj cty.Value) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	if obj.IsNull() {
		return obj, diags
	}

	if val := obj.GetAttr("organization"); val.IsNull() || val.AsString() == "" {
		diags = diags.Append(invalidOrganizationConfigMissingValue)
	}

	WorkspaceMapping := WorkspaceMapping{}
	if workspaces := obj.GetAttr("workspaces"); !workspaces.IsNull() {
		if val := workspaces.GetAttr("name"); !val.IsNull() {
			WorkspaceMapping.Name = val.AsString()
		}
		if val := workspaces.GetAttr("tags"); !val.IsNull() {
			err := gocty.FromCtyValue(val, &WorkspaceMapping.Tags)
			if err != nil {
				log.Panicf("An unxpected error occurred: %s", err)
			}
		}
	}

	switch WorkspaceMapping.Strategy() {
	// Make sure have a workspace mapping strategy present
	case WorkspaceNoneStrategy:
		diags = diags.Append(invalidWorkspaceConfigMissingValues)
	// Make sure that a workspace name is configured.
	case WorkspaceInvalidStrategy:
		diags = diags.Append(invalidWorkspaceConfigMisconfiguration)
	}

	return obj, diags
}

// Configure implements backend.Enhanced.
func (b *Cloud) Configure(obj cty.Value) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	if obj.IsNull() {
		return diags
	}

	diagErr := b.setConfigurationFields(obj)
	if diagErr.HasErrors() {
		return diagErr
	}

	// Discover the service URL to confirm that it provides the Terraform Cloud/Enterprise API
	service, err := b.discover()

	// Check for errors before we continue.
	if err != nil {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			strings.ToUpper(err.Error()[:1])+err.Error()[1:],
			"", // no description is needed here, the error is clear
			cty.Path{cty.GetAttrStep{Name: "hostname"}},
		))
		return diags
	}

	// Retrieve the token for this host as configured in the credentials
	// section of the CLI Config File.
	token, err := b.token()
	if err != nil {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			strings.ToUpper(err.Error()[:1])+err.Error()[1:],
			"", // no description is needed here, the error is clear
			cty.Path{cty.GetAttrStep{Name: "hostname"}},
		))
		return diags
	}

	// Get the token from the config if no token was configured for this
	// host in credentials section of the CLI Config File.
	if token == "" {
		if val := obj.GetAttr("token"); !val.IsNull() {
			token = val.AsString()
		}
	}

	// Return an error if we still don't have a token at this point.
	if token == "" {
		loginCommand := "terraform login"
		if b.hostname != defaultHostname {
			loginCommand = loginCommand + " " + b.hostname
		}
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Required token could not be found",
			fmt.Sprintf(
				"Run the following command to generate a token for %s:\n    %s",
				b.hostname,
				loginCommand,
			),
		))
		return diags
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
	cfg.Headers.Set(headerSourceKey, headerSourceValue)

	// Create the TFC/E API client.
	b.client, err = tfe.NewClient(cfg)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to create the Terraform Cloud/Enterprise client",
			fmt.Sprintf(
				`Encountered an unexpected error while creating the `+
					`Terraform Cloud/Enterprise client: %s.`, err,
			),
		))
		return diags
	}

	// Check if the organization exists by reading its entitlements.
	entitlements, err := b.client.Organizations.Entitlements(context.Background(), b.organization)
	if err != nil {
		if err == tfe.ErrResourceNotFound {
			err = fmt.Errorf("organization %q at host %s not found.\n\n"+
				"Please ensure that the organization and hostname are correct "+
				"and that your API token for %s is valid.",
				b.organization, b.hostname, b.hostname)
		}
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			fmt.Sprintf("Failed to read organization %q at host %s", b.organization, b.hostname),
			fmt.Sprintf("Encountered an unexpected error while reading the "+
				"organization settings: %s", err),
			cty.Path{cty.GetAttrStep{Name: "organization"}},
		))
		return diags
	}

	// Check for the minimum version of Terraform Enterprise required.
	//
	// For API versions prior to 2.3, RemoteAPIVersion will return an empty string,
	// so if there's an error when parsing the RemoteAPIVersion, it's handled as
	// equivalent to an API version < 2.3.
	currentAPIVersion, parseErr := version.NewVersion(b.client.RemoteAPIVersion())
	desiredAPIVersion, _ := version.NewVersion("2.5")

	if parseErr != nil || currentAPIVersion.LessThan(desiredAPIVersion) {
		log.Printf("[TRACE] API version check failed; want: >= %s, got: %s", desiredAPIVersion.Original(), currentAPIVersion)
		if b.runningInAutomation {
			// It should never be possible for this Terraform process to be mistakenly
			// used internally within an unsupported Terraform Enterprise install - but
			// just in case it happens, give an actionable error.
			diags = diags.Append(
				tfdiags.Sourceless(
					tfdiags.Error,
					"Unsupported Terraform Enterprise version",
					cloudIntegrationUsedInUnsupportedTFE,
				),
			)
		} else {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Unsupported Terraform Enterprise version",
				`The 'cloud' option is not supported with this version of Terraform Enterprise.`,
			),
			)
		}
	}

	// Configure a local backend for when we need to run operations locally.
	b.local = backendLocal.NewWithBackend(b)
	b.forceLocal = b.forceLocal || !entitlements.Operations

	// Enable retries for server errors as the backend is now fully configured.
	b.client.RetryServerErrors(true)

	return diags
}

func (b *Cloud) setConfigurationFields(obj cty.Value) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// Get the hostname.
	if val := obj.GetAttr("hostname"); !val.IsNull() && val.AsString() != "" {
		b.hostname = val.AsString()
	} else {
		b.hostname = defaultHostname
	}

	// Get the organization.
	if val := obj.GetAttr("organization"); !val.IsNull() {
		b.organization = val.AsString()
	}

	// Get the workspaces configuration block and retrieve the
	// default workspace name.
	if workspaces := obj.GetAttr("workspaces"); !workspaces.IsNull() {

		// PrepareConfig checks that you cannot set both of these.
		if val := workspaces.GetAttr("name"); !val.IsNull() {
			b.WorkspaceMapping.Name = val.AsString()
		}
		if val := workspaces.GetAttr("tags"); !val.IsNull() {
			var tags []string
			err := gocty.FromCtyValue(val, &tags)
			if err != nil {
				log.Panicf("An unxpected error occurred: %s", err)
			}

			b.WorkspaceMapping.Tags = tags
		}
	}

	// Determine if we are forced to use the local backend.
	b.forceLocal = os.Getenv("TF_FORCE_LOCAL_BACKEND") != ""

	return diags
}

// discover the TFC/E API service URL and version constraints.
func (b *Cloud) discover() (*url.URL, error) {
	hostname, err := svchost.ForComparison(b.hostname)
	if err != nil {
		return nil, err
	}

	host, err := b.services.Discover(hostname)
	if err != nil {
		return nil, err
	}

	service, err := host.ServiceURL(tfeServiceID)
	// Return the error, unless its a disco.ErrVersionNotSupported error.
	if _, ok := err.(*disco.ErrVersionNotSupported); !ok && err != nil {
		return nil, err
	}

	return service, err
}

// token returns the token for this host as configured in the credentials
// section of the CLI Config File. If no token was configured, an empty
// string will be returned instead.
func (b *Cloud) token() (string, error) {
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
func (b *Cloud) retryLogHook(attemptNum int, resp *http.Response) {
	if b.CLI != nil {
		// Ignore the first retry to make sure any delayed output will
		// be written to the console before we start logging retries.
		//
		// The retry logic in the TFE client will retry both rate limited
		// requests and server errors, but in the cloud backend we only
		// care about server errors so we ignore rate limit (429) errors.
		if attemptNum == 0 || (resp != nil && resp.StatusCode == 429) {
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

// Workspaces implements backend.Enhanced, returning a filtered list of workspace names according to
// the workspace mapping strategy configured.
func (b *Cloud) Workspaces() ([]string, error) {
	// Create a slice to contain all the names.
	var names []string

	// If configured for a single workspace, return that exact name only.  The StateMgr for this
	// backend will automatically create the remote workspace if it does not yet exist.
	if b.WorkspaceMapping.Strategy() == WorkspaceNameStrategy {
		names = append(names, b.WorkspaceMapping.Name)
		return names, nil
	}

	// Otherwise, multiple workspaces are being mapped. Query Terraform Cloud for all the remote
	// workspaces by the provided mapping strategy.
	options := tfe.WorkspaceListOptions{}
	if b.WorkspaceMapping.Strategy() == WorkspaceTagsStrategy {
		taglist := strings.Join(b.WorkspaceMapping.Tags, ",")
		options.Tags = &taglist
	}

	for {
		wl, err := b.client.Workspaces.List(context.Background(), b.organization, options)
		if err != nil {
			return nil, err
		}

		for _, w := range wl.Items {
			names = append(names, w.Name)
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

// DeleteWorkspace implements backend.Enhanced.
func (b *Cloud) DeleteWorkspace(name string) error {
	if name == backend.DefaultStateName {
		return backend.ErrDefaultWorkspaceNotSupported
	}

	if b.WorkspaceMapping.Strategy() == WorkspaceNameStrategy {
		return backend.ErrWorkspacesNotSupported
	}

	// Configure the remote workspace name.
	client := &remoteClient{
		client:       b.client,
		organization: b.organization,
		workspace: &tfe.Workspace{
			Name: name,
		},
	}

	return client.Delete()
}

// StateMgr implements backend.Enhanced.
func (b *Cloud) StateMgr(name string) (statemgr.Full, error) {
	var remoteTFVersion string

	if name == backend.DefaultStateName {
		return nil, backend.ErrDefaultWorkspaceNotSupported
	}

	if b.WorkspaceMapping.Strategy() == WorkspaceNameStrategy && name != b.WorkspaceMapping.Name {
		return nil, backend.ErrWorkspacesNotSupported
	}

	workspace, err := b.client.Workspaces.Read(context.Background(), b.organization, name)
	if err != nil && err != tfe.ErrResourceNotFound {
		return nil, fmt.Errorf("Failed to retrieve workspace %s: %v", name, err)
	}
	if workspace != nil {
		remoteTFVersion = workspace.TerraformVersion
	}

	if err == tfe.ErrResourceNotFound {
		// Create a workspace
		options := tfe.WorkspaceCreateOptions{
			Name: tfe.String(name),
			Tags: b.WorkspaceMapping.tfeTags(),
		}

		log.Printf("[TRACE] cloud: Creating Terraform Cloud workspace %s/%s", b.organization, name)
		workspace, err = b.client.Workspaces.Create(context.Background(), b.organization, options)
		if err != nil {
			return nil, fmt.Errorf("Error creating workspace %s: %v", name, err)
		}

		remoteTFVersion = workspace.TerraformVersion

		// Attempt to set the new workspace to use this version of Terraform. This
		// can fail if there's no enabled tool_version whose name matches our
		// version string, but that's expected sometimes -- just warn and continue.
		versionOptions := tfe.WorkspaceUpdateOptions{
			TerraformVersion: tfe.String(tfversion.String()),
		}
		_, err := b.client.Workspaces.UpdateByID(context.Background(), workspace.ID, versionOptions)
		if err == nil {
			remoteTFVersion = tfversion.String()
		} else {
			// TODO: Ideally we could rely on the client to tell us what the actual
			// problem was, but we currently can't get enough context from the error
			// object to do a nicely formatted message, so we're just assuming the
			// issue was that the version wasn't available since that's probably what
			// happened.
			log.Printf("[TRACE] cloud: Attempted to select version %s for TFC workspace; unavailable, so %s will be used instead.", tfversion.String(), workspace.TerraformVersion)
			if b.CLI != nil {
				versionUnavailable := fmt.Sprintf(unavailableTerraformVersion, tfversion.String(), workspace.TerraformVersion)
				b.CLI.Output(b.Colorize().Color(versionUnavailable))
			}
		}
	}

	if b.workspaceTagsRequireUpdate(workspace, b.WorkspaceMapping) {
		options := tfe.WorkspaceAddTagsOptions{
			Tags: b.WorkspaceMapping.tfeTags(),
		}
		log.Printf("[TRACE] cloud: Adding tags for Terraform Cloud workspace %s/%s", b.organization, name)
		err = b.client.Workspaces.AddTags(context.Background(), workspace.ID, options)
		if err != nil {
			return nil, fmt.Errorf("Error updating workspace %s: %v", name, err)
		}
	}

	// This is a fallback error check. Most code paths should use other
	// mechanisms to check the version, then set the ignoreVersionConflict
	// field to true. This check is only in place to ensure that we don't
	// accidentally upgrade state with a new code path, and the version check
	// logic is coarser and simpler.
	if !b.ignoreVersionConflict {
		// Explicitly ignore the pseudo-version "latest" here, as it will cause
		// plan and apply to always fail.
		if remoteTFVersion != tfversion.String() && remoteTFVersion != "latest" {
			return nil, fmt.Errorf("Remote workspace Terraform version %q does not match local Terraform version %q", remoteTFVersion, tfversion.String())
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

// Operation implements backend.Enhanced.
func (b *Cloud) Operation(ctx context.Context, op *backend.Operation) (*backend.RunningOperation, error) {
	name := op.Workspace

	// Retrieve the workspace for this operation.
	w, err := b.client.Workspaces.Read(ctx, b.organization, name)
	if err != nil {
		switch err {
		case context.Canceled:
			return nil, err
		case tfe.ErrResourceNotFound:
			return nil, fmt.Errorf(
				"workspace %s not found\n\n"+
					"For security, Terraform Cloud returns '404 Not Found' responses for resources\n"+
					"for resources that a user doesn't have access to, in addition to resources that\n"+
					"do not exist. If the resource does exist, please check the permissions of the provided token.",
				name,
			)
		default:
			return nil, fmt.Errorf(
				"Terraform Cloud returned an unexpected error:\n\n%s",
				err,
			)
		}
	}

	// Terraform remote version conflicts are not a concern for operations. We
	// are in one of three states:
	//
	// - Running remotely, in which case the local version is irrelevant;
	// - Workspace configured for local operations, in which case the remote
	//   version is meaningless;
	// - Forcing local operations, which should only happen in the Terraform Cloud worker, in
	//   which case the Terraform versions by definition match.
	b.IgnoreVersionConflict()

	// Check if we need to use the local backend to run the operation.
	if b.forceLocal || isLocalExecutionMode(w.ExecutionMode) {
		// Record that we're forced to run operations locally to allow the
		// command package UI to operate correctly
		b.forceLocal = true
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
	case backend.OperationTypeRefresh:
		// The `terraform refresh` command has been deprecated in favor of `terraform apply -refresh-state`.
		// Rather than respond with an error telling the user to run the other command we can just run
		// that command instead. We will tell the user what we are doing, and then do it.
		if b.CLI != nil {
			b.CLI.Output(b.Colorize().Color(strings.TrimSpace(refreshToApplyRefresh) + "\n"))
		}
		op.PlanMode = plans.RefreshOnlyMode
		op.PlanRefresh = true
		op.AutoApprove = true
		f = b.opApply
	default:
		return nil, fmt.Errorf(
			"\n\nTerraform Cloud does not support the %q operation.", op.Type)
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
			var diags tfdiags.Diagnostics
			diags = diags.Append(opErr)
			op.ReportResult(runningOp, diags)
			return
		}

		if r == nil && opErr == context.Canceled {
			runningOp.Result = backend.OperationFailure
			return
		}

		if r != nil {
			// Retrieve the run to get its current status.
			r, err := b.client.Runs.Read(cancelCtx, r.ID)
			if err != nil {
				var diags tfdiags.Diagnostics
				diags = diags.Append(generalError("Failed to retrieve run", err))
				op.ReportResult(runningOp, diags)
				return
			}

			// Record if there are any changes.
			runningOp.PlanEmpty = !r.HasChanges

			if opErr == context.Canceled {
				if err := b.cancel(cancelCtx, op, r); err != nil {
					var diags tfdiags.Diagnostics
					diags = diags.Append(generalError("Failed to retrieve run", err))
					op.ReportResult(runningOp, diags)
					return
				}
			}

			if r.Status == tfe.RunCanceled || r.Status == tfe.RunErrored {
				runningOp.Result = backend.OperationFailure
			}
		}
	}()

	// Return the running operation.
	return runningOp, nil
}

func (b *Cloud) cancel(cancelCtx context.Context, op *backend.Operation, r *tfe.Run) error {
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
				return generalError("Failed asking to cancel", err)
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
			return generalError("Failed to cancel run", err)
		}
		if b.CLI != nil {
			b.CLI.Output(b.Colorize().Color(strings.TrimSpace(operationCanceled)))
		}
	}

	return nil
}

// IgnoreVersionConflict allows commands to disable the fall-back check that
// the local Terraform version matches the remote workspace's configured
// Terraform version. This should be called by commands where this check is
// unnecessary, such as those performing remote operations, or read-only
// operations. It will also be called if the user uses a command-line flag to
// override this check.
func (b *Cloud) IgnoreVersionConflict() {
	b.ignoreVersionConflict = true
}

// VerifyWorkspaceTerraformVersion compares the local Terraform version against
// the workspace's configured Terraform version. If they are compatible, this
// means that there are no state compatibility concerns, so it returns no
// diagnostics.
//
// If the versions aren't compatible, it returns an error (or, if
// b.ignoreVersionConflict is set, a warning).
func (b *Cloud) VerifyWorkspaceTerraformVersion(workspaceName string) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	workspace, err := b.getRemoteWorkspace(context.Background(), workspaceName)
	if err != nil {
		// If the workspace doesn't exist, there can be no compatibility
		// problem, so we can return. This is most likely to happen when
		// migrating state from a local backend to a new workspace.
		if err == tfe.ErrResourceNotFound {
			return nil
		}

		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error looking up workspace",
			fmt.Sprintf("Workspace read failed: %s", err),
		))
		return diags
	}

	// If the workspace has the pseudo-version "latest", all bets are off. We
	// cannot reasonably determine what the intended Terraform version is, so
	// we'll skip version verification.
	if workspace.TerraformVersion == "latest" {
		return nil
	}

	// If the workspace has execution-mode set to local, the remote Terraform
	// version is effectively meaningless, so we'll skip version verification.
	if isLocalExecutionMode(workspace.ExecutionMode) {
		return nil
	}

	remoteConstraint, err := version.NewConstraint(workspace.TerraformVersion)
	if err != nil {
		message := fmt.Sprintf(
			"The remote workspace specified an invalid Terraform version or constraint (%s), "+
				"and it isn't possible to determine whether the local Terraform version (%s) is compatible.",
			workspace.TerraformVersion,
			tfversion.String(),
		)
		diags = diags.Append(incompatibleWorkspaceTerraformVersion(message, b.ignoreVersionConflict))
		return diags
	}

	remoteVersion, _ := version.NewSemver(workspace.TerraformVersion)

	// We can use a looser version constraint if the workspace specifies a
	// literal Terraform version, and it is not a prerelease. The latter
	// restriction is because we cannot compare prerelease versions with any
	// operator other than simple equality.
	if remoteVersion != nil && remoteVersion.Prerelease() == "" {
		v014 := version.Must(version.NewSemver("0.14.0"))
		v120 := version.Must(version.NewSemver("1.2.0"))

		// Versions from 0.14 through the early 1.x series should be compatible
		// (though we don't know about 1.2 yet).
		if remoteVersion.GreaterThanOrEqual(v014) && remoteVersion.LessThan(v120) {
			early1xCompatible, err := version.NewConstraint(fmt.Sprintf(">= 0.14.0, < %s", v120.String()))
			if err != nil {
				panic(err)
			}
			remoteConstraint = early1xCompatible
		}

		// Any future new state format will require at least a minor version
		// increment, so x.y.* will always be compatible with each other.
		if remoteVersion.GreaterThanOrEqual(v120) {
			rwvs := remoteVersion.Segments64()
			if len(rwvs) >= 3 {
				// ~> x.y.0
				minorVersionCompatible, err := version.NewConstraint(fmt.Sprintf("~> %d.%d.0", rwvs[0], rwvs[1]))
				if err != nil {
					panic(err)
				}
				remoteConstraint = minorVersionCompatible
			}
		}
	}

	// Re-parsing tfversion.String because tfversion.SemVer omits the prerelease
	// prefix, and we want to allow constraints like `~> 1.2.0-beta1`.
	fullTfversion := version.Must(version.NewSemver(tfversion.String()))

	if remoteConstraint.Check(fullTfversion) {
		return diags
	}

	message := fmt.Sprintf(
		"The local Terraform version (%s) does not meet the version requirements for remote workspace %s/%s (%s).",
		tfversion.String(),
		b.organization,
		workspace.Name,
		remoteConstraint,
	)
	diags = diags.Append(incompatibleWorkspaceTerraformVersion(message, b.ignoreVersionConflict))
	return diags
}

func (b *Cloud) IsLocalOperations() bool {
	return b.forceLocal
}

// Colorize returns the Colorize structure that can be used for colorizing
// output. This is guaranteed to always return a non-nil value and so useful
// as a helper to wrap any potentially colored strings.
//
// TODO SvH: Rename this back to Colorize as soon as we can pass -no-color.
//lint:ignore U1000 see above todo
func (b *Cloud) cliColorize() *colorstring.Colorize {
	if b.CLIColor != nil {
		return b.CLIColor
	}

	return &colorstring.Colorize{
		Colors:  colorstring.DefaultColors,
		Disable: true,
	}
}

func (b *Cloud) workspaceTagsRequireUpdate(workspace *tfe.Workspace, workspaceMapping WorkspaceMapping) bool {
	if workspaceMapping.Strategy() != WorkspaceTagsStrategy {
		return false
	}

	existingTags := map[string]struct{}{}
	for _, t := range workspace.TagNames {
		existingTags[t] = struct{}{}
	}

	for _, tag := range workspaceMapping.Tags {
		if _, ok := existingTags[tag]; !ok {
			return true
		}
	}

	return false
}

type WorkspaceMapping struct {
	Name string
	Tags []string
}

type workspaceStrategy string

const (
	WorkspaceTagsStrategy    workspaceStrategy = "tags"
	WorkspaceNameStrategy    workspaceStrategy = "name"
	WorkspaceNoneStrategy    workspaceStrategy = "none"
	WorkspaceInvalidStrategy workspaceStrategy = "invalid"
)

func (wm WorkspaceMapping) Strategy() workspaceStrategy {
	switch {
	case len(wm.Tags) > 0 && wm.Name == "":
		return WorkspaceTagsStrategy
	case len(wm.Tags) == 0 && wm.Name != "":
		return WorkspaceNameStrategy
	case len(wm.Tags) == 0 && wm.Name == "":
		return WorkspaceNoneStrategy
	default:
		// Any other combination is invalid as each strategy is mutually exclusive
		return WorkspaceInvalidStrategy
	}
}

func isLocalExecutionMode(execMode string) bool {
	return execMode == "local"
}

func (wm WorkspaceMapping) tfeTags() []*tfe.Tag {
	var tags []*tfe.Tag

	if wm.Strategy() != WorkspaceTagsStrategy {
		return tags
	}

	for _, tag := range wm.Tags {
		t := tfe.Tag{Name: tag}
		tags = append(tags, &t)
	}

	return tags
}

func generalError(msg string, err error) error {
	var diags tfdiags.Diagnostics

	if urlErr, ok := err.(*url.Error); ok {
		err = urlErr.Err
	}

	switch err {
	case context.Canceled:
		return err
	case tfe.ErrResourceNotFound:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			fmt.Sprintf("%s: %v", msg, err),
			"For security, Terraform Cloud returns '404 Not Found' responses for resources\n"+
				"for resources that a user doesn't have access to, in addition to resources that\n"+
				"do not exist. If the resource does exist, please check the permissions of the provided token.",
		))
		return diags.Err()
	default:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			fmt.Sprintf("%s: %v", msg, err),
			`Terraform Cloud returned an unexpected error. Sometimes `+
				`this is caused by network connection problems, in which case you could retry `+
				`the command. If the issue persists please open a support ticket to get help `+
				`resolving the problem.`,
		))
		return diags.Err()
	}
}

// The newline in this error is to make it look good in the CLI!
const initialRetryError = `
[reset][yellow]There was an error connecting to Terraform Cloud. Please do not exit
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

const refreshToApplyRefresh = `[bold][yellow]Proceeding with 'terraform apply -refresh-only -auto-approve'.[reset]`

const unavailableTerraformVersion = `
[reset][yellow]The local Terraform version (%s) is not available in Terraform Cloud, or your
organization does not have access to it. The new workspace will use %s. You can
change this later in the workspace settings.[reset]`

const cloudIntegrationUsedInUnsupportedTFE = `
This version of Terraform Cloud/Enterprise does not support the state mechanism
attempting to be used by the platform. This should never happen.

Please reach out to HashiCorp Support to resolve this issue.`

var (
	workspaceConfigurationHelp = fmt.Sprintf(
		`The 'workspaces' block configures how Terraform CLI maps its workspaces for this single
configuration to workspaces within a Terraform Cloud organization. Two strategies are available:

[bold]tags[reset] - %s

[bold]name[reset] - %s`, schemaDescriptionTags, schemaDescriptionName)

	schemaDescriptionHostname = `The Terraform Enterprise hostname to connect to. This optional argument defaults to app.terraform.io
for use with Terraform Cloud.`

	schemaDescriptionOrganization = `The name of the organization containing the targeted workspace(s).`

	schemaDescriptionToken = `The token used to authenticate with Terraform Cloud/Enterprise. Typically this argument should not
be set, and 'terraform login' used instead; your credentials will then be fetched from your CLI
configuration file or configured credential helper.`

	schemaDescriptionTags = `A set of tags used to select remote Terraform Cloud workspaces to be used for this single
configuration. New workspaces will automatically be tagged with these tag values. Generally, this
is the primary and recommended strategy to use.  This option conflicts with "name".`

	schemaDescriptionName = `The name of a single Terraform Cloud workspace to be used with this configuration.
When configured, only the specified workspace can be used. This option conflicts with "tags".`
)
