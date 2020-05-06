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
	Methods []*ProviderInstallationMethod
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
		// HCL only tracks whether the input was JSON or native syntax inside
		// individual tokens, so we'll use our block type token to decide
		// and assume that the rest of the block must be written in the same
		// syntax, because syntax is a whole-file idea.
		isJSON := block.Keys[0].Token.JSON
		if block.Assign.Line != 0 && !isJSON {
			// Seems to be an attribute rather than a block
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid provider_installation block",
				fmt.Sprintf("The provider_installation block at %s must not be introduced with an equals sign.", block.Pos()),
			))
			continue
		}
		if len(block.Keys) > 1 && !isJSON {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid provider_installation block",
				fmt.Sprintf("The provider_installation block at %s must not have any labels.", block.Pos()),
			))
		}

		pi := &ProviderInstallation{}

		body, ok := block.Val.(*hclast.ObjectType)
		if !ok {
			// We can't get in here with native HCL syntax because we
			// already checked above that we're using block syntax, but
			// if we're reading JSON then our value could potentially be
			// anything.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid provider_installation block",
				fmt.Sprintf("The provider_installation block at %s must not be introduced with an equals sign.", block.Pos()),
			))
			continue
		}

		for _, methodBlock := range body.List.Items {
			if methodBlock.Assign.Line != 0 && !isJSON {
				// Seems to be an attribute rather than a block
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid provider_installation method block",
					fmt.Sprintf("The items inside the provider_installation block at %s must all be blocks.", block.Pos()),
				))
				continue
			}
			if len(methodBlock.Keys) > 1 && !isJSON {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid provider_installation method block",
					fmt.Sprintf("The blocks inside the provider_installation block at %s may not have any labels.", block.Pos()),
				))
			}

			methodBody, ok := methodBlock.Val.(*hclast.ObjectType)
			if !ok {
				// We can't get in here with native HCL syntax because we
				// already checked above that we're using block syntax, but
				// if we're reading JSON then our value could potentially be
				// anything.
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid provider_installation method block",
					fmt.Sprintf("The items inside the provider_installation block at %s must all be blocks.", block.Pos()),
				))
				continue
			}

			methodTypeStr := methodBlock.Keys[0].Token.Value().(string)
			var location ProviderInstallationLocation
			var include, exclude []string
			switch methodTypeStr {
			case "direct":
				type BodyContent struct {
					Include []string `hcl:"include"`
					Exclude []string `hcl:"exclude"`
				}
				var bodyContent BodyContent
				err := hcl.DecodeObject(&bodyContent, methodBody)
				if err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid provider_installation method block",
						fmt.Sprintf("Invalid %s block at %s: %s.", methodTypeStr, block.Pos(), err),
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
				err := hcl.DecodeObject(&bodyContent, methodBody)
				if err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid provider_installation method block",
						fmt.Sprintf("Invalid %s block at %s: %s.", methodTypeStr, block.Pos(), err),
					))
					continue
				}
				if bodyContent.Path == "" {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid provider_installation method block",
						fmt.Sprintf("Invalid %s block at %s: \"path\" argument is required.", methodTypeStr, block.Pos()),
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
				err := hcl.DecodeObject(&bodyContent, methodBody)
				if err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid provider_installation method block",
						fmt.Sprintf("Invalid %s block at %s: %s.", methodTypeStr, block.Pos(), err),
					))
					continue
				}
				if bodyContent.URL == "" {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid provider_installation method block",
						fmt.Sprintf("Invalid %s block at %s: \"url\" argument is required.", methodTypeStr, block.Pos()),
					))
					continue
				}
				location = ProviderInstallationNetworkMirror(bodyContent.URL)
				include = bodyContent.Include
				exclude = bodyContent.Exclude
			default:
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid provider_installation method block",
					fmt.Sprintf("Unknown provider installation method %q at %s.", methodTypeStr, methodBlock.Pos()),
				))
				continue
			}

			pi.Methods = append(pi.Methods, &ProviderInstallationMethod{
				Location: location,
				Include:  include,
				Exclude:  exclude,
			})
		}

		ret = append(ret, pi)
	}

	return ret, diags
}

// ProviderInstallationMethod represents an installation method block inside
// a provider_installation block.
type ProviderInstallationMethod struct {
	Location ProviderInstallationLocation
	Include  []string `hcl:"include"`
	Exclude  []string `hcl:"exclude"`
}

// ProviderInstallationLocation is an interface type representing the
// different installation location types. The concrete implementations of
// this interface are:
//
//     ProviderInstallationDirect:                install from the provider's origin registry
//     ProviderInstallationFilesystemMirror(dir): install from a local filesystem mirror
//     ProviderInstallationNetworkMirror(host):   install from a network mirror
type ProviderInstallationLocation interface {
	providerInstallationLocation()
}

type providerInstallationDirect [0]byte

func (i providerInstallationDirect) providerInstallationLocation() {}

// ProviderInstallationDirect is a ProviderInstallationSourceLocation
// representing installation from a provider's origin registry.
var ProviderInstallationDirect ProviderInstallationLocation = providerInstallationDirect{}

func (i providerInstallationDirect) GoString() string {
	return "cliconfig.ProviderInstallationDirect"
}

// ProviderInstallationFilesystemMirror is a ProviderInstallationSourceLocation
// representing installation from a particular local filesystem mirror. The
// string value is the filesystem path to the mirror directory.
type ProviderInstallationFilesystemMirror string

func (i ProviderInstallationFilesystemMirror) providerInstallationLocation() {}

func (i ProviderInstallationFilesystemMirror) GoString() string {
	return fmt.Sprintf("cliconfig.ProviderInstallationFilesystemMirror(%q)", i)
}

// ProviderInstallationNetworkMirror is a ProviderInstallationSourceLocation
// representing installation from a particular local network mirror. The
// string value is the HTTP base URL exactly as written in the configuration,
// without any normalization.
type ProviderInstallationNetworkMirror string

func (i ProviderInstallationNetworkMirror) providerInstallationLocation() {}

func (i ProviderInstallationNetworkMirror) GoString() string {
	return fmt.Sprintf("cliconfig.ProviderInstallationNetworkMirror(%q)", i)
}
