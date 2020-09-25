package command

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
	tfversion "github.com/hashicorp/terraform/version"
	"github.com/zclconf/go-cty/cty"
)

// ZeroThirteenUpgradeCommand upgrades configuration files for a module
// to include explicit provider source settings
type ZeroThirteenUpgradeCommand struct {
	Meta
}

// Warning diagnostic detail message used for JSON and override config files
const skippedConfigurationFileWarning = "The %s configuration file %q was skipped, because %s files are assumed to be generated. The program that generated this file may need to be updated for changes to the configuration language."

func (c *ZeroThirteenUpgradeCommand) Run(args []string) int {
	args = c.Meta.process(args)

	var skipConfirm bool

	flags := c.Meta.defaultFlagSet("0.13upgrade")
	flags.BoolVar(&skipConfirm, "yes", false, "skip confirmation prompt")
	flags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := flags.Parse(args); err != nil {
		return 1
	}

	var diags tfdiags.Diagnostics

	var dir string
	args = flags.Args()
	switch len(args) {
	case 0:
		dir = "."
	case 1:
		dir = args[0]
	default:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Too many arguments",
			"The command 0.13upgrade expects only a single argument, giving the directory containing the module to upgrade.",
		))
		c.showDiagnostics(diags)
		return 1
	}

	// Check for user-supplied plugin path
	var err error
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading plugin path: %s", err))
		return 1
	}

	dir = c.normalizePath(dir)

	// Upgrade only if some configuration is present
	empty, err := configs.IsEmptyDir(dir)
	if err != nil {
		diags = diags.Append(fmt.Errorf("Error checking configuration: %s", err))
		return 1
	}
	if empty {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Not a module directory",
			fmt.Sprintf("The given directory %s does not contain any Terraform configuration files.", dir),
		))
		c.showDiagnostics(diags)
		return 1
	}

	// Set up the config loader and find all the config files
	loader, err := c.initConfigLoader()
	if err != nil {
		diags = diags.Append(err)
		c.showDiagnostics(diags)
		return 1
	}
	parser := loader.Parser()
	primary, overrides, hclDiags := parser.ConfigDirFiles(dir)
	diags = diags.Append(hclDiags)
	if diags.HasErrors() {
		c.Ui.Error(strings.TrimSpace("Failed to load configuration"))
		c.showDiagnostics(diags)
		return 1
	}

	// Load and parse all primary files
	files := make(map[string]*configs.File)
	for _, path := range primary {
		// Skip JSON configuration files, because we can't rewrite them and
		// they're probably generated anyway.
		if strings.HasSuffix(strings.ToLower(path), ".json") {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Warning,
				"JSON configuration file ignored",
				fmt.Sprintf(
					skippedConfigurationFileWarning,
					"JSON",
					path,
					"JSON",
				),
			))
			continue
		}
		file, fileDiags := parser.LoadConfigFile(path)
		diags = diags.Append(fileDiags)
		if file != nil {
			files[path] = file
		}
	}
	if diags.HasErrors() {
		c.Ui.Error(strings.TrimSpace("Failed to load configuration"))
		c.showDiagnostics(diags)
		return 1
	}

	// Explain what the command does and how to use it, and ask for confirmation.
	if !skipConfirm {
		c.Ui.Output(fmt.Sprintf(`
This command will update the configuration files in the given directory to use
the new provider source features from Terraform v0.13. It will also highlight
any providers for which the source cannot be detected, and advise how to
proceed.

We recommend using this command in a clean version control work tree, so that
you can easily see the proposed changes as a diff against the latest commit.
If you have uncommited changes already present, we recommend aborting this
command and dealing with them before running this command again.
`))

		query := "Would you like to upgrade the module in the current directory?"
		if dir != "." {
			query = fmt.Sprintf("Would you like to upgrade the module in %s?", dir)
		}
		v, err := c.UIInput().Input(context.Background(), &terraform.InputOpts{
			Id:          "approve",
			Query:       query,
			Description: `Only 'yes' will be accepted to confirm.`,
		})
		if err != nil {
			diags = diags.Append(err)
			c.showDiagnostics(diags)
			return 1
		}
		if v != "yes" {
			c.Ui.Info("Upgrade cancelled.")
			return 0
		}

		c.Ui.Output(`-----------------------------------------------------------------------------`)
	}

	// It's not clear what the correct behaviour is for upgrading override
	// files. For now, just log that we're ignoring the file.
	for _, path := range overrides {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"Override configuration file ignored",
			fmt.Sprintf(
				skippedConfigurationFileWarning,
				"override",
				path,
				"override",
			),
		))
	}

	// Check Terraform required_version constraints
	for _, file := range files {
		for _, constraint := range file.CoreVersionConstraints {
			if !constraint.Required.Check(tfversion.SemVer) {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Unsupported Terraform Core version",
					Detail: fmt.Sprintf(
						"This configuration does not support Terraform version %s. To proceed, either choose another supported Terraform version or update this version constraint. Version constraints are normally set for good reason, so updating the constraint may lead to other errors or unexpected behavior.",
						tfversion.String(),
					),
					Subject: &constraint.DeclRange,
				})
			}
		}
	}
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Build up a list of required providers, uniquely by local name
	requiredProviders := make(map[string]*configs.RequiredProvider)
	rewritePaths := make(map[string]bool)
	allProviderConstraints := make(map[string]getproviders.VersionConstraints)

	// Step 1: copy all explicit provider requirements across
	for path, file := range files {
		for _, rps := range file.RequiredProviders {
			rewritePaths[path] = true
			for _, rp := range rps.RequiredProviders {
				if previous, exist := requiredProviders[rp.Name]; exist {
					diags = diags.Append(&hcl.Diagnostic{
						Summary:  "Duplicate required provider configuration",
						Detail:   fmt.Sprintf("Found duplicate required provider configuration for %q.Previously configured at %s", rp.Name, previous.DeclRange),
						Severity: hcl.DiagWarning,
						Context:  rps.DeclRange.Ptr(),
						Subject:  rp.DeclRange.Ptr(),
					})
				} else {
					// We're copying the struct here to ensure that any
					// mutation does not affect the original, if we rewrite
					// this file
					requiredProviders[rp.Name] = &configs.RequiredProvider{
						Name:        rp.Name,
						Source:      rp.Source,
						Type:        rp.Type,
						Requirement: rp.Requirement,
						DeclRange:   rp.DeclRange,
					}

					// Parse and store version constraints for later use when
					// processing the provider redirect
					constraints, err := getproviders.ParseVersionConstraints(rp.Requirement.Required.String())
					if err != nil {
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Invalid version constraint",
							// The errors returned by ParseVersionConstraint
							// already include the section of input that was
							// incorrect, so we don't need to
							// include that here.
							Detail:  fmt.Sprintf("Incorrect version constraint syntax: %s.", err.Error()),
							Subject: rp.Requirement.DeclRange.Ptr(),
						})
					} else {
						allProviderConstraints[rp.Name] = append(allProviderConstraints[rp.Name], constraints...)
					}
				}
			}
		}
	}

	for _, file := range files {
		// Step 2: add missing provider requirements from provider blocks
		for _, p := range file.ProviderConfigs {
			// Skip internal providers
			if p.Name == "terraform" {
				continue
			}

			// If no explicit provider configuration exists for the
			// provider configuration's local name, add one with a legacy
			// provider address.
			if _, exist := requiredProviders[p.Name]; !exist {
				requiredProviders[p.Name] = &configs.RequiredProvider{
					Name: p.Name,
				}
			}
			// Parse and store version constraints for later use when
			// processing the provider redirect
			constraints, err := getproviders.ParseVersionConstraints(p.Version.Required.String())
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid version constraint",
					// The errors returned by ParseVersionConstraint
					// already include the section of input that was
					// incorrect, so we don't need to
					// include that here.
					Detail:  fmt.Sprintf("Incorrect version constraint syntax: %s.", err.Error()),
					Subject: p.Version.DeclRange.Ptr(),
				})
			} else {
				allProviderConstraints[p.Name] = append(allProviderConstraints[p.Name], constraints...)
			}
		}

		// Step 3: add missing provider requirements from resources
		resources := [][]*configs.Resource{file.ManagedResources, file.DataResources}
		for _, rs := range resources {
			for _, r := range rs {
				// Find the appropriate provider local name for this resource
				var localName string

				// If there's a provider config, use that to determine the
				// local name. Otherwise use the implied provider local name
				// based on the resource's address.
				if r.ProviderConfigRef != nil {
					localName = r.ProviderConfigRef.Name
				} else {
					localName = r.Addr().ImpliedProvider()
				}

				// Skip internal providers
				if localName == "terraform" {
					continue
				}

				// If no explicit provider configuration exists for this local
				// name, add one with a legacy provider address.
				if _, exist := requiredProviders[localName]; !exist {
					requiredProviders[localName] = &configs.RequiredProvider{
						Name: localName,
					}
				}
			}
		}
	}

	// We should now have a complete understanding of the provider requirements
	// stated in the config.  If there are any providers, attempt to detect
	// their sources, and rewrite the config.
	if len(requiredProviders) > 0 {
		detectDiags := c.detectProviderSources(requiredProviders, allProviderConstraints)
		diags = diags.Append(detectDiags)
		if diags.HasErrors() {
			c.Ui.Error("Unable to detect sources for providers")
			c.showDiagnostics(diags)
			return 1
		}

		// Default output filename is "versions.tf", which is also where the
		// 0.12upgrade command added the required_version constraint.
		filename := path.Join(dir, "versions.tf")

		// Special case: if we only have one file with a required providers
		// block, output to that file instead.
		if len(rewritePaths) == 1 {
			for path := range rewritePaths {
				filename = path
				break
			}
		}

		// Remove the output file from the list of paths we want to rewrite
		// later. Otherwise we'd delete the required providers block after
		// writing it.
		delete(rewritePaths, filename)

		// Open or create the output file
		out, openDiags := c.openOrCreateFile(filename)
		diags = diags.Append(openDiags)

		if diags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}

		// Find all required_providers blocks, and store them alongside a map
		// back to the parent terraform block.
		var requiredProviderBlocks []*hclwrite.Block
		parentBlocks := make(map[*hclwrite.Block]*hclwrite.Block)
		root := out.Body()
		for _, rootBlock := range root.Blocks() {
			if rootBlock.Type() != "terraform" {
				continue
			}
			for _, childBlock := range rootBlock.Body().Blocks() {
				if childBlock.Type() == "required_providers" {
					requiredProviderBlocks = append(requiredProviderBlocks, childBlock)
					parentBlocks[childBlock] = rootBlock
				}
			}
		}

		// First required provider block, and the rest found in this file.
		var first *hclwrite.Block
		var rest []*hclwrite.Block

		// First terraform block in the first file. Declared at this scope so
		// that it can be used to write the version constraint later, if this
		// is the "versions.tf" file.
		var tfBlock *hclwrite.Block

		if len(requiredProviderBlocks) > 0 {
			// If we already have one or more required provider blocks, we'll rewrite
			// the first one, and remove the rest.
			first, rest = requiredProviderBlocks[0], requiredProviderBlocks[1:]

			// Set the terraform block here for later use to update the
			// required version constraint.
			tfBlock = parentBlocks[first]
		} else {
			// Otherwise, find or a create a terraform block, and add a new
			// empty required providers block to it.
			for _, rootBlock := range root.Blocks() {
				if rootBlock.Type() == "terraform" {
					tfBlock = rootBlock
					break
				}
			}
			if tfBlock == nil {
				tfBlock = root.AppendNewBlock("terraform", nil)
			}
			first = tfBlock.Body().AppendNewBlock("required_providers", nil)
		}

		// Find the body of the first block to prepare for rewriting it
		body := first.Body()

		// Build a sorted list of provider local names, for consistent ordering
		var localNames []string
		for localName := range requiredProviders {
			localNames = append(localNames, localName)
		}
		sort.Strings(localNames)

		// Populate the required providers block
		for _, localName := range localNames {
			requiredProvider := requiredProviders[localName]
			var attributes = make(map[string]cty.Value)

			if !requiredProvider.Type.IsZero() {
				attributes["source"] = cty.StringVal(requiredProvider.Type.ForDisplay())
			}

			if version := requiredProvider.Requirement.Required.String(); version != "" {
				attributes["version"] = cty.StringVal(version)
			}

			var attributesObject cty.Value
			if len(attributes) > 0 {
				attributesObject = cty.ObjectVal(attributes)
			} else {
				attributesObject = cty.EmptyObjectVal
			}
			// If this block already has an entry for this local name, we only
			// want to replace it if it's semantically different
			if existing := body.GetAttribute(localName); existing != nil {
				bytes := existing.Expr().BuildTokens(nil).Bytes()
				expr, _ := hclsyntax.ParseExpression(bytes, "", hcl.InitialPos)
				value, _ := expr.Value(nil)
				if !attributesObject.RawEquals(value) {
					body.SetAttributeValue(localName, attributesObject)
				}
			} else {
				body.SetAttributeValue(localName, attributesObject)
			}

			// If we don't have a source attribute, manually construct a commented
			// block explaining what to do
			if _, hasSource := attributes["source"]; !hasSource {
				// Generate the token stream for the required provider
				rp := body.GetAttribute(localName)
				expr := rp.Expr().BuildTokens(nil)

				// Partition the tokens into before and after the opening brace
				before, after := c.partitionTokensAfter(expr, hclsyntax.TokenOBrace)

				// If the value is an empty object, add a newline between the
				// braces so that the comment is not on the same line as either
				// brace.
				if len(before) == 1 && len(after) == 1 {
					newline := &hclwrite.Token{
						Type:  hclsyntax.TokenNewline,
						Bytes: []byte{'\n'},
					}
					after = append(hclwrite.Tokens{newline}, after...)
				}

				// Generate the comment and insert it at the start of the object
				comment := noSourceDetectedComment(localName)
				commentedBlock := append(before, comment...)
				commentedBlock = append(commentedBlock, after...)

				// Set the required provider object to this raw token stream
				body.SetAttributeRaw(localName, commentedBlock)
			}
		}

		// If this is the "versions.tf" file, add a new version constraint to
		// the first terraform block. If this isn't the "versions.tf" file,
		// we'll update that file separately.
		versionsFilename := path.Join(dir, "versions.tf")
		if filename == versionsFilename {
			tfBlock.Body().SetAttributeValue("required_version", cty.StringVal(">= 0.13"))
		}

		// Remove the rest of the blocks (and the parent block, if it's empty)
		for _, rpBlock := range rest {
			tfBlock := parentBlocks[rpBlock]
			tfBody := tfBlock.Body()
			tfBody.RemoveBlock(rpBlock)

			// If the terraform block has no blocks and no attributes, it's
			// basically empty (aside from comments and whitespace), so it's
			// more useful to remove it than leave it in.
			if len(tfBody.Blocks()) == 0 && len(tfBody.Attributes()) == 0 {
				root.RemoveBlock(tfBlock)
			}
		}

		// Write the config back to the file
		writeDiags := c.writeFile(out, filename)
		diags = diags.Append(writeDiags)
		if diags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}

		// If the file we just updated was not a "versions.tf" file, add or
		// update that file to set the required version constraint in the first
		// terraform block.
		if filename != versionsFilename {
			file, openDiags := c.openOrCreateFile(versionsFilename)
			diags = diags.Append(openDiags)

			if diags.HasErrors() {
				c.showDiagnostics(diags)
				return 1
			}

			// Find or create a terraform block
			root := file.Body()
			var tfBlock *hclwrite.Block
			for _, rootBlock := range root.Blocks() {
				if rootBlock.Type() == "terraform" {
					tfBlock = rootBlock
					break
				}
			}
			if tfBlock == nil {
				tfBlock = root.AppendNewBlock("terraform", nil)
			}

			// Set the required version attribute
			tfBlock.Body().SetAttributeValue("required_version", cty.StringVal(">= 0.13"))

			// Write the config back to the file
			writeDiags := c.writeFile(file, versionsFilename)
			diags = diags.Append(writeDiags)
			if diags.HasErrors() {
				c.showDiagnostics(diags)
				return 1
			}
		}

		// After successfully writing the new configuration, remove all other
		// required provider blocks from remaining configuration files.
		for path := range rewritePaths {
			// Read and parse the existing file
			config, err := ioutil.ReadFile(path)
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Unable to read configuration file",
					fmt.Sprintf("Error when reading configuration file %q: %s", filename, err),
				))
				c.showDiagnostics(diags)
				return 1
			}
			file, parseDiags := hclwrite.ParseConfig(config, filename, hcl.InitialPos)
			diags = diags.Append(parseDiags)
			if diags.HasErrors() {
				c.showDiagnostics(diags)
				return 1
			}

			// Find and remove all terraform.required_providers blocks
			root := file.Body()
			for _, rootBlock := range root.Blocks() {
				if rootBlock.Type() != "terraform" {
					continue
				}
				tfBody := rootBlock.Body()
				for _, childBlock := range tfBody.Blocks() {
					if childBlock.Type() == "required_providers" {
						rootBlock.Body().RemoveBlock(childBlock)

						// If the terraform block is now empty, remove it
						if len(tfBody.Blocks()) == 0 && len(tfBody.Attributes()) == 0 {
							root.RemoveBlock(rootBlock)
						}
					}
				}
			}

			// Write the config back to the file
			writeDiags := c.writeFile(file, path)
			diags = diags.Append(writeDiags)
			if diags.HasErrors() {
				c.showDiagnostics(diags)
				return 1
			}
		}
	}

	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 1
	}

	if len(diags) != 0 {
		c.Ui.Output(`-----------------------------------------------------------------------------`)
	}
	c.Ui.Output(c.Colorize().Color(`
[bold][green]Upgrade complete![reset]

Use your version control system to review the proposed changes, make any
necessary adjustments, and then commit.
`))

	return 0
}

// For providers which need a source attribute, detect the source
func (c *ZeroThirteenUpgradeCommand) detectProviderSources(requiredProviders map[string]*configs.RequiredProvider, allProviderConstraints map[string]getproviders.VersionConstraints) tfdiags.Diagnostics {
	source := c.providerInstallSource()
	var diags tfdiags.Diagnostics

providers:
	for name, rp := range requiredProviders {
		// If there's already an explicit source, skip it
		if rp.Source != "" {
			continue
		}

		// Construct a legacy provider FQN using the required provider local
		// name.  This ignores any auto-generated provider FQN from the load &
		// parse process, because we know that without an explicit source it is
		// not explicitly specified.
		addr := addrs.NewLegacyProvider(name)
		p, moved, err := getproviders.LookupLegacyProvider(addr, source)
		if err == nil {
			rp.Type = p

			if !moved.IsZero() {
				constraints, ok := allProviderConstraints[name]
				// If there's no version constraint, always use the redirect
				// target as there should be at least one version we can
				// install
				if !ok {
					rp.Type = moved
					continue providers
				}

				// Check that the redirect target has a version meeting our
				// constraints
				acceptable := versions.MeetingConstraints(constraints)
				available, _, err := source.AvailableVersions(moved)
				// If something goes wrong with the registry lookup here, fall
				// back to the non-redirect provider
				if err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Warning,
						"Failed to query available provider packages",
						fmt.Sprintf("Could not retrieve the list of available versions for provider %s: %s",
							moved.ForDisplay(), err),
					))
					continue providers
				}

				// Walk backwards to consider newer versions first
				for i := len(available) - 1; i >= 0; i-- {
					if acceptable.Has(available[i]) {
						// Success! Provider redirect target has a version
						// meeting our constraints, so we can use it
						rp.Type = moved
						continue providers
					}
				}

				// Find the last version available at the old location
				oldAvailable, _, err := source.AvailableVersions(p)
				if err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Warning,
						"Failed to query available provider packages",
						fmt.Sprintf("Could not retrieve the list of available versions for provider %s: %s",
							p.ForDisplay(), err),
					))
					continue providers
				}
				lastAvailable := oldAvailable[len(oldAvailable)-1]

				// If we fall through here, no versions at the target meet our
				// version constraints, so warn the user
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					"Provider has moved",
					fmt.Sprintf(
						"Provider %q has moved to %q. No action is required to continue using %q (%s), but if you want to upgrade beyond version %s, you must also update the source.",
						moved.Type, moved.ForDisplay(), p.ForDisplay(),
						getproviders.VersionConstraintsString(constraints), lastAvailable),
				))
			}
		} else {
			// Setting the provider address to a zero value struct
			// indicates that there is no known FQN for this provider,
			// which will cause us to write an explanatory comment in the
			// HCL output advising the user what to do about this.
			rp.Type = addrs.Provider{}

			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Warning,
				"Could not detect provider source",
				fmt.Sprintf("Error looking up provider source for %q: %s", name, err),
			))
		}
	}

	return diags
}

func (c *ZeroThirteenUpgradeCommand) openOrCreateFile(filename string) (*hclwrite.File, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// If the file doesn't exist, create a new empty file
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return hclwrite.NewEmptyFile(), diags
	} else if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unable to read configuration file",
			fmt.Sprintf("Error when reading configuration file %q: %s", filename, err),
		))
		return nil, diags
	} else {
		// File already exists, so load and parse it
		config, err := ioutil.ReadFile(filename)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Unable to read configuration file",
				fmt.Sprintf("Error when reading configuration file %q: %s", filename, err),
			))
			return nil, diags
		}
		file, parseDiags := hclwrite.ParseConfig(config, filename, hcl.InitialPos)
		diags = diags.Append(parseDiags)
		return file, diags
	}
}

func (c *ZeroThirteenUpgradeCommand) writeFile(file *hclwrite.File, filename string) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	f, err := os.OpenFile(filename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unable to open configuration file for writing",
			fmt.Sprintf("Error when reading configuration file %q: %s", filename, err),
		))
		return diags
	}
	_, err = file.WriteTo(f)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unable to rewrite configuration file",
			fmt.Sprintf("Error when rewriting configuration file %q: %s", filename, err),
		))
		return diags
	}
	return diags
}

// Take a list of tokens and a separator token, and return two lists: one up to
// and including the first instance of the separator, and the rest of the
// tokens. If the separator is not present, return the entire list in the first
// return value.
func (c *ZeroThirteenUpgradeCommand) partitionTokensAfter(tokens hclwrite.Tokens, separator hclsyntax.TokenType) (hclwrite.Tokens, hclwrite.Tokens) {
	for i := 0; i < len(tokens); i++ {
		if tokens[i].Type == separator {
			return tokens[0 : i+1], tokens[i+1:]
		}
	}

	return tokens, nil
}

// Generate a list of tokens for a comment explaining that a provider source
// could not be detected.
func noSourceDetectedComment(name string) hclwrite.Tokens {
	comment := fmt.Sprintf(`# TF-UPGRADE-TODO
#
# No source detected for this provider. You must add a source address
# in the following format:
#
# source = "your-registry.example.com/organization/%s"
#
# For more information, see the provider source documentation:
#
# https://www.terraform.io/docs/configuration/providers.html#provider-source`, name)

	var tokens hclwrite.Tokens
	for _, line := range strings.Split(comment, "\n") {
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenComment, Bytes: []byte(line)})
	}
	return tokens
}

func (c *ZeroThirteenUpgradeCommand) Help() string {
	helpText := `
Usage: terraform 0.13upgrade [options] [module-dir]

  Updates module configuration files to add provider source attributes and
  merge multiple required_providers blocks into one.

  By default, 0.13upgrade rewrites the files in the current working directory.
  However, a path to a different directory can be provided. The command will
  prompt for confirmation interactively unless the -yes option is given.

Options:

  -yes        Skip the initial introduction messages and interactive
              confirmation. This can be used to run this command in
              batch from a script.
`
	return strings.TrimSpace(helpText)
}

func (c *ZeroThirteenUpgradeCommand) Synopsis() string {
	return "Rewrites pre-0.13 module source code for v0.13"
}
