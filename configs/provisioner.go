package configs

import (
	"fmt"

	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
)

// Provisioner represents a "provisioner" block when used within a
// "resource" block in a module or file.
type Provisioner struct {
	Type       string
	Config     hcl.Body
	Connection *Connection
	When       ProvisionerWhen
	OnFailure  ProvisionerOnFailure

	DeclRange hcl.Range
	TypeRange hcl.Range
}

func decodeProvisionerBlock(block *hcl.Block) (*Provisioner, hcl.Diagnostics) {
	pv := &Provisioner{
		Type:      block.Labels[0],
		TypeRange: block.LabelRanges[0],
		DeclRange: block.DefRange,
		When:      ProvisionerWhenCreate,
		OnFailure: ProvisionerOnFailureFail,
	}

	content, config, diags := block.Body.PartialContent(provisionerBlockSchema)
	pv.Config = config

	if attr, exists := content.Attributes["when"]; exists {
		switch hcl.ExprAsKeyword(attr.Expr) {
		case "create":
			pv.When = ProvisionerWhenCreate
		case "destroy":
			pv.When = ProvisionerWhenDestroy
		default:
			if exprIsNativeQuotedString(attr.Expr) {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid \"when\" keyword",
					Detail:   "The \"when\" argument keyword must not be given in quotes.",
					Subject:  attr.Expr.Range().Ptr(),
				})
			} else {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid \"when\" keyword",
					Detail:   "The \"when\" argument requires one of the following keywords: create or destroy.",
					Subject:  attr.Expr.Range().Ptr(),
				})
			}
		}
	}

	if attr, exists := content.Attributes["on_failure"]; exists {
		switch hcl.ExprAsKeyword(attr.Expr) {
		case "continue":
			pv.OnFailure = ProvisionerOnFailureContinue
		case "fail":
			pv.OnFailure = ProvisionerOnFailureFail
		default:
			if exprIsNativeQuotedString(attr.Expr) {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid \"on_failure\" keyword",
					Detail:   "The \"on_failure\" argument keyword must not be given in quotes.",
					Subject:  attr.Expr.Range().Ptr(),
				})
			} else {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid \"on_failure\" keyword",
					Detail:   "The \"on_failure\" argument requires one of the following keywords: continue or fail.",
					Subject:  attr.Expr.Range().Ptr(),
				})
			}
		}
	}

	var seenConnection *hcl.Block
	for _, block := range content.Blocks {
		switch block.Type {

		case "connection":
			if seenConnection != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate connection block",
					Detail:   fmt.Sprintf("This provisioner already has a connection block at %s.", seenConnection.DefRange),
					Subject:  &block.DefRange,
				})
				continue
			}
			seenConnection = block

			conn, connDiags := decodeConnectionBlock(block)
			diags = append(diags, connDiags...)
			pv.Connection = conn

		default:
			// Should never happen because there are no other block types
			// declared in our schema.
		}
	}

	return pv, diags
}

// Connection represents a "connection" block when used within either a
// "resource" or "provisioner" block in a module or file.
type Connection struct {
	Type   string
	Config hcl.Body

	DeclRange hcl.Range
	TypeRange *hcl.Range // nil if type is not set
}

func decodeConnectionBlock(block *hcl.Block) (*Connection, hcl.Diagnostics) {
	content, config, diags := block.Body.PartialContent(&hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{
				Name: "type",
			},
		},
	})

	conn := &Connection{
		Type:      "ssh",
		Config:    config,
		DeclRange: block.DefRange,
	}

	if attr, exists := content.Attributes["type"]; exists {
		valDiags := gohcl.DecodeExpression(attr.Expr, nil, &conn.Type)
		diags = append(diags, valDiags...)
		conn.TypeRange = attr.Expr.Range().Ptr()
	}

	return conn, diags
}

// ProvisionerWhen is an enum for valid values for when to run provisioners.
type ProvisionerWhen int

//go:generate stringer -type ProvisionerWhen

const (
	ProvisionerWhenInvalid ProvisionerWhen = iota
	ProvisionerWhenCreate
	ProvisionerWhenDestroy
)

// ProvisionerOnFailure is an enum for valid values for on_failure options
// for provisioners.
type ProvisionerOnFailure int

//go:generate stringer -type ProvisionerOnFailure

const (
	ProvisionerOnFailureInvalid ProvisionerOnFailure = iota
	ProvisionerOnFailureContinue
	ProvisionerOnFailureFail
)

var provisionerBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name: "when",
		},
		{
			Name: "on_failure",
		},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "connection",
		},
	},
}
