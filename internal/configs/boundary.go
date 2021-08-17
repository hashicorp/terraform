package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

type Boundary struct {
	Config     hcl.Body
	Connection hcl.Body

	Schema *configschema.Block

	DeclRange hcl.Range
}

func decodeBoundaryBlock(block *hcl.Block) (map[string]*Boundary, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	res := map[string]*Boundary{}

	schema := &hcl.BodySchema{}
	for name := range boundaryBlockSchema.Attributes {
		schema.Attributes = append(schema.Attributes, hcl.AttributeSchema{Name: name})
	}
	for name := range boundaryBlockSchema.BlockTypes {
		schema.Blocks = append(schema.Blocks, hcl.BlockHeaderSchema{
			Type:       name,
			LabelNames: []string{"name"},
		})
	}
	content, moreDiags := block.Body.Content(schema)
	diags = append(diags, moreDiags...)

	schema = &hcl.BodySchema{}
	for name := range boundaryBlockSchema.BlockTypes["connection"].Attributes {
		schema.Attributes = append(schema.Attributes, hcl.AttributeSchema{Name: name})
	}

	if len(content.Blocks) == 0 {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "At least one connection must be defined in a Boundary block",
			Subject:  &block.DefRange,
		})
	}

	for _, conn := range content.Blocks {
		name := conn.Labels[0]
		if !hclsyntax.ValidIdentifier(name) {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Invalid connection name %q", name),
				Detail:   badIdentifierDetail,
				Subject:  &conn.DefRange,
			})
		}

		if _, found := res[name]; found {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Connection %q has been defined multiple times", name),
				Subject:  &conn.LabelRanges[0],
			})
		}

		_, moreDiags := conn.Body.Content(schema)
		diags = append(diags, moreDiags...)

		res[name] = &Boundary{
			Config:     block.Body,
			Connection: conn.Body,
			Schema:     boundaryBlockSchema,
			DeclRange:  block.DefRange,
		}
	}

	return res, diags
}

var boundaryBlockSchema = &configschema.Block{
	Attributes: map[string]*configschema.Attribute{
		"address":         {Type: cty.String},
		"ca_cert":         {Type: cty.String},
		"ca_path":         {Type: cty.String},
		"client_cert":     {Type: cty.String},
		"client_key":      {Type: cty.String},
		"tls_insecure":    {Type: cty.Bool},
		"tls_server_name": {Type: cty.String},
		"token":           {Type: cty.String},
		"keyring_type":    {Type: cty.String},
		"token_name":      {Type: cty.String},
	},
	BlockTypes: map[string]*configschema.NestedBlock{
		"connection": {
			Nesting: configschema.NestingMap,
			Block: configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"listen_addr":         {Type: cty.String},
					"listen_port":         {Type: cty.Number},
					"authorization_token": {Type: cty.String},
					"host_id":             {Type: cty.String},
					"target_id":           {Type: cty.String},
					"target_name":         {Type: cty.String},
					"target_scope_id":     {Type: cty.String},
					"target_scope_name":   {Type: cty.String},
				},
			},
		},
	},
}
