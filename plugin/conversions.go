package plugin

import (
	"encoding/json"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/plugin/proto"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// ProtoToProviderSchema takes a proto.Schema and converts it to a providers.Schema.
func ProtoToProviderSchema(s *proto.Schema) providers.Schema {
	return providers.Schema{
		Version: int(s.Version),
		Block:   schemaBlock(s.Block),
	}
}

// schemaBlock takes the GetSchcema_Block from a grpc response and converts it
// to a terraform *configschema.Block.
func schemaBlock(b *proto.Schema_Block) *configschema.Block {
	block := &configschema.Block{
		Attributes: make(map[string]*configschema.Attribute),
		BlockTypes: make(map[string]*configschema.NestedBlock),
	}

	for _, a := range b.Attributes {
		attr := &configschema.Attribute{
			Description: a.Description,
			Required:    a.Required,
			Optional:    a.Optional,
			Computed:    a.Computed,
			Sensitive:   a.Sensitive,
		}

		if err := json.Unmarshal(a.Type, &attr.Type); err != nil {
			panic(err)
		}

		block.Attributes[a.Name] = attr
	}

	for _, b := range b.BlockTypes {
		block.BlockTypes[b.TypeName] = schemaNestedBlock(b)
	}

	return block
}

func schemaNestedBlock(b *proto.Schema_NestedBlock) *configschema.NestedBlock {
	nb := &configschema.NestedBlock{
		Nesting:  configschema.NestingMode(b.Nesting),
		MinItems: int(b.MinItems),
		MaxItems: int(b.MaxItems),
	}

	nested := schemaBlock(b.Block)
	nb.Block = *nested
	return nb
}

// ProtoToDiagnostics converts a list of proto.Diagnostics to a tf.Diagnostics.
// for now we assume these only contain a basic message
func ProtoToDiagnostics(ds []*proto.Diagnostic) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	for _, d := range ds {
		var severity tfdiags.Severity

		switch d.Severity {
		case proto.Diagnostic_ERROR:
			severity = tfdiags.Error
		case proto.Diagnostic_WARNING:
			severity = tfdiags.Warning
		}

		var newDiag tfdiags.Diagnostic

		// if there's an attribute path, we need to create a AttributeValue diagnostic
		if d.Attribute != nil {
			path := attributePath(d.Attribute)
			newDiag = tfdiags.AttributeValue(severity, d.Summary, d.Detail, path)
		} else {
			newDiag = tfdiags.Sourceless(severity, d.Summary, d.Detail)
		}

		diags = diags.Append(newDiag)
	}

	return diags
}

// attributePath takes the proto encoded path and converts it to a cty.Path
func attributePath(ap *proto.AttributePath) cty.Path {
	var p cty.Path
	for _, step := range ap.Steps {
		switch selector := step.Selector.(type) {
		case *proto.AttributePath_Step_AttributeName:
			p = p.GetAttr(selector.AttributeName)
		case *proto.AttributePath_Step_ElementKeyString:
			p = p.Index(cty.StringVal(selector.ElementKeyString))
		case *proto.AttributePath_Step_ElementKeyInt:
			p = p.Index(cty.NumberIntVal(selector.ElementKeyInt))
		}
	}
	return p
}
