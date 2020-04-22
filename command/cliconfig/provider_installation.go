package cliconfig

import (
	"fmt"

	"github.com/hashicorp/hcl"
	hclast "github.com/hashicorp/hcl/hcl/ast"
	"github.com/hashicorp/terraform/tfdiags"
)

// ProviderInstallation is the structure of the "provider_installation"
// nested block within the CLI configuration.
type ProviderInstallation struct {
	Sources []*ProviderInstallationSource
}

// decodeProviderInstallationFromConfig uses the HCL AST API directly to
// decode "provider_installation" blocks from the given file.
//
// This uses the HCL AST directly, rather than HCL's decoder, because the
// intended configuration structure can't be represented using the HCL
// decoder's struct tags. This structure is intended as something that would
// be relatively easier to deal with in HCL 2 once we eventually migrate
// CLI config over to that, and so this function is stricter than HCL 1's
// decoder would be in terms of exactly what configuration shape it is
// expecting.
//
// Note that this function wants the top-level file object which might or
// might not contain provider_installation blocks, not a provider_installation
// block directly itself.
func decodeProviderInstallationFromConfig(hclFile *hclast.File) ([]*ProviderInstallation, tfdiags.Diagnostics) {
	var ret []*ProviderInstallation
	var diags tfdiags.Diagnostics

	root := hclFile.Node.(*hclast.ObjectList)

	// This is a rather odd hybrid: it's a HCL 2-like decode implemented using
	// the HCL 1 AST API. That makes it a bit awkward in places, but it allows
	// us to mimick the strictness of HCL 2 (making a later migration easier)
	// and to support a block structure that the HCL 1 decoder can't represent.
	for _, block := range root.Items {
		if block.Keys[0].Token.Value() != "provider_installation" {
			continue
		}
		if block.Assign.Line != 0 {
			// Seems to be an attribute rather than a block
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid provider_installation block",
				fmt.Sprintf("The provider_installation block at %s must not be introduced with an equals sign.", block.Pos()),
			))
			continue
		}
		if len(block.Keys) > 1 {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid provider_installation block",
				fmt.Sprintf("The provider_installation block at %s must not have any labels.", block.Pos()),
			))
		}

		pi := &ProviderInstallation{}

		// Because we checked block.Assign was unset above we can assume that
		// we're reading something produced with block syntax and therefore
		// it will always be an hclast.ObjectType.
		body := block.Val.(*hclast.ObjectType)

		for _, sourceBlock := range body.List.Items {
			if sourceBlock.Assign.Line != 0 {
				// Seems to be an attribute rather than a block
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid provider_installation source block",
					fmt.Sprintf("The items inside the provider_installation block at %s must all be blocks.", block.Pos()),
				))
				continue
			}
			if len(sourceBlock.Keys) > 1 {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid provider_installation source block",
					fmt.Sprintf("The blocks inside the provider_installation block at %s may not have any labels.", block.Pos()),
				))
			}

			sourceBody := sourceBlock.Val.(*hclast.ObjectType)

			sourceTypeStr := sourceBlock.Keys[0].Token.Value().(string)
			var location ProviderInstallationSourceLocation
			var include, exclude []string
			var extraArgs []string
			switch sourceTypeStr {
			case "direct":
				type BodyContent struct {
					Include []string `hcl:"include"`
					Exclude []string `hcl:"exclude"`
				}
				var bodyContent BodyContent
				err := hcl.DecodeObject(&bodyContent, sourceBody)
				if err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid provider_installation source block",
						fmt.Sprintf("Invalid %s block at %s: %s.", sourceTypeStr, block.Pos(), err),
					))
					continue
				}
				location = ProviderInstallationDirect
				include = bodyContent.Include
				exclude = bodyContent.Exclude
			case "filesystem_mirror":
				type BodyContent struct {
					Path    string   `hcl:"path"`
					Include []string `hcl:"include"`
					Exclude []string `hcl:"exclude"`
				}
				var bodyContent BodyContent
				err := hcl.DecodeObject(&bodyContent, sourceBody)
				if err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid provider_installation source block",
						fmt.Sprintf("Invalid %s block at %s: %s.", sourceTypeStr, block.Pos(), err),
					))
					continue
				}
				if bodyContent.Path == "" {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid provider_installation source block",
						fmt.Sprintf("Invalid %s block at %s: \"path\" argument is required.", sourceTypeStr, block.Pos()),
					))
					continue
				}
				location = ProviderInstallationFilesystemMirror(bodyContent.Path)
				include = bodyContent.Include
				exclude = bodyContent.Exclude
			case "network_mirror":
				type BodyContent struct {
					URL     string   `hcl:"url"`
					Include []string `hcl:"include"`
					Exclude []string `hcl:"exclude"`
				}
				var bodyContent BodyContent
				err := hcl.DecodeObject(&bodyContent, sourceBody)
				if err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid provider_installation source block",
						fmt.Sprintf("Invalid %s block at %s: %s.", sourceTypeStr, block.Pos(), err),
					))
					continue
				}
				if bodyContent.URL == "" {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid provider_installation source block",
						fmt.Sprintf("Invalid %s block at %s: \"url\" argument is required.", sourceTypeStr, block.Pos()),
					))
					continue
				}
				location = ProviderInstallationNetworkMirror(bodyContent.URL)
				include = bodyContent.Include
				exclude = bodyContent.Exclude
			default:
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid provider_installation source block",
					fmt.Sprintf("Unknown provider installation source type %q at %s.", sourceTypeStr, sourceBlock.Pos()),
				))
				continue
			}

			for _, argName := range extraArgs {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid provider_installation source block",
					fmt.Sprintf("Invalid %s block at %s: this source type does not expect the argument %q.", sourceTypeStr, block.Pos(), argName),
				))
			}

			pi.Sources = append(pi.Sources, &ProviderInstallationSource{
				Location: location,
				Include:  include,
				Exclude:  exclude,
			})
		}

		ret = append(ret, pi)
	}

	return ret, diags
}

// ProviderInstallationSource represents an installation source block inside
// a provider_installation block.
type ProviderInstallationSource struct {
	Location ProviderInstallationSourceLocation
	Include  []string `hcl:"include"`
	Exclude  []string `hcl:"exclude"`
}

// ProviderInstallationSourceLocation is an interface type representing the
// different installation source types. The concrete implementations of
// this interface are:
//
//     ProviderInstallationDirect:                install from the provider's origin registry
//     ProviderInstallationFilesystemMirror(dir): install from a local filesystem mirror
//     ProviderInstallationNetworkMirror(host):   install from a network mirror
type ProviderInstallationSourceLocation interface {
	providerInstallationLocation()
}

type configProviderInstallationDirect [0]byte

func (i configProviderInstallationDirect) providerInstallationLocation() {}

// ProviderInstallationDirect is a ProviderInstallationSourceLocation
// representing installation from a provider's origin registry.
var ProviderInstallationDirect ProviderInstallationSourceLocation = configProviderInstallationDirect{}

// ProviderInstallationFilesystemMirror is a ProviderInstallationSourceLocation
// representing installation from a particular local filesystem mirror. The
// string value is the filesystem path to the mirror directory.
type ProviderInstallationFilesystemMirror string

func (i ProviderInstallationFilesystemMirror) providerInstallationLocation() {}

// ProviderInstallationNetworkMirror is a ProviderInstallationSourceLocation
// representing installation from a particular local network mirror. The
// string value is the HTTP base URL exactly as written in the configuration,
// without any normalization.
type ProviderInstallationNetworkMirror string

func (i ProviderInstallationNetworkMirror) providerInstallationLocation() {}
