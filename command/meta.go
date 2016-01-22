package command

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hashicorp/go-getter"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
)

// Meta are the meta-options that are available on all or most commands.
type Meta struct {
	Color       bool
	ContextOpts *terraform.ContextOpts
	Ui          cli.Ui

	// State read when calling `Context`. This is available after calling
	// `Context`.
	state       state.State
	stateResult *StateResult

	// This can be set by the command itself to provide extra hooks.
	extraHooks []terraform.Hook

	// This can be set by tests to change some directories
	dataDir string

	// Variables for the context (private)
	autoKey       string
	autoVariables map[string]string
	input         bool
	variables     map[string]string

	// Targets for this context (private)
	targets []string

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
	statePath    string
	stateOutPath string
	backupPath   string
	parallelism  int
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

// Context returns a Terraform Context taking into account the context
// options used to initialize this meta configuration.
func (m *Meta) Context(copts contextOpts) (*terraform.Context, bool, error) {
	opts := m.contextOpts()

	// First try to just read the plan directly from the path given.
	f, err := os.Open(copts.Path)
	if err == nil {
		plan, err := terraform.ReadPlan(f)
		f.Close()
		if err == nil {
			// Setup our state
			state, statePath, err := StateFromPlan(m.statePath, plan)
			if err != nil {
				return nil, false, fmt.Errorf("Error loading plan: %s", err)
			}

			// Set our state
			m.state = state
			m.stateOutPath = statePath

			if len(m.variables) > 0 {
				return nil, false, fmt.Errorf(
					"You can't set variables with the '-var' or '-var-file' flag\n" +
						"when you're applying a plan file. The variables used when\n" +
						"the plan was created will be used. If you wish to use different\n" +
						"variable values, create a new plan file.")
			}

			return plan.Context(opts), true, nil
		}
	}

	// Load the statePath if not given
	if copts.StatePath != "" {
		m.statePath = copts.StatePath
	}

	// Tell the context if we're in a destroy plan / apply
	opts.Destroy = copts.Destroy

	// Store the loaded state
	state, err := m.State()
	if err != nil {
		return nil, false, err
	}

	// Load the root module
	mod, err := module.NewTreeModule("", copts.Path)
	if err != nil {
		return nil, false, fmt.Errorf("Error loading config: %s", err)
	}

	err = mod.Load(m.moduleStorage(m.DataDir()), copts.GetMode)
	if err != nil {
		return nil, false, fmt.Errorf("Error downloading modules: %s", err)
	}

	opts.Module = mod
	opts.Parallelism = copts.Parallelism
	opts.State = state.State()
	ctx := terraform.NewContext(opts)
	return ctx, false, nil
}

// DataDir returns the directory where local data will be stored.
func (m *Meta) DataDir() string {
	dataDir := DefaultDataDirectory
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
	if len(m.variables) == 0 {
		mode |= terraform.InputModeVar
		mode |= terraform.InputModeVarUnset
	}

	return mode
}

// State returns the state for this meta.
func (m *Meta) State() (state.State, error) {
	if m.state != nil {
		return m.state, nil
	}

	result, err := State(m.StateOpts())
	if err != nil {
		return nil, err
	}

	m.state = result.State
	m.stateOutPath = result.StatePath
	m.stateResult = result
	return m.state, nil
}

// StateRaw is used to setup the state manually.
func (m *Meta) StateRaw(opts *StateOpts) (*StateResult, error) {
	result, err := State(opts)
	if err != nil {
		return nil, err
	}

	m.state = result.State
	m.stateOutPath = result.StatePath
	m.stateResult = result
	return result, nil
}

// StateOpts returns the default state options
func (m *Meta) StateOpts() *StateOpts {
	localPath := m.statePath
	if localPath == "" {
		localPath = DefaultStateFilename
	}
	remotePath := filepath.Join(m.DataDir(), DefaultStateFilename)

	return &StateOpts{
		LocalPath:     localPath,
		LocalPathOut:  m.stateOutPath,
		RemotePath:    remotePath,
		RemoteRefresh: true,
		BackupPath:    m.backupPath,
	}
}

// UIInput returns a UIInput object to be used for asking for input.
func (m *Meta) UIInput() terraform.UIInput {
	return &UIInput{
		Colorize: m.Colorize(),
	}
}

// PersistState is used to write out the state, handling backup of
// the existing state file and respecting path configurations.
func (m *Meta) PersistState(s *terraform.State) error {
	if err := m.state.WriteState(s); err != nil {
		return err
	}

	return m.state.PersistState()
}

// Input returns true if we should ask for input for context.
func (m *Meta) Input() bool {
	return !test && m.input && len(m.variables) == 0
}

// contextOpts returns the options to use to initialize a Terraform
// context with the settings from this Meta.
func (m *Meta) contextOpts() *terraform.ContextOpts {
	var opts terraform.ContextOpts = *m.ContextOpts
	opts.Hooks = make(
		[]terraform.Hook,
		len(m.ContextOpts.Hooks)+len(m.extraHooks)+1)
	opts.Hooks[0] = m.uiHook()
	copy(opts.Hooks[1:], m.ContextOpts.Hooks)
	copy(opts.Hooks[len(m.ContextOpts.Hooks)+1:], m.extraHooks)

	vs := make(map[string]string)
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

	return &opts
}

// flags adds the meta flags to the given FlagSet.
func (m *Meta) flagSet(n string) *flag.FlagSet {
	f := flag.NewFlagSet(n, flag.ContinueOnError)
	f.BoolVar(&m.input, "input", true, "input")
	f.Var((*FlagKV)(&m.variables), "var", "variables")
	f.Var((*FlagKVFile)(&m.variables), "var-file", "variable file")
	f.Var((*FlagStringSlice)(&m.targets), "target", "resource to target")

	if m.autoKey != "" {
		f.Var((*FlagKVFile)(&m.autoVariables), m.autoKey, "variable file")
	}

	// Create an io.Writer that writes to our Ui properly for errors.
	// This is kind of a hack, but it does the job. Basically: create
	// a pipe, use a scanner to break it into lines, and output each line
	// to the UI. Do this forever.
	errR, errW := io.Pipe()
	errScanner := bufio.NewScanner(errR)
	go func() {
		for errScanner.Scan() {
			m.Ui.Error(errScanner.Text())
		}
	}()
	f.SetOutput(errW)

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
func (m *Meta) process(args []string, vars bool) []string {
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
		if _, err := os.Stat(DefaultVarsFilename); err == nil {
			m.autoKey = "var-file-default"
			args = append(args, "", "")
			copy(args[2:], args[0:])
			args[0] = "-" + m.autoKey
			args[1] = DefaultVarsFilename
		}

		if _, err := os.Stat(DefaultVarsFilename + ".json"); err == nil {
			m.autoKey = "var-file-default"
			args = append(args, "", "")
			copy(args[2:], args[0:])
			args[0] = "-" + m.autoKey
			args[1] = DefaultVarsFilename + ".json"
		}
	}

	return args
}

// uiHook returns the UiHook to use with the context.
func (m *Meta) uiHook() *UiHook {
	return &UiHook{
		Colorize: m.Colorize(),
		Ui:       m.Ui,
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

// contextOpts are the options used to load a context from a command.
type contextOpts struct {
	// Path to the directory where the root module is.
	Path string

	// StatePath is the path to the state file. If this is empty, then
	// no state will be loaded. It is also okay for this to be a path to
	// a file that doesn't exist; it is assumed that this means that there
	// is simply no state.
	StatePath string

	// GetMode is the module.GetMode to use when loading the module tree.
	GetMode module.GetMode

	// Set to true when running a destroy plan/apply.
	Destroy bool

	// Number of concurrent operations allowed
	Parallelism int
}
