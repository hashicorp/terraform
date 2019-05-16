package command

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"

	hcl2syntax "github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/httpclient"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/hashicorp/terraform/svchost"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/version"
)

var pluginProtocol5Constraint = discovery.ConstraintStr("~> 5.0").MustParse()

var zeroTwelveReservedVariableNames = map[string]struct{}{
	"source":     {},
	"version":    {},
	"count":      {},
	"for_each":   {},
	"depends_on": {},
	"providers":  {},
	"lifecycle":  {},
	"locals":     {},
	"provider":   {},
}

// ZeroTwelveChecklistCommand is a Command implementation that checks whether
// a configuration is ready for upgrade to Terraform 0.12, producing a list
// of remaining preparation steps if not.
type ZeroTwelveChecklistCommand struct {
	Meta
}

func (c *ZeroTwelveChecklistCommand) Help() string {
	return zeroTwelveChecklistCommandHelp
}

func (c *ZeroTwelveChecklistCommand) Synopsis() string {
	return "Checks whether the configuration is ready for Terraform v0.12"
}

func (c *ZeroTwelveChecklistCommand) Run(args []string) int {
	c.Meta.process(args, false)

	cmdFlags := c.Meta.flagSet("0.12checklist")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	configPath, err := ModulePath(cmdFlags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Load the config
	root, diags := c.Module(configPath)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}
	if root == nil {
		c.Ui.Error(fmt.Sprintf(
			"No configuration files found in the directory: %s\n\n"+
				"This command requires configuration to run.",
			configPath))
		return 1
	}

	items := make(map[string][]string)
	hasItems := c.zeroTwelveChecklists(root, items)
	if !hasItems {
		fmt.Print(
			"Looks good! We did not detect any problems that ought to be\naddressed before upgrading to Terraform v0.12.\n\n" +
				"This tool is not perfect though, so please check the v0.12 upgrade\nguide for additional guidance, and for next steps:\n    https://www.terraform.io/upgrade-guides/0-12.html\n\n",
		)
		return 0
	}

	fmt.Print(
		"After analyzing this configuration and working directory, we have identified some necessary steps that we recommend you take before upgrading to Terraform v0.12:\n\n",
	)

	modKeys := make([]string, 0, len(items))
	for k := range items {
		modKeys = append(modKeys, k)
	}
	sort.Strings(modKeys)

	for _, k := range modKeys {
		modItems := items[k]
		sort.Strings(modItems)

		if k != "" {
			fmt.Printf("# Module `%q`\n\n", k)
		}

		for _, item := range modItems {
			fmt.Print("- [ ] ")
			sc := bufio.NewScanner(strings.NewReader(item))
			i := 0
			for sc.Scan() {
				if i == 0 {
					fmt.Printf("%s\n", sc.Text())
				} else {
					fmt.Printf("  %s\n", sc.Text())
				}
				i++
			}
			fmt.Printf("\n")
		}
	}

	fmt.Print(
		"Taking these steps before upgrading to Terraform v0.12 will simplify the upgrade process by avoiding syntax errors and other compatibility problems.\n\n",
	)

	return 1
}

func (c *ZeroTwelveChecklistCommand) zeroTwelveChecklists(mod *module.Tree, into map[string][]string) bool {
	key := strings.Join(mod.Path(), ".")
	items := c.zeroTwelveChecklistForModule(mod)
	hasItems := false

	if len(mod.Path()) == 0 { // It's the root module, then
		// We only report providers for the root module because they are
		// configuration-global and so this method already traverses the
		// whole tree itself.
		items = append(items, c.zeroTwelveChecklistForProviders(mod)...)
	}

	childMods := mod.Children()
	for _, modCall := range mod.Config().Modules {
		childMod, ok := childMods[modCall.Name]
		if !ok {
			// Should never happen.
			log.Printf("[WARN] Module %s declares child module %q but its tree node is missing", key, modCall.Name)
			continue
		}
		if !(strings.HasPrefix(modCall.Source, "./") || strings.HasPrefix(modCall.Source, "../")) {
			// For non-local modules we'll still run the checks but we'll roll
			// up into a single action item for our calling module if any
			// changes are needed, since the changes really need to be made
			// in the upstream repository.
			childItems := c.zeroTwelveChecklistForModule(childMod)
			if len(childItems) > 0 {
				items = append(items, fmt.Sprintf("Upgrade child module %q to a version that passes \"terraform 0.12checklist\".", strings.Join(childMod.Path(), ".")))
			}
			continue
		}

		childHasItems := c.zeroTwelveChecklists(childMod, into)
		if childHasItems {
			hasItems = true
		}
	}

	if len(items) > 0 {
		hasItems = true
	}
	into[key] = items
	return hasItems
}

func (c *ZeroTwelveChecklistCommand) zeroTwelveChecklistForModule(mod *module.Tree) []string {
	var items []string
	cfg := mod.Config()

	// Strings added to items must be Markdown-formatted. They can be multi-line
	// as long as all of the lines are valid to be nested inside a list item.
	// The caller above will eventually add the required initial indendation
	// to make the item's content appear as part of the item.
	//
	// In particular, items can include fenced code blocks and sub-lists.
	// However, it's best to keep Markdown metacharacters to a minimum so that
	// the result is also easy to read directly with human eyes, without
	// passing through a Markdown renderer.
	//
	// Each element of "items" will be rendered as a task list item using
	// GitHub's task list extension.

	for _, rc := range cfg.Resources {
		var blockType string
		switch rc.Mode {
		case config.ManagedResourceMode:
			blockType = "resource"
		case config.DataResourceMode:
			blockType = "data"
		default: // should never happen, because any other type would be a configuration loading error
			blockType = "???"
		}
		if !hcl2syntax.ValidIdentifier(rc.Name) {
			items = append(items, fmt.Sprintf(
				"`%s %q %q` has a name that is not a valid identifier.\n\n"+
					"In Terraform 0.12, resource names must start with a letter. To fix this, rename the resource in the configuration and then use `terraform state mv` to mirror that name change in the state.",
				blockType, rc.Type, rc.Name,
			))
		}
	}
	for _, pc := range cfg.ProviderConfigs {
		if pc.Alias == "" {
			continue
		}
		if !hcl2syntax.ValidIdentifier(pc.Alias) {
			items = append(items, fmt.Sprintf(
				"`provider %q` alias %q is not a valid identifier.\n\n"+
					"In Terraform 0.12, provider aliases must start with a letter. To fix this, rename the provider alias and any references to it in the configuration and then run `terraform apply` to re-attach any existing resources to the new alias name.",
				pc.Name, pc.Alias,
			))
		}
	}
	for _, vc := range cfg.Variables {
		if _, exists := zeroTwelveReservedVariableNames[vc.Name]; exists {
			items = append(items, fmt.Sprintf(
				"The variable name %q is now reserved.\n\n"+
					"Terraform 0.12 reserves certain variable names that have (or will have in future) a special meaning when used in a \"module\" block. The name of this variable must be changed before upgrading to v0.12. "+
					"This will unfortunately be a breaking change for any user of this module, requiring a major release to indicate that.\n"+
					"For more information on the reserved variable names, see the documentation at https://www.terraform.io/docs/configuration/variables.html .",
				vc.Name,
			))
		}
	}

	return items
}

func (c *ZeroTwelveChecklistCommand) zeroTwelveChecklistForProviders(root *module.Tree) []string {
	var items []string

	const registryHost = svchost.Hostname("registry.terraform.io")
	httpClient := httpclient.New()

	host, err := c.Services.Discover(registryHost)
	if err != nil {
		items = append(items, "Terraform couldn't reach the Terraform Registry (at `registry.terraform.io`) to determine whether current provider plugins are v0.12-compatible.\n\nIn general, we recommend upgrading to the latest version of each provider before upgrading to Terraform v0.12.")
		return items
	}
	baseURL, err := host.ServiceURL("providers.v1")
	if err != nil {
		items = append(items, "The Terraform Registry (at `registry.terraform.io`) does not seem to support the provider registry protocol (v1) that we depend on for provider compatibilty information. Perhaps an intermediate proxy is interfering with our requests, or this protocol version has become obsolete.\n\nIn general, we recommend upgrading to the latest version of each provider before upgrading to Terraform v0.12.")
		return items
	}

	// What we are looking for here is any installed plugin that could be
	// selected by this configuration but doesn't support Terraform v0.12.
	// We can't determine protocol support by interrogating the executable
	// directly, so instead we'll try to look it up via the Terraform Registry
	// (mimicking what Terraform v0.12 would do) and see what the registry
	// thinks is compatible.
	available := c.providerPluginSet()
	requirements := terraform.ModuleTreeDependencies(root, nil).AllPluginRequirements()
	candidates := available.ConstrainVersions(requirements)
	names := make([]string, 0, len(candidates))
	for name := range candidates {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		// We'll reach out to the registry now to see which versions are
		// available that support protocol version 5, so we can filter
		// our candidates further.
		versionsPath := path.Join("-", url.PathEscape(name), "versions")
		versionsURL := baseURL.String() + versionsPath

		req, err := http.NewRequest("GET", versionsURL, nil)
		if err != nil {
			// We control all of the input to NewRequest above, so this should never happen in practice.
			items = append(items, fmt.Sprintf("Failed to construct HTTP request to %s to discover what is available for provider %q: %s.", versionsURL, name, err))
			continue
		}
		req.Header.Set("X-Terraform-Version", version.String())

		// We assume no auth required here; in the unlikely event that the public
		// registry starts requiring auth in future, this tool is likely to be
		// obsolete.
		resp, err := httpClient.Do(req)
		if err != nil {
			items = append(items, fmt.Sprintf("Provider %q may need to be upgraded to a newer version that supports Terraform 0.12. (Request for supported version information failed: %s.)", name, err))
			continue
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			// OK
		case http.StatusNotFound:
			// Could happen if the provider is not one that HashiCorp distributes.
			items = append(items, fmt.Sprintf("Provider %q may need to be upgraded to a newer version that supports Terraform 0.12. (Supported version information is not available for this provider.)", name))
			continue
		default:
			items = append(items, fmt.Sprintf("Provider %q may need to be upgraded to a newer version that supports Terraform 0.12. (Request for supported version information failed with status %s.)", name, resp.Status))
			continue
		}

		type ResponseBody struct {
			Versions []struct {
				Version   discovery.VersionStr   `json:"version"`
				Protocols []discovery.VersionStr `json:"protocols"`
			} `json:"versions"`
		}
		var body ResponseBody
		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(&body); err != nil {
			items = append(items, fmt.Sprintf("Provider %q may need to be upgraded to a newer version that supports Terraform 0.12. (Request for supported version information returned an invalid response: %s.)", name, err))
			continue
		}

		have := candidates[name]
		supportedVersions := make(discovery.PluginMetaSet)
		compatible := false
	Versions:
		for _, raw := range body.Versions {
			proto5 := false
			for _, protoVerRaw := range raw.Protocols {
				protoVer, err := protoVerRaw.Parse()
				if err != nil {
					continue
				}
				if pluginProtocol5Constraint.Allows(protoVer) {
					proto5 = true
					break
				}
			}
			if !proto5 {
				continue
			}

			for meta := range have {
				if meta.Name == name && meta.Version == raw.Version {
					compatible = true
					break Versions
				}
			}
			supportedVersions.Add(discovery.PluginMeta{
				Name:    name,
				Version: raw.Version,
			})
		}

		if !compatible {
			if len(supportedVersions) == 0 {
				// If we get here then this seems to be a HashiCorp-distributed
				// provider (otherwise the registry would've returned 404 above)
				// but there isn't a v0.12-compatible release available for it.
				items = append(items, fmt.Sprintf(
					"Upgrade provider %q to a version that is compatible with Terraform 0.12.\n\n"+
						"No compatible version is available for automatic installation at this time. If this provider is still supported (not archived) then a compatible release should be available soon. For more information, check for 0.12 compatibility tasks in the provider's issue tracker.",
					name,
				))
				continue
			}

			newest := supportedVersions.Newest()
			items = append(items, fmt.Sprintf(
				"Upgrade provider %q to version %s or newer.\n\n"+
					"No currently-installed version is compatible with Terraform 0.12. To upgrade, set the version constraint for this provider as follows and then run `terraform init`:\n\n"+
					"    version = \"~> %s\"",
				name, newest.Version, newest.Version,
			))
		}
	}

	return items
}

const zeroTwelveChecklistCommandHelp = `
Usage: terraform 0.12checklist [dir]

  Analyzes a configuration and produces a list of any preparation steps
  required before upgrading to Terraform v0.12.

  For best results, run this command with no Terraform changes pending, so that
  it can analyze your infrastructure as currently deployed, rather than as
  currently planned.

  The resulting output uses Markdown formatting so you can easily copy it into
  a Markdown-capable issue tracker. We use GitHub-flavored Markdown.
`
