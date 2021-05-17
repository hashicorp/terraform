package cliconfig

import (
	"fmt"
	"path/filepath"

	"github.com/hashicorp/hcl"
	hclast "github.com/hashicorp/hcl/hcl/ast"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ProviderInstallation is the structure of the "provider_installation"
// nested block within the CLI configuration.
type ProviderInstallation struct {
	Methods []*ProviderInstallationMethod

	// DevOverrides allows overriding the normal selection process for
	// a particular subset of providers to force using a particular
	// local directory and disregard version numbering altogether.
	// This is here to allow provider developers to conveniently test
	// local builds of their plugins in a development environment, without
	// having to fuss with version constraints, dependency lock files, and
	// so forth.
	//
	// This is _not_ intended for "production" use because it bypasses the
	// usual version selection and checksum verification mechanisms for
	// the providers in question. To make that intent/effect clearer, some
	// Terraform commands emit warnings when overrides are present. Local
	// mirror directories are a better way to distribute "released"
	// providers, because they are still subject to version constraints and
	// checksum verification.
	DevOverrides map[addrs.Provider]getproviders.PackageLocalDir
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
		devOverrides := make(map[addrs.Provider]getproviders.PackageLocalDir)

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
			case "dev_overrides":
				if len(pi.Methods) > 0 {
					// We require dev_overrides to appear first if it's present,
					// because dev_overrides effectively bypass the normal
					// selection process for a particular provider altogether,
					// and so they don't participate in the usual
					// include/exclude arguments and priority ordering.
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid provider_installation method block",
						fmt.Sprintf("The dev_overrides block at at %s must appear before all other installation methods, because development overrides always have the highest priority.", methodBlock.Pos()),
					))
					continue
				}

				// The content of a dev_overrides block is a mapping from
				// provider source addresses to local filesystem paths. To get
				// our decoding started, we'll use the normal HCL decoder to
				// populate a map of strings and then decode further from
				// that.
				var rawItems map[string]string
				err := hcl.DecodeObject(&rawItems, methodBody)
				if err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid provider_installation method block",
						fmt.Sprintf("Invalid %s block at %s: %s.", methodTypeStr, block.Pos(), err),
					))
					continue
				}

				for rawAddr, rawPath := range rawItems {
					addr, moreDiags := addrs.ParseProviderSourceString(rawAddr)
					if moreDiags.HasErrors() {
						diags = diags.Append(tfdiags.Sourceless(
							tfdiags.Error,
							"Invalid provider installation dev overrides",
							fmt.Sprintf("The entry %q in %s is not a valid provider source string.", rawAddr, block.Pos()),
						))
						continue
					}
					dirPath := filepath.Clean(rawPath)
					devOverrides[addr] = getproviders.PackageLocalDir(dirPath)
				}

				continue // We won't add anything to pi.Methods for this one

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

		if len(devOverrides) > 0 {
			pi.DevOverrides = devOverrides
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
