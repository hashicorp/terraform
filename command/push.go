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
	"github.com/hashicorp/terraform/terraform"
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
	args = c.Meta.process(args, true)
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

	// The pwd is used for the configuration path if one is not given
	pwd, err := os.Getwd()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error getting pwd: %s", err))
		return 1
	}

	// Get the path to the configuration depending on the args.
	var configPath string
	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error("The apply command expects at most one argument.")
		cmdFlags.Usage()
		return 1
	} else if len(args) == 1 {
		configPath = args[0]
	} else {
		configPath = pwd
	}

	// Verify the state is remote, we can't push without a remote state
	s, err := c.State()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to read state: %s", err))
		return 1
	}
	if !s.State().IsRemote() {
		c.Ui.Error(
			"Remote state is not enabled. For Atlas to run Terraform\n" +
				"for you, remote state must be used and configured. Remote\n" +
				"state via any backend is accepted, not just Atlas. To\n" +
				"configure remote state, use the `terraform remote config`\n" +
				"command.")
		return 1
	}

	// Build the context based on the arguments given
	ctx, planned, err := c.Context(contextOpts{
		Path:      configPath,
		StatePath: c.Meta.statePath,
	})

	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	if planned {
		c.Ui.Error(
			"A plan file cannot be given as the path to the configuration.\n" +
				"A path to a module (directory with configuration) must be given.")
		return 1
	}

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

		client.DefaultHeader.Set(terraform.VersionHeader, terraform.Version)

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

	// filter any overwrites from the atlas vars
	for k := range overwriteMap {
		delete(atlasVars, k)
	}

	// Set remote variables in the context if we don't have a value here. These
	// don't have to be correct, it just prevents the Input walk from prompting
	// the user for input, The atlas variable may be an hcl-encoded object, but
	// we're just going to set it as the raw string value.
	ctxVars := ctx.Variables()
	for k, av := range atlasVars {
		if _, ok := ctxVars[k]; !ok {
			ctx.SetVariable(k, av.Value)
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

	// Build the archiving options, which includes everything it can
	// by default according to VCS rules but forcing the data directory.
	archiveOpts := &archive.ArchiveOpts{
		VCS: archiveVCS,
		Extra: map[string]string{
			DefaultDataDir: c.DataDir(),
		},
	}
	if !moduleUpload {
		// If we're not uploading modules, then exclude the modules dir.
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

	// Output to the user the variables that will be uploaded
	var setVars []string
	// variables to upload
	var uploadVars []atlas.TFVar

	// Now we can combine the vars for upload to atlas and list the variables
	// we're uploading for the user
	for _, sv := range serializedVars {
		if av, ok := atlasVars[sv.Key]; ok {
			// this belongs to Atlas
			uploadVars = append(uploadVars, av)
		} else {
			// we're uploading our local version
			setVars = append(setVars, sv.Key)
			uploadVars = append(uploadVars, sv)
		}

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
                       a file. If "terraform.tfvars" is present, it will be
                       automatically loaded if this flag is not specified.

  -vcs=true            If true (default), push will upload only files
                       committed to your VCS, if detected.

  -no-color           If specified, output won't contain any color.

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

		case []interface{}:
			hcl, err = encodeHCL(v)
			if err != nil {
				break RANGE
			}

			tfv.Value = string(hcl)
			tfv.IsHCL = true

		case map[string]interface{}:
			hcl, err = encodeHCL(v)
			if err != nil {
				break RANGE
			}

			tfv.Value = string(hcl)
			tfv.IsHCL = true
		default:
			err = fmt.Errorf("unknown type %T for variable %s", v, k)
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
