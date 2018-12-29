package configs

import (
	"fmt"

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
		expr, shimDiags := shimTraversalInString(attr.Expr, true)
		diags = append(diags, shimDiags...)

		switch hcl.ExprAsKeyword(expr) {
		case "create":
			pv.When = ProvisionerWhenCreate
		case "destroy":
			pv.When = ProvisionerWhenDestroy
		default:
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid \"when\" keyword",
				Detail:   "The \"when\" argument requires one of the following keywords: create or destroy.",
				Subject:  expr.Range().Ptr(),
			})
		}
	}

	if attr, exists := content.Attributes["on_failure"]; exists {
		expr, shimDiags := shimTraversalInString(attr.Expr, true)
		diags = append(diags, shimDiags...)

		switch hcl.ExprAsKeyword(expr) {
		case "continue":
			pv.OnFailure = ProvisionerOnFailureContinue
		case "fail":
			pv.OnFailure = ProvisionerOnFailureFail
		default:
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid \"on_failure\" keyword",
				Detail:   "The \"on_failure\" argument requires one of the following keywords: continue or fail.",
				Subject:  attr.Expr.Range().Ptr(),
			})
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

			//conn, connDiags := decodeConnectionBlock(block)
			//diags = append(diags, connDiags...)
			pv.Connection = &Connection{
				Config:    block.Body,
				DeclRange: block.DefRange,
			}

		default:
			// Any other block types are ones we've reserved for future use,
			// so they get a generic message.
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reserved block type name in provisioner block",
				Detail:   fmt.Sprintf("The block type name %q is reserved for use by Terraform in a future version.", block.Type),
				Subject:  &block.TypeRange,
			})
		}
	}

	return pv, diags
}

// Connection represents a "connection" block when used within either a
// "resource" or "provisioner" block in a module or file.
type Connection struct {
	Config hcl.Body

	DeclRange hcl.Range
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
		{Name: "when"},
		{Name: "on_failure"},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "connection"},
		{Type: "lifecycle"}, // reserved for future use
	},
}
