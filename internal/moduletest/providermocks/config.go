package providermocks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/spf13/afero"
	"github.com/zclconf/go-cty/cty"
)

// Config represents the configuration for how to respond to requests for
// a particular provider.
type Config struct {
	ForProvider addrs.Provider
	BaseDir     string

	ResourceTypes map[ResourceType]*ResourceTypeConfig

	Sources map[string][]byte
}

type ResourceTypeConfig struct {
	ForType ResourceType

	// Responses describes a series of possible responses to various different
	// kinds of provider request.
	//
	// A mock provider will try each configured response in order and select
	// the first one whose condition returns true. The content of the selected
	// response will then be merged into the request object to produce the
	// full object to respond with. The exact details of how merging works
	// vary depending on the operation, because each operation has some
	// different constraints on exactly what changes a provider is allowed to
	// make compared to the objects given as input.
	//
	// For a data resource type only readRequest is used. Managed resource types
	// can use all three of readRequest, planRequest, and applyRequest.
	Responses map[requestType][]*ResponseConfig
}

type ResponseConfig struct {
	Name      string
	Condition hcl.Expression
	Content   hcl.Body

	DeclRange hcl.Range
}

const mockFilenameSuffix string = ".tfmock"

// LoadMockConfig decodes a mock configuration for the given provider from
// the given base directory.
//
// This function also performs broad validation of the overall structure of
// the mock provider configuration, but it does not have access to the
// schema of the provider and so it cannot verify that the declared resource
// types are valid or that the mock responses conform to the resource type
// schemas.
func LoadMockConfig(provider addrs.Provider, baseDir string) (*Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &Config{
		ForProvider:   provider,
		ResourceTypes: make(map[ResourceType]*ResourceTypeConfig),
	}

	// We'll normalize the given base directory just because it helps make
	// our reported source locations easier to read.
	baseDir = filepath.Clean(baseDir)
	ret.BaseDir = baseDir

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Cannot read mock provider configuration",
			fmt.Sprintf("Failed to list the configuration files in %s: %s.", baseDir, err),
		))
		return ret, diags
	}

	parser := configs.NewParser(afero.NewOsFs())

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, mockFilenameSuffix) {
			continue
		}
		name = name[:len(name)-len(mockFilenameSuffix)]
		dot := strings.IndexByte(name, '.')
		if dot == -1 {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid mock resource type configuration file",
				fmt.Sprintf("Cannot use %q as a mock resource type configuration: must have either prefix \"resource.\" or \"data.\" to specify the intended resource mode.", entry.Name()),
			))
			continue
		}
		modeName := name[:dot]
		typeName := name[dot+1:]

		var mode addrs.ResourceMode
		switch modeName {
		case "resource":
			mode = addrs.ManagedResourceMode
		case "data":
			mode = addrs.DataResourceMode
		default:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid mock resource type configuration file",
				fmt.Sprintf("Cannot use %q as a mock resource type configuration: unrecognized resource mode prefix %q.", entry.Name(), modeName),
			))
			continue
		}

		if !hclsyntax.ValidIdentifier(typeName) {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid mock resource type name",
				fmt.Sprintf("Cannot use %q as a mock resource type name: must be a valid identifier.", typeName),
			))
		}

		resourceType := ResourceType{
			Mode: mode,
			Type: typeName,
		}
		rtc, moreDiags := loadResourceTypeConfig(resourceType, filepath.Join(baseDir, entry.Name()), parser)
		diags = diags.Append(moreDiags)
		if diags.HasErrors() {
			continue
		}
		ret.ResourceTypes[resourceType] = rtc
	}

	ret.Sources = parser.Sources()

	return ret, diags
}

func loadResourceTypeConfig(rTy ResourceType, filename string, parser *configs.Parser) (*ResourceTypeConfig, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &ResourceTypeConfig{
		ForType:   rTy,
		Responses: make(map[requestType][]*ResponseConfig),
	}

	body, hclDiags := parser.LoadHCLFile(filename)
	diags = diags.Append(hclDiags)
	if body == nil {
		// Nothing else we can do, then.
		return ret, diags
	}

	content, hclDiags := body.Content(resourceTypeRootSchema)
	diags = diags.Append(hclDiags)

	reqTypeBlocks := make(map[requestType]*hcl.Block)

	for _, block := range content.Blocks {
		var reqType requestType
		switch block.Type {
		case "read":
			reqType = readRequest
		case "plan":
			reqType = planRequest
		case "apply":
			reqType = applyRequest
		default:
			panic(fmt.Sprintf("unexpected block type %q", block.Type))
		}

		if existing, exists := reqTypeBlocks[reqType]; exists {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Duplicate %s block", block.Type),
				Detail:   fmt.Sprintf("A set of %s responses was already declared at %s.", block.Type, existing.DefRange),
				Subject:  block.DefRange.Ptr(),
			})
			continue
		}
		reqTypeBlocks[reqType] = block

		if rTy.Mode == addrs.DataResourceMode && reqType != readRequest {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid request type",
				Detail:   "Data resource types only support \"read\" requests.",
				Subject:  block.TypeRange.Ptr(),
			})
		}

		resps, moreDiags := decodeResponseBlocks(block.Body)
		diags = diags.Append(moreDiags)
		ret.Responses[reqType] = resps
	}

	switch rTy.Mode {
	case addrs.ManagedResourceMode:
		if len(ret.Responses[applyRequest]) == 0 {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "No apply responses for managed resource type",
				Detail:   fmt.Sprintf("Resource type %s must have at least one mock response for apply requests.", rTy),
				Subject:  body.MissingItemRange().Ptr(),
			})
		}
	case addrs.DataResourceMode:
		if len(ret.Responses[readRequest]) == 0 {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "No read responses for managed resource type",
				Detail:   fmt.Sprintf("Resource type %s must have at least one mock response for read requests.", rTy),
				Subject:  body.MissingItemRange().Ptr(),
			})
		}
	}

	return ret, diags
}

func decodeResponseBlocks(body hcl.Body) ([]*ResponseConfig, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var ret []*ResponseConfig

	content, hclDiags := body.Content(resourceTypeResponsesListSchema)
	diags = diags.Append(hclDiags)

	ret = make([]*ResponseConfig, 0, len(content.Blocks))
	responses := make(map[string]*ResponseConfig, len(content.Blocks))

	for _, block := range content.Blocks {
		if block.Type != "response" {
			panic(fmt.Sprintf("unexpected block type %q", block.Type))
		}

		respConfig, moreDiags := decodeResponseBlock(block)
		diags = diags.Append(moreDiags)

		name := block.Labels[0]
		if existing, exists := responses[name]; exists {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Duplicate response block",
				Detail:   fmt.Sprintf("A response named %q was already declared at %s.", name, existing.DeclRange),
				Subject:  block.LabelRanges[0].Ptr(),
			})
		} else {
			responses[name] = respConfig
		}

		if !hclsyntax.ValidIdentifier(name) {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid mock response name",
				fmt.Sprintf("Cannot use %q as a mock response name: must be a valid identifier.", name),
			))
		}

		ret = append(ret, respConfig)
	}

	return ret, diags
}

func decodeResponseBlock(block *hcl.Block) (*ResponseConfig, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &ResponseConfig{
		Name:      block.Labels[0],
		DeclRange: block.DefRange,
	}

	content, hclDiags := block.Body.Content(resourceTypeResponseSchema)
	diags = diags.Append(hclDiags)

	if attr, ok := content.Attributes["condition"]; ok {
		ret.Condition = attr.Expr
	} else {
		ret.Condition = hcl.StaticExpr(cty.True, block.DefRange)
	}

	if len(content.Blocks) < 1 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Missing response content block",
			Detail:   "A response block must have a nested block of type content, defining attributes to merge into the request to produce the response.",
			Subject:  block.Body.MissingItemRange().Ptr(),
		})
		ret.Content = hcl.EmptyBody()
		return ret, diags
	}

	ret.Content = content.Blocks[0].Body

	if len(content.Blocks) > 1 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Duplicate response content block",
			Detail:   fmt.Sprintf("The content for this response was already defined at %s.", content.Blocks[0].DefRange),
			Subject:  block.LabelRanges[1].Ptr(),
		})
	}

	return ret, diags
}

var resourceTypeRootSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "read"},
		{Type: "plan"},
		{Type: "apply"},
	},
}

var resourceTypeResponsesListSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "response", LabelNames: []string{"name"}},
	},
}

var resourceTypeResponseSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "condition"},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "content"},
	},
}
