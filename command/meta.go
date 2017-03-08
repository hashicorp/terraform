package command

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-getter"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/backend/local"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/experiment"
	"github.com/hashicorp/terraform/helper/variables"
	"github.com/hashicorp/terraform/helper/wrappedstreams"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
)

// Meta are the meta-options that are available on all or most commands.
type Meta struct {
	// The exported fields below should be set by anyone using a
	// command with a Meta field. These are expected to be set externally
	// (not from within the command itself).

	Color            bool             // True if output should be colored
	GlobalPluginDirs []string         // Additional paths to search for plugins
	PluginOverrides  *PluginOverrides // legacy overrides from .terraformrc file
	Ui               cli.Ui           // Ui for output

	// ExtraHooks are extra hooks to add to the context.
	ExtraHooks []terraform.Hook

	//----------------------------------------------------------
	// Protected: commands can set these
	//----------------------------------------------------------

	// Modify the data directory location. This should be accessed through the
	// DataDir method.
	dataDir string

	// pluginPath is a user defined set of directories to look for plugins.
	// This is set during init with the `-plugin-dir` flag, saved to a file in
	// the data directory.
	// This overrides all other search paths when discovering plugins.
	pluginPath []string

	ignorePluginChecksum bool

	// Override certain behavior for tests within this package
	testingOverrides *testingOverrides

	//----------------------------------------------------------
	// Private: do not set these
	//----------------------------------------------------------

	// backendState is the currently active backend state
	backendState *terraform.BackendState

	// Variables for the context (private)
	autoKey       string
	autoVariables map[string]interface{}
	input         bool
	variables     map[string]interface{}

	// Targets for this context (private)
	targets []string

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
	// be overriden.
	//
	// backupPath is used to backup the state file before writing a modified
	// version. It defaults to stateOutPath + DefaultBackupExtension
	//
	// parallelism is used to control the number of concurrent operations
	// allowed when walking the graph
	//
	// shadow is used to enable/disable the shadow graph
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
	statePath        string
	stateOutPath     string
	backupPath       string
	parallelism      int
	shadow           bool
	provider         string
	stateLock        bool
	stateLockTimeout time.Duration
	forceInitCopy    bool
	reconfigure      bool

	// errWriter is the write side of a pipe for the FlagSet output. We need to
	// keep track of this to close previous pipes between tests. Normal
	// operation never needs to close this.
	errWriter *io.PipeWriter
	// done chan to wait for the scanner goroutine
	errScannerDone chan struct{}
}

type PluginOverrides struct {
	Providers    map[string]string
	Provisioners map[string]string
}

type testingOverrides struct {
	ProviderResolver terraform.ResourceProviderResolver
	Provisioners     map[string]terraform.ResourceProvisionerFactory
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
	return &colorstring.Colorize{
		Colors:  colorstring.DefaultColors,
		Disable: !m.color,
		Reset:   true,
	}
}

// DataDir returns the directory where local data will be stored.
// Defaults to DefaultsDataDir in the current working directory.
func (m *Meta) DataDir() string {
	dataDir := DefaultDataDir
	if m.dataDir != "" {
		dataDir = m.dataDir
	}

	return dataDir
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
	mode |= terraform.InputModeVar
	mode |= terraform.InputModeVarUnset

	return mode
}

// UIInput returns a UIInput object to be used for asking for input.
func (m *Meta) UIInput() terraform.UIInput {
	return &UIInput{
		Colorize: m.Colorize(),
	}
}

// StdinPiped returns true if the input is piped.
func (m *Meta) StdinPiped() bool {
	fi, err := wrappedstreams.Stdin().Stat()
	if err != nil {
		// If there is an error, let's just say its not piped
		return false
	}

	return fi.Mode()&os.ModeNamedPipe != 0
}

const (
	ProviderSkipVerifyEnvVar = "TF_SKIP_PROVIDER_VERIFY"
)

// contextOpts returns the options to use to initialize a Terraform
// context with the settings from this Meta.
func (m *Meta) contextOpts() *terraform.ContextOpts {
	var opts terraform.ContextOpts
	opts.Hooks = []terraform.Hook{m.uiHook(), &terraform.DebugHook{}}
	opts.Hooks = append(opts.Hooks, m.ExtraHooks...)

	vs := make(map[string]interface{})
	for k, v := range opts.Variables {
		vs[k] = v
	}
	for k, v := range m.autoVariables {
		vs[k] = v
	}
	for k, v := range m.variables {
		vs[k] = v
	}
	opts.Variables = vs

	opts.Targets = m.targets
	opts.UIInput = m.UIInput()
	opts.Parallelism = m.parallelism
	opts.Shadow = m.shadow

	// If testingOverrides are set, we'll skip the plugin discovery process
	// and just work with what we've been given, thus allowing the tests
	// to provide mock providers and provisioners.
	if m.testingOverrides != nil {
		opts.ProviderResolver = m.testingOverrides.ProviderResolver
		opts.Provisioners = m.testingOverrides.Provisioners
	} else {
		opts.ProviderResolver = m.providerResolver()
		opts.Provisioners = m.provisionerFactories()
	}

	opts.ProviderSHA256s = m.providerPluginsLock().Read()
	if v := os.Getenv(ProviderSkipVerifyEnvVar); v != "" {
		opts.SkipProviderVerify = true
	}

	opts.Meta = &terraform.ContextMeta{
		Env: m.Workspace(),
	}

	return &opts
}

// flags adds the meta flags to the given FlagSet.
func (m *Meta) flagSet(n string) *flag.FlagSet {
	f := flag.NewFlagSet(n, flag.ContinueOnError)
	f.BoolVar(&m.input, "input", true, "input")
	f.Var((*variables.Flag)(&m.variables), "var", "variables")
	f.Var((*variables.FlagFile)(&m.variables), "var-file", "variable file")
	f.Var((*FlagStringSlice)(&m.targets), "target", "resource to target")

	if m.autoKey != "" {
		f.Var((*variables.FlagFile)(&m.autoVariables), m.autoKey, "variable file")
	}

	// Advanced (don't need documentation, or unlikely to be set)
	f.BoolVar(&m.shadow, "shadow", true, "shadow graph")

	// Experimental features
	experiment.Flag(f)

	// Create an io.Writer that writes to our Ui properly for errors.
	// This is kind of a hack, but it does the job. Basically: create
	// a pipe, use a scanner to break it into lines, and output each line
	// to the UI. Do this forever.

	// If a previous pipe exists, we need to close that first.
	// This should only happen in testing.
	if m.errWriter != nil {
		m.errWriter.Close()
	}

	if m.errScannerDone != nil {
		<-m.errScannerDone
	}

	errR, errW := io.Pipe()
	errScanner := bufio.NewScanner(errR)
	m.errWriter = errW
	m.errScannerDone = make(chan struct{})
	go func() {
		defer close(m.errScannerDone)
		for errScanner.Scan() {
			m.Ui.Error(errScanner.Text())
		}
	}()
	f.SetOutput(errW)

	// Set the default Usage to empty
	f.Usage = func() {}

	// command that bypass locking will supply their own flag on this var, but
	// set the initial meta value to true as a failsafe.
	m.stateLock = true

	return f
}

// moduleStorage returns the module.Storage implementation used to store
// modules for commands.
func (m *Meta) moduleStorage(root string) getter.Storage {
	return &uiModuleStorage{
		Storage: &getter.FolderStorage{
			StorageDir: filepath.Join(root, "modules"),
		},
		Ui: m.Ui,
	}
}

// process will process the meta-parameters out of the arguments. This
// will potentially modify the args in-place. It will return the resulting
// slice.
//
// vars says whether or not we support variables.
func (m *Meta) process(args []string, vars bool) ([]string, error) {
	// We do this so that we retain the ability to technically call
	// process multiple times, even if we have no plans to do so
	if m.oldUi != nil {
		m.Ui = m.oldUi
	}

	// Set colorization
	m.color = m.Color
	for i, v := range args {
		if v == "-no-color" {
			m.color = false
			m.Color = false
			args = append(args[:i], args[i+1:]...)
			break
		}
	}

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

	// If we support vars and the default var file exists, add it to
	// the args...
	m.autoKey = ""
	if vars {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		f, err := os.Open(wd)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		fi, err := f.Stat()
		if err != nil {
			return nil, err
		}
		if !fi.IsDir() {
			return nil, err
		}

		err = nil
		var preArgs []string
		for err != io.EOF {
			var fis []os.FileInfo
			fis, err = f.Readdir(128)
			if err != nil && err != io.EOF {
				return nil, err
			}

			for _, fi := range fis {
				name := fi.Name()
				// Ignore directories, non-var-files, and ignored files
				if fi.IsDir() || !isVarFile(name) || config.IsIgnoredFile(name) {
					continue
				}

				m.autoKey = "var-file-default"
				preArgs = append(preArgs, "-"+m.autoKey, name)
			}
		}

		args = append(preArgs, args...)
	}

	return args, nil
}

// uiHook returns the UiHook to use with the context.
func (m *Meta) uiHook() *UiHook {
	return &UiHook{
		Colorize: m.Colorize(),
		Ui:       m.Ui,
	}
}

// confirm asks a yes/no confirmation.
func (m *Meta) confirm(opts *terraform.InputOpts) (bool, error) {
	if !m.Input() {
		return false, errors.New("input is disabled")
	}
	for {
		v, err := m.UIInput().Input(opts)
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
}

const (
	// ModuleDepthDefault is the default value for
	// module depth, which can be overridden by flag
	// or env var
	ModuleDepthDefault = -1

	// ModuleDepthEnvVar is the name of the environment variable that can be used to set module depth.
	ModuleDepthEnvVar = "TF_MODULE_DEPTH"
)

func (m *Meta) addModuleDepthFlag(flags *flag.FlagSet, moduleDepth *int) {
	flags.IntVar(moduleDepth, "module-depth", ModuleDepthDefault, "module-depth")
	if envVar := os.Getenv(ModuleDepthEnvVar); envVar != "" {
		if md, err := strconv.Atoi(envVar); err == nil {
			*moduleDepth = md
		}
	}
}

// outputShadowError outputs the error from ctx.ShadowError. If the
// error is nil then nothing happens. If output is false then it isn't
// outputted to the user (you can define logic to guard against outputting).
func (m *Meta) outputShadowError(err error, output bool) bool {
	// Do nothing if no error
	if err == nil {
		return false
	}

	// If not outputting, do nothing
	if !output {
		return false
	}

	// Write the shadow error output to a file
	path := fmt.Sprintf("terraform-error-%d.log", time.Now().UTC().Unix())
	if err := ioutil.WriteFile(path, []byte(err.Error()), 0644); err != nil {
		// If there is an error writing it, just let it go
		log.Printf("[ERROR] Error writing shadow error: %s", err)
		return false
	}

	// Output!
	m.Ui.Output(m.Colorize().Color(fmt.Sprintf(
		"[reset][bold][yellow]\nExperimental feature failure! Please report a bug.\n\n"+
			"This is not an error. Your Terraform operation completed successfully.\n"+
			"Your real infrastructure is unaffected by this message.\n\n"+
			"[reset][yellow]While running, Terraform sometimes tests experimental features in the\n"+
			"background. These features cannot affect real state and never touch\n"+
			"real infrastructure. If the features work properly, you see nothing.\n"+
			"If the features fail, this message appears.\n\n"+
			"You can report an issue at: https://github.com/hashicorp/terraform/issues\n\n"+
			"The failure was written to %q. Please\n"+
			"double check this file contains no sensitive information and report\n"+
			"it with your issue.\n\n"+
			"This is not an error. Your terraform operation completed successfully\n"+
			"and your real infrastructure is unaffected by this message.",
		path,
	)))

	return true
}

// WorkspaceNameEnvVar is the name of the environment variable that can be used
// to set the name of the Terraform workspace, overriding the workspace chosen
// by `terraform workspace select`.
//
// Note that this environment variable is ignored by `terraform workspace new`
// and `terraform workspace delete`.
const WorkspaceNameEnvVar = "TF_WORKSPACE"

// Workspace returns the name of the currently configured workspace, corresponding
// to the desired named state.
func (m *Meta) Workspace() string {
	current, _ := m.WorkspaceOverridden()
	return current
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

// isVarFile determines if the file ends with .tfvars or .tfvars.json
func isVarFile(path string) bool {
	return strings.HasSuffix(path, ".tfvars") ||
		strings.HasSuffix(path, ".tfvars.json")
}
