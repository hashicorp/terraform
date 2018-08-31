package command

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/atlas-go/archive"
	"github.com/hashicorp/atlas-go/v1"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/version"
)

type PushCommand struct {
	Meta

	// client is the client to use for the actual push operations.
	// If this isn't set, then the Atlas client is used. This should
	// really only be set for testing reasons (and is hence not exported).
	client pushClient
}

func (c *PushCommand) Run(args []string) int {
	var atlasAddress, atlasToken string
	var archiveVCS, moduleUpload bool
	var name string
	var overwrite []string
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}
	cmdFlags := c.Meta.flagSet("push")
	cmdFlags.StringVar(&atlasAddress, "atlas-address", "", "")
	cmdFlags.StringVar(&c.Meta.statePath, "state", DefaultStateFilename, "path")
	cmdFlags.StringVar(&atlasToken, "token", "", "")
	cmdFlags.BoolVar(&moduleUpload, "upload-modules", true, "")
	cmdFlags.StringVar(&name, "name", "", "")
	cmdFlags.BoolVar(&archiveVCS, "vcs", true, "")
	cmdFlags.Var((*FlagStringSlice)(&overwrite), "overwrite", "")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	// Make a map of the set values
	overwriteMap := make(map[string]struct{}, len(overwrite))
	for _, v := range overwrite {
		overwriteMap[v] = struct{}{}
	}

	// This is a map of variables specifically from the CLI that we want to overwrite.
	// We need this because there is a chance that the user is trying to modify
	// a variable we don't see in our context, but which exists in this Terraform
	// Enterprise workspace.
	cliVars := make(map[string]string)
	for k, v := range c.variables {
		if _, ok := overwriteMap[k]; ok {
			if val, ok := v.(string); ok {
				cliVars[k] = val
			} else {
				c.Ui.Error(fmt.Sprintf("Error reading value for variable: %s", k))
				return 1
			}
		}
	}

	// Get the path to the configuration depending on the args.
	configPath, err := ModulePath(cmdFlags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Check if the path is a plan
	plan, err := c.Plan(configPath)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	if plan != nil {
		c.Ui.Error(
			"A plan file cannot be given as the path to the configuration.\n" +
				"A path to a module (directory with configuration) must be given.")
		return 1
	}

	// Load the module
	mod, diags := c.Module(configPath)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}
	if mod == nil {
		c.Ui.Error(fmt.Sprintf(
			"No configuration files found in the directory: %s\n\n"+
				"This command requires configuration to run.",
			configPath))
		return 1
	}

	var conf *config.Config
	if mod != nil {
		conf = mod.Config()
	}

	// Load the backend
	b, err := c.Backend(&BackendOpts{
		Config: conf,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load backend: %s", err))
		return 1
	}

	// We require a non-local backend
	if c.IsLocalBackend(b) {
		c.Ui.Error(
			"A remote backend is not enabled. For Atlas to run Terraform\n" +
				"for you, remote state must be used and configured. Remote \n" +
				"state via any backend is accepted, not just Atlas. To configure\n" +
				"a backend, please see the documentation at the URL below:\n\n" +
				"https://www.terraform.io/docs/state/remote.html")
		return 1
	}

	// We require a local backend
	local, ok := b.(backend.Local)
	if !ok {
		c.Ui.Error(ErrUnsupportedLocalOp)
		return 1
	}

	// Build the operation
	opReq := c.Operation()
	opReq.Module = mod
	opReq.Plan = plan

	// Get the context
	ctx, _, err := local.Context(opReq)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	defer func() {
		err := opReq.StateLocker.Unlock(nil)
		if err != nil {
			c.Ui.Error(err.Error())
		}
	}()

	// Get the configuration
	config := ctx.Module().Config()
	if name == "" {
		if config.Atlas == nil || config.Atlas.Name == "" {
			c.Ui.Error(
				"The name of this Terraform configuration in Atlas must be\n" +
					"specified within your configuration or the command-line. To\n" +
					"set it on the command-line, use the `-name` parameter.")
			return 1
		}
		name = config.Atlas.Name
	}

	// Initialize the client if it isn't given.
	if c.client == nil {
		// Make sure to nil out our client so our token isn't sitting around
		defer func() { c.client = nil }()

		// Initialize it to the default client, we set custom settings later
		client := atlas.DefaultClient()
		if atlasAddress != "" {
			client, err = atlas.NewClient(atlasAddress)
			if err != nil {
				c.Ui.Error(fmt.Sprintf("Error initializing Atlas client: %s", err))
				return 1
			}
		}

		client.DefaultHeader.Set(version.Header, version.Version)

		if atlasToken != "" {
			client.Token = atlasToken
		}

		c.client = &atlasPushClient{Client: client}
	}

	// Get the variables we already have in atlas
	atlasVars, err := c.client.Get(name)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Error looking up previously pushed configuration: %s", err))
		return 1
	}

	// Set remote variables in the context if we don't have a value here. These
	// don't have to be correct, it just prevents the Input walk from prompting
	// the user for input.
	ctxVars := ctx.Variables()
	atlasVarSentry := "ATLAS_78AC153CA649EAA44815DAD6CBD4816D"
	for k, _ := range atlasVars {
		if _, ok := ctxVars[k]; !ok {
			ctx.SetVariable(k, atlasVarSentry)
		}
	}

	// Ask for input
	if err := ctx.Input(c.InputMode()); err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Error while asking for variable input:\n\n%s", err))
		return 1
	}

	// Now that we've gone through the input walk, we can be sure we have all
	// the variables we're going to get.
	// We are going to keep these separate from the atlas variables until
	// upload, so we can notify the user which local variables we're sending.
	serializedVars, err := tfVars(ctx.Variables())
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"An error has occurred while serializing the variables for uploading:\n"+
				"%s", err))
		return 1
	}

	// Get the absolute path for our data directory, since the Extra field
	// value below needs to be absolute.
	dataDirAbs, err := filepath.Abs(c.DataDir())
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Error while expanding the data directory %q: %s", c.DataDir(), err))
		return 1
	}

	// Build the archiving options, which includes everything it can
	// by default according to VCS rules but forcing the data directory.
	archiveOpts := &archive.ArchiveOpts{
		VCS: archiveVCS,
		Extra: map[string]string{
			DefaultDataDir: archive.ExtraEntryDir,
		},
	}

	// Always store the state file in here so we can find state
	statePathKey := fmt.Sprintf("%s/%s", DefaultDataDir, DefaultStateFilename)
	archiveOpts.Extra[statePathKey] = filepath.Join(dataDirAbs, DefaultStateFilename)
	if moduleUpload {
		// If we're uploading modules, explicitly add that directory if exists.
		moduleKey := fmt.Sprintf("%s/%s", DefaultDataDir, "modules")
		moduleDir := filepath.Join(dataDirAbs, "modules")
		_, err := os.Stat(moduleDir)
		if err == nil {
			archiveOpts.Extra[moduleKey] = filepath.Join(dataDirAbs, "modules")
		}
		if err != nil && !os.IsNotExist(err) {
			c.Ui.Error(fmt.Sprintf(
				"Error checking for module dir %q: %s", moduleDir, err))
			return 1
		}
	} else {
		// If we're not uploading modules, explicitly exclude add that
		archiveOpts.Exclude = append(
			archiveOpts.Exclude,
			filepath.Join(c.DataDir(), "modules"))
	}

	archiveR, err := archive.CreateArchive(configPath, archiveOpts)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"An error has occurred while archiving the module for uploading:\n"+
				"%s", err))
		return 1
	}

	// List of the vars we're uploading to display to the user.
	// We always upload all vars from atlas, but only report them if they are overwritten.
	var setVars []string

	// variables to upload
	var uploadVars []atlas.TFVar

	// first add all the variables we want to send which have been serialized
	// from the local context.
	for _, sv := range serializedVars {
		_, inOverwrite := overwriteMap[sv.Key]
		_, inAtlas := atlasVars[sv.Key]

		// We have a variable that's not in atlas, so always send it.
		if !inAtlas {
			uploadVars = append(uploadVars, sv)
			setVars = append(setVars, sv.Key)
		}

		// We're overwriting an atlas variable.
		// We also want to check that we
		// don't send the dummy sentry value back to atlas. This could happen
		// if it's specified as an overwrite on the cli, but we didn't set a
		// new value.
		if inAtlas && inOverwrite && sv.Value != atlasVarSentry {
			uploadVars = append(uploadVars, sv)
			setVars = append(setVars, sv.Key)

			// remove this value from the atlas vars, because we're going to
			// send back the remainder regardless.
			delete(atlasVars, sv.Key)
		}
	}

	// now send back all the existing atlas vars, inserting any overwrites from the cli.
	for k, av := range atlasVars {
		if v, ok := cliVars[k]; ok {
			av.Value = v
			setVars = append(setVars, k)
		}
		uploadVars = append(uploadVars, av)
	}

	sort.Strings(setVars)
	if len(setVars) > 0 {
		c.Ui.Output(
			"The following variables will be set or overwritten within Atlas from\n" +
				"their local values. All other variables are already set within Atlas.\n" +
				"If you want to modify the value of a variable, use the Atlas web\n" +
				"interface or set it locally and use the -overwrite flag.\n\n")
		for _, v := range setVars {
			c.Ui.Output(fmt.Sprintf("  * %s", v))
		}

		// Newline
		c.Ui.Output("")
	}

	// Upsert!
	opts := &pushUpsertOptions{
		Name:      name,
		Archive:   archiveR,
		Variables: ctx.Variables(),
		TFVars:    uploadVars,
	}

	c.Ui.Output("Uploading Terraform configuration...")
	vsn, err := c.client.Upsert(opts)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"An error occurred while uploading the module:\n\n%s", err))
		return 1
	}

	c.Ui.Output(c.Colorize().Color(fmt.Sprintf(
		"[reset][bold][green]Configuration %q uploaded! (v%d)",
		name, vsn)))

	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 1
	}

	return 0
}

func (c *PushCommand) Help() string {
	helpText := `
Usage: terraform push [options] [DIR]

  Upload this Terraform module to an Atlas server for remote
  infrastructure management.

Options:

  -atlas-address=<url> An alternate address to an Atlas instance. Defaults
                       to https://atlas.hashicorp.com

  -upload-modules=true If true (default), then the modules are locked at
                       their current checkout and uploaded completely. This
                       prevents Atlas from running "terraform get".

  -name=<name>         Name of the configuration in Atlas. This can also
                       be set in the configuration itself. Format is
                       typically: "username/name".

  -token=<token>       Access token to use to upload. If blank or unspecified,
                       the ATLAS_TOKEN environmental variable will be used.

  -overwrite=foo       Variable keys that should overwrite values in Atlas.
                       Otherwise, variables already set in Atlas will overwrite
                       local values. This flag can be repeated.

  -var 'foo=bar'       Set a variable in the Terraform configuration. This
                       flag can be set multiple times.

  -var-file=foo        Set variables in the Terraform configuration from
                       a file. If "terraform.tfvars" or any ".auto.tfvars"
                       files are present, they will be automatically loaded.

  -vcs=true            If true (default), push will upload only files
                       committed to your VCS, if detected.

  -no-color            If specified, output won't contain any color.

`
	return strings.TrimSpace(helpText)
}

func sortedKeys(m map[string]interface{}) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// build the set of TFVars for push
func tfVars(vars map[string]interface{}) ([]atlas.TFVar, error) {
	var tfVars []atlas.TFVar
	var err error

RANGE:
	for _, k := range sortedKeys(vars) {
		v := vars[k]

		var hcl []byte
		tfv := atlas.TFVar{Key: k}

		switch v := v.(type) {
		case string:
			tfv.Value = v

		default:
			// everything that's not a string is now HCL encoded
			hcl, err = encodeHCL(v)
			if err != nil {
				break RANGE
			}

			tfv.Value = string(hcl)
			tfv.IsHCL = true
		}

		tfVars = append(tfVars, tfv)
	}

	return tfVars, err
}

func (c *PushCommand) Synopsis() string {
	return "Upload this Terraform module to Atlas to run"
}

// pushClient is implemented internally to control where pushes go. This is
// either to Atlas or a mock for testing. We still return a map to make it
// easier to check for variable existence when filtering the overrides.
type pushClient interface {
	Get(string) (map[string]atlas.TFVar, error)
	Upsert(*pushUpsertOptions) (int, error)
}

type pushUpsertOptions struct {
	Name      string
	Archive   *archive.Archive
	Variables map[string]interface{}
	TFVars    []atlas.TFVar
}

type atlasPushClient struct {
	Client *atlas.Client
}

func (c *atlasPushClient) Get(name string) (map[string]atlas.TFVar, error) {
	user, name, err := atlas.ParseSlug(name)
	if err != nil {
		return nil, err
	}

	version, err := c.Client.TerraformConfigLatest(user, name)
	if err != nil {
		return nil, err
	}

	variables := make(map[string]atlas.TFVar)

	if version == nil {
		return variables, nil
	}

	// Variables is superseded by TFVars
	if version.TFVars == nil {
		for k, v := range version.Variables {
			variables[k] = atlas.TFVar{Key: k, Value: v}
		}
	} else {
		for _, v := range version.TFVars {
			variables[v.Key] = v
		}
	}

	return variables, nil
}

func (c *atlasPushClient) Upsert(opts *pushUpsertOptions) (int, error) {
	user, name, err := atlas.ParseSlug(opts.Name)
	if err != nil {
		return 0, err
	}

	data := &atlas.TerraformConfigVersion{
		TFVars: opts.TFVars,
	}

	version, err := c.Client.CreateTerraformConfigVersion(
		user, name, data, opts.Archive, opts.Archive.Size)
	if err != nil {
		return 0, err
	}

	return version, nil
}

type mockPushClient struct {
	File string

	GetCalled bool
	GetName   string
	GetResult map[string]atlas.TFVar
	GetError  error

	UpsertCalled  bool
	UpsertOptions *pushUpsertOptions
	UpsertVersion int
	UpsertError   error
}

func (c *mockPushClient) Get(name string) (map[string]atlas.TFVar, error) {
	c.GetCalled = true
	c.GetName = name
	return c.GetResult, c.GetError
}

func (c *mockPushClient) Upsert(opts *pushUpsertOptions) (int, error) {
	f, err := os.Create(c.File)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	data := opts.Archive
	size := opts.Archive.Size
	if _, err := io.CopyN(f, data, size); err != nil {
		return 0, err
	}

	c.UpsertCalled = true
	c.UpsertOptions = opts
	return c.UpsertVersion, c.UpsertError
}
