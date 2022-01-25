package command

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/command/webbrowser"
	"github.com/hashicorp/terraform/internal/command/workdir"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/getproviders"
	legacy "github.com/hashicorp/terraform/internal/legacy/terraform"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/provisioners"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Meta are the meta-options that are available on all or most commands.
type Meta struct {
	// The exported fields below should be set by anyone using a
	// command with a Meta field. These are expected to be set externally
	// (not from within the command itself).

	// WorkingDir is an object representing the "working directory" where we're
	// running commands. In the normal case this literally refers to the
	// working directory of the Terraform process, though this can take on
	// a more symbolic meaning when the user has overridden default behavior
	// to specify a different working directory or to override the special
	// data directory where we'll persist settings that must survive between
	// consecutive commands.
	//
	// We're currently gradually migrating the various bits of state that
	// must persist between consecutive commands in a session to be encapsulated
	// in here, but we're not there yet and so there are also some methods on
	// Meta which directly read and modify paths inside the data directory.
	WorkingDir *workdir.Dir

	// Streams tracks the raw Stdout, Stderr, and Stdin handles along with
	// some basic metadata about them, such as whether each is connected to
	// a terminal, how wide the possible terminal is, etc.
	//
	// For historical reasons this might not be set in unit test code, and
	// so functions working with this field must check if it's nil and
	// do some default behavior instead if so, rather than panicking.
	Streams *terminal.Streams

	View *views.View

	Color            bool     // True if output should be colored
	GlobalPluginDirs []string // Additional paths to search for plugins
	Ui               cli.Ui   // Ui for output

	// Services provides access to remote endpoint information for
	// "terraform-native' services running at a specific user-facing hostname.
	Services *disco.Disco

	// RunningInAutomation indicates that commands are being run by an
	// automated system rather than directly at a command prompt.
	//
	// This is a hint to various command routines that it may be confusing
	// to print out messages that suggest running specific follow-up
	// commands, since the user consuming the output will not be
	// in a position to run such commands.
	//
	// The intended use-case of this flag is when Terraform is running in
	// some sort of workflow orchestration tool which is abstracting away
	// the specific commands being run.
	RunningInAutomation bool

	// CLIConfigDir is the directory from which CLI configuration files were
	// read by the caller and the directory where any changes to CLI
	// configuration files by commands should be made.
	//
	// If this is empty then no configuration directory is available and
	// commands which require one cannot proceed.
	CLIConfigDir string

	// PluginCacheDir, if non-empty, enables caching of downloaded plugins
	// into the given directory.
	PluginCacheDir string

	// ProviderSource allows determining the available versions of a provider
	// and determines where a distribution package for a particular
	// provider version can be obtained.
	ProviderSource getproviders.Source

	// BrowserLauncher is used by commands that need to open a URL in a
	// web browser.
	BrowserLauncher webbrowser.Launcher

	// When this channel is closed, the command will be cancelled.
	ShutdownCh <-chan struct{}

	// ProviderDevOverrides are providers where we ignore the lock file, the
	// configured version constraints, and the local cache directory and just
	// always use exactly the path specified. This is intended to allow
	// provider developers to easily test local builds without worrying about
	// what version number they might eventually be released as, or what
	// checksums they have.
	ProviderDevOverrides map[addrs.Provider]getproviders.PackageLocalDir

	// UnmanagedProviders are a set of providers that exist as processes
	// predating Terraform, which Terraform should use but not worry about the
	// lifecycle of.
	//
	// This is essentially a more extreme version of ProviderDevOverrides where
	// Terraform doesn't even worry about how the provider server gets launched,
	// just trusting that someone else did it before running Terraform.
	UnmanagedProviders map[addrs.Provider]*plugin.ReattachConfig

	//----------------------------------------------------------
	// Protected: commands can set these
	//----------------------------------------------------------

	// pluginPath is a user defined set of directories to look for plugins.
	// This is set during init with the `-plugin-dir` flag, saved to a file in
	// the data directory.
	// This overrides all other search paths when discovering plugins.
	pluginPath []string

	// Override certain behavior for tests within this package
	testingOverrides *testingOverrides

	//----------------------------------------------------------
	// Private: do not set these
	//----------------------------------------------------------

	// configLoader is a shared configuration loader that is used by
	// LoadConfig and other commands that access configuration files.
	// It is initialized on first use.
	configLoader *configload.Loader

	// backendState is the currently active backend state
	backendState *legacy.BackendState

	// Variables for the context (private)
	variableArgs rawFlags
	input        bool

	// Targets for this context (private)
	targets     []addrs.Targetable
	targetFlags []string

	// Internal fields
	color bool
	oldUi cli.Ui

	// The fields below are expected to be set by the command via
	// command line flags. See the Apply command for an example.
	//
	// statePath is the path to the state file. If this is empty, then
	// no state will be loaded. It is also okay for this to be a path to
	// a file that doesn't exist; it is assumed that this means that there
	// is simply no state.
	//
	// stateOutPath is used to override the output path for the state.
	// If not provided, the StatePath is used causing the old state to
	// be overridden.
	//
	// backupPath is used to backup the state file before writing a modified
	// version. It defaults to stateOutPath + DefaultBackupExtension
	//
	// parallelism is used to control the number of concurrent operations
	// allowed when walking the graph
	//
	// provider is to specify specific resource providers
	//
	// stateLock is set to false to disable state locking
	//
	// stateLockTimeout is the optional duration to retry a state locks locks
	// when it is already locked by another process.
	//
	// forceInitCopy suppresses confirmation for copying state data during
	// init.
	//
	// reconfigure forces init to ignore any stored configuration.
	//
	// migrateState confirms the user wishes to migrate from the prior backend
	// configuration to a new configuration.
	//
	// compactWarnings (-compact-warnings) selects a more compact presentation
	// of warnings in the output when they are not accompanied by errors.
	statePath        string
	stateOutPath     string
	backupPath       string
	parallelism      int
	stateLock        bool
	stateLockTimeout time.Duration
	forceInitCopy    bool
	reconfigure      bool
	migrateState     bool
	compactWarnings  bool

	// Used with the import command to allow import of state when no matching config exists.
	allowMissingConfig bool

	// Used with commands which write state to allow users to write remote
	// state even if the remote and local Terraform versions don't match.
	ignoreRemoteVersion bool
}

type testingOverrides struct {
	Providers    map[addrs.Provider]providers.Factory
	Provisioners map[string]provisioners.Factory
}

// initStatePaths is used to initialize the default values for
// statePath, stateOutPath, and backupPath
func (m *Meta) initStatePaths() {
	if m.statePath == "" {
		m.statePath = DefaultStateFilename
	}
	if m.stateOutPath == "" {
		m.stateOutPath = m.statePath
	}
	if m.backupPath == "" {
		m.backupPath = m.stateOutPath + DefaultBackupExtension
	}
}

// StateOutPath returns the true output path for the state file
func (m *Meta) StateOutPath() string {
	return m.stateOutPath
}

// Colorize returns the colorization structure for a command.
func (m *Meta) Colorize() *colorstring.Colorize {
	colors := make(map[string]string)
	for k, v := range colorstring.DefaultColors {
		colors[k] = v
	}
	colors["purple"] = "38;5;57"

	return &colorstring.Colorize{
		Colors:  colors,
		Disable: !m.color,
		Reset:   true,
	}
}

// fixupMissingWorkingDir is a compensation for various existing tests which
// directly construct incomplete "Meta" objects. Specifically, it deals with
// a test that omits a WorkingDir value by constructing one just-in-time.
//
// We shouldn't ever rely on this in any real codepath, because it doesn't
// take into account the various ways users can override our default
// directory selection behaviors.
func (m *Meta) fixupMissingWorkingDir() {
	if m.WorkingDir == nil {
		log.Printf("[WARN] This 'Meta' object is missing its WorkingDir, so we're creating a default one suitable only for tests")
		m.WorkingDir = workdir.NewDir(".")
	}
}

// DataDir returns the directory where local data will be stored.
// Defaults to DefaultDataDir in the current working directory.
func (m *Meta) DataDir() string {
	m.fixupMissingWorkingDir()
	return m.WorkingDir.DataDir()
}

const (
	// InputModeEnvVar is the environment variable that, if set to "false" or
	// "0", causes terraform commands to behave as if the `-input=false` flag was
	// specified.
	InputModeEnvVar = "TF_INPUT"
)

// InputMode returns the type of input we should ask for in the form of
// terraform.InputMode which is passed directly to Context.Input.
func (m *Meta) InputMode() terraform.InputMode {
	if test || !m.input {
		return 0
	}

	if envVar := os.Getenv(InputModeEnvVar); envVar != "" {
		if v, err := strconv.ParseBool(envVar); err == nil {
			if !v {
				return 0
			}
		}
	}

	var mode terraform.InputMode
	mode |= terraform.InputModeProvider

	return mode
}

// UIInput returns a UIInput object to be used for asking for input.
func (m *Meta) UIInput() terraform.UIInput {
	return &UIInput{
		Colorize: m.Colorize(),
	}
}

// OutputColumns returns the number of columns that normal (non-error) UI
// output should be wrapped to fill.
//
// This is the column count to use if you'll be printing your message via
// the Output or Info methods of m.Ui.
func (m *Meta) OutputColumns() int {
	if m.Streams == nil {
		// A default for unit tests that don't populate Meta fully.
		return 78
	}
	return m.Streams.Stdout.Columns()
}

// ErrorColumns returns the number of columns that error UI output should be
// wrapped to fill.
//
// This is the column count to use if you'll be printing your message via
// the Error or Warn methods of m.Ui.
func (m *Meta) ErrorColumns() int {
	if m.Streams == nil {
		// A default for unit tests that don't populate Meta fully.
		return 78
	}
	return m.Streams.Stderr.Columns()
}

// StdinPiped returns true if the input is piped.
func (m *Meta) StdinPiped() bool {
	if m.Streams == nil {
		// If we don't have m.Streams populated then we're presumably in a unit
		// test that doesn't properly populate Meta, so we'll just say the
		// output _isn't_ piped because that's the common case and so most likely
		// to be useful to a unit test.
		return false
	}
	return !m.Streams.Stdin.IsTerminal()
}

// InterruptibleContext returns a context.Context that will be cancelled
// if the process is interrupted by a platform-specific interrupt signal.
//
// As usual with cancelable contexts, the caller must always call the given
// cancel function once all operations are complete in order to make sure
// that the context resources will still be freed even if there is no
// interruption.
func (m *Meta) InterruptibleContext() (context.Context, context.CancelFunc) {
	base := context.Background()
	if m.ShutdownCh == nil {
		// If we're running in a unit testing context without a shutdown
		// channel populated then we'll return an uncancelable channel.
		return base, func() {}
	}

	ctx, cancel := context.WithCancel(base)
	go func() {
		select {
		case <-m.ShutdownCh:
			cancel()
		case <-ctx.Done():
			// finished without being interrupted
		}
	}()
	return ctx, cancel
}

// RunOperation executes the given operation on the given backend, blocking
// until that operation completes or is interrupted, and then returns
// the RunningOperation object representing the completed or
// aborted operation that is, despite the name, no longer running.
//
// An error is returned if the operation either fails to start or is cancelled.
// If the operation runs to completion then no error is returned even if the
// operation itself is unsuccessful. Use the "Result" field of the
// returned operation object to recognize operation-level failure.
func (m *Meta) RunOperation(b backend.Enhanced, opReq *backend.Operation) (*backend.RunningOperation, error) {
	if opReq.View == nil {
		panic("RunOperation called with nil View")
	}
	if opReq.ConfigDir != "" {
		opReq.ConfigDir = m.normalizePath(opReq.ConfigDir)
	}

	op, err := b.Operation(context.Background(), opReq)
	if err != nil {
		return nil, fmt.Errorf("error starting operation: %s", err)
	}

	// Wait for the operation to complete or an interrupt to occur
	select {
	case <-m.ShutdownCh:
		// gracefully stop the operation
		op.Stop()

		// Notify the user
		opReq.View.Interrupted()

		// Still get the result, since there is still one
		select {
		case <-m.ShutdownCh:
			opReq.View.FatalInterrupt()

			// cancel the operation completely
			op.Cancel()

			// the operation should return asap
			// but timeout just in case
			select {
			case <-op.Done():
			case <-time.After(5 * time.Second):
			}

			return nil, errors.New("operation canceled")

		case <-op.Done():
			// operation completed after Stop
		}
	case <-op.Done():
		// operation completed normally
	}

	return op, nil
}

// contextOpts returns the options to use to initialize a Terraform
// context with the settings from this Meta.
func (m *Meta) contextOpts() (*terraform.ContextOpts, error) {
	workspace, err := m.Workspace()
	if err != nil {
		return nil, err
	}

	var opts terraform.ContextOpts

	opts.UIInput = m.UIInput()
	opts.Parallelism = m.parallelism

	// If testingOverrides are set, we'll skip the plugin discovery process
	// and just work with what we've been given, thus allowing the tests
	// to provide mock providers and provisioners.
	if m.testingOverrides != nil {
		opts.Providers = m.testingOverrides.Providers
		opts.Provisioners = m.testingOverrides.Provisioners
	} else {
		var providerFactories map[addrs.Provider]providers.Factory
		providerFactories, err = m.providerFactories()
		opts.Providers = providerFactories
		opts.Provisioners = m.provisionerFactories()
	}

	opts.Meta = &terraform.ContextMeta{
		Env:                workspace,
		OriginalWorkingDir: m.WorkingDir.OriginalWorkingDir(),
	}

	return &opts, err
}

// defaultFlagSet creates a default flag set for commands.
// See also command/arguments/default.go
func (m *Meta) defaultFlagSet(n string) *flag.FlagSet {
	f := flag.NewFlagSet(n, flag.ContinueOnError)
	f.SetOutput(ioutil.Discard)

	// Set the default Usage to empty
	f.Usage = func() {}

	return f
}

// ignoreRemoteVersionFlagSet add the ignore-remote version flag to suppress
// the error when the configured Terraform version on the remote workspace
// does not match the local Terraform version.
func (m *Meta) ignoreRemoteVersionFlagSet(n string) *flag.FlagSet {
	f := m.defaultFlagSet(n)

	f.BoolVar(&m.ignoreRemoteVersion, "ignore-remote-version", false, "continue even if remote and local Terraform versions are incompatible")

	return f
}

// extendedFlagSet adds custom flags that are mostly used by commands
// that are used to run an operation like plan or apply.
func (m *Meta) extendedFlagSet(n string) *flag.FlagSet {
	f := m.defaultFlagSet(n)

	f.BoolVar(&m.input, "input", true, "input")
	f.Var((*FlagStringSlice)(&m.targetFlags), "target", "resource to target")
	f.BoolVar(&m.compactWarnings, "compact-warnings", false, "use compact warnings")

	if m.variableArgs.items == nil {
		m.variableArgs = newRawFlags("-var")
	}
	varValues := m.variableArgs.Alias("-var")
	varFiles := m.variableArgs.Alias("-var-file")
	f.Var(varValues, "var", "variables")
	f.Var(varFiles, "var-file", "variable file")

	// commands that bypass locking will supply their own flag on this var,
	// but set the initial meta value to true as a failsafe.
	m.stateLock = true

	return f
}

// process will process any -no-color entries out of the arguments. This
// will potentially modify the args in-place. It will return the resulting
// slice, and update the Meta and Ui.
func (m *Meta) process(args []string) []string {
	// We do this so that we retain the ability to technically call
	// process multiple times, even if we have no plans to do so
	if m.oldUi != nil {
		m.Ui = m.oldUi
	}

	// Set colorization
	m.color = m.Color
	i := 0 // output index
	for _, v := range args {
		if v == "-no-color" {
			m.color = false
			m.Color = false
		} else {
			// copy and increment index
			args[i] = v
			i++
		}
	}
	args = args[:i]

	// Set the UI
	m.oldUi = m.Ui
	m.Ui = &cli.ConcurrentUi{
		Ui: &ColorizeUi{
			Colorize:   m.Colorize(),
			ErrorColor: "[red]",
			WarnColor:  "[yellow]",
			Ui:         m.oldUi,
		},
	}

	// Reconfigure the view. This is necessary for commands which use both
	// views.View and cli.Ui during the migration phase.
	if m.View != nil {
		m.View.Configure(&arguments.View{
			CompactWarnings: m.compactWarnings,
			NoColor:         !m.Color,
		})
	}

	return args
}

// uiHook returns the UiHook to use with the context.
func (m *Meta) uiHook() *views.UiHook {
	return views.NewUiHook(m.View)
}

// confirm asks a yes/no confirmation.
func (m *Meta) confirm(opts *terraform.InputOpts) (bool, error) {
	if !m.Input() {
		return false, errors.New("input is disabled")
	}

	for i := 0; i < 2; i++ {
		v, err := m.UIInput().Input(context.Background(), opts)
		if err != nil {
			return false, fmt.Errorf(
				"Error asking for confirmation: %s", err)
		}

		switch strings.ToLower(v) {
		case "no":
			return false, nil
		case "yes":
			return true, nil
		}
	}
	return false, nil
}

// showDiagnostics displays error and warning messages in the UI.
//
// "Diagnostics" here means the Diagnostics type from the tfdiag package,
// though as a convenience this function accepts anything that could be
// passed to the "Append" method on that type, converting it to Diagnostics
// before displaying it.
//
// Internally this function uses Diagnostics.Append, and so it will panic
// if given unsupported value types, just as Append does.
func (m *Meta) showDiagnostics(vals ...interface{}) {
	var diags tfdiags.Diagnostics
	diags = diags.Append(vals...)
	diags.Sort()

	if len(diags) == 0 {
		return
	}

	outputWidth := m.ErrorColumns()

	diags = diags.ConsolidateWarnings(1)

	// Since warning messages are generally competing
	if m.compactWarnings {
		// If the user selected compact warnings and all of the diagnostics are
		// warnings then we'll use a more compact representation of the warnings
		// that only includes their summaries.
		// We show full warnings if there are also errors, because a warning
		// can sometimes serve as good context for a subsequent error.
		useCompact := true
		for _, diag := range diags {
			if diag.Severity() != tfdiags.Warning {
				useCompact = false
				break
			}
		}
		if useCompact {
			msg := format.DiagnosticWarningsCompact(diags, m.Colorize())
			msg = "\n" + msg + "\nTo see the full warning notes, run Terraform without -compact-warnings.\n"
			m.Ui.Warn(msg)
			return
		}
	}

	for _, diag := range diags {
		var msg string
		if m.Color {
			msg = format.Diagnostic(diag, m.configSources(), m.Colorize(), outputWidth)
		} else {
			msg = format.DiagnosticPlain(diag, m.configSources(), outputWidth)
		}

		switch diag.Severity() {
		case tfdiags.Error:
			m.Ui.Error(msg)
		case tfdiags.Warning:
			m.Ui.Warn(msg)
		default:
			m.Ui.Output(msg)
		}
	}
}

// WorkspaceNameEnvVar is the name of the environment variable that can be used
// to set the name of the Terraform workspace, overriding the workspace chosen
// by `terraform workspace select`.
//
// Note that this environment variable is ignored by `terraform workspace new`
// and `terraform workspace delete`.
const WorkspaceNameEnvVar = "TF_WORKSPACE"

var errInvalidWorkspaceNameEnvVar = fmt.Errorf("Invalid workspace name set using %s", WorkspaceNameEnvVar)

// Workspace returns the name of the currently configured workspace, corresponding
// to the desired named state.
func (m *Meta) Workspace() (string, error) {
	current, overridden := m.WorkspaceOverridden()
	if overridden && !validWorkspaceName(current) {
		return "", errInvalidWorkspaceNameEnvVar
	}
	return current, nil
}

// WorkspaceOverridden returns the name of the currently configured workspace,
// corresponding to the desired named state, as well as a bool saying whether
// this was set via the TF_WORKSPACE environment variable.
func (m *Meta) WorkspaceOverridden() (string, bool) {
	if envVar := os.Getenv(WorkspaceNameEnvVar); envVar != "" {
		return envVar, true
	}

	envData, err := ioutil.ReadFile(filepath.Join(m.DataDir(), local.DefaultWorkspaceFile))
	current := string(bytes.TrimSpace(envData))
	if current == "" {
		current = backend.DefaultStateName
	}

	if err != nil && !os.IsNotExist(err) {
		// always return the default if we can't get a workspace name
		log.Printf("[ERROR] failed to read current workspace: %s", err)
	}

	return current, false
}

// SetWorkspace saves the given name as the current workspace in the local
// filesystem.
func (m *Meta) SetWorkspace(name string) error {
	err := os.MkdirAll(m.DataDir(), 0755)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(m.DataDir(), local.DefaultWorkspaceFile), []byte(name), 0644)
	if err != nil {
		return err
	}
	return nil
}

// isAutoVarFile determines if the file ends with .auto.tfvars or .auto.tfvars.json
func isAutoVarFile(path string) bool {
	return strings.HasSuffix(path, ".auto.tfvars") ||
		strings.HasSuffix(path, ".auto.tfvars.json")
}

// FIXME: as an interim refactoring step, we apply the contents of the state
// arguments directly to the Meta object. Future work would ideally update the
// code paths which use these arguments to be passed them directly for clarity.
func (m *Meta) applyStateArguments(args *arguments.State) {
	m.stateLock = args.Lock
	m.stateLockTimeout = args.LockTimeout
	m.statePath = args.StatePath
	m.stateOutPath = args.StateOutPath
	m.backupPath = args.BackupPath
}
