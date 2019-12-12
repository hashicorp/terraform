package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
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

	// destroy provisioners can only refer to self
	if pv.When == ProvisionerWhenDestroy {
		diags = append(diags, onlySelfRefs(config)...)
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

			// destroy provisioners can only refer to self
			if pv.When == ProvisionerWhenDestroy {
				diags = append(diags, onlySelfRefs(block.Body)...)
			}

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

func onlySelfRefs(body hcl.Body) hcl.Diagnostics {
	var diags hcl.Diagnostics

	// Provisioners currently do not use any blocks in their configuration.
	// Blocks are likely to remain solely for meta parameters, but in the case
	// that blocks are supported for provisioners, we will want to extend this
	// to find variables in nested blocks.
	attrs, _ := body.JustAttributes()
	for _, attr := range attrs {
		for _, v := range attr.Expr.Variables() {
			valid := false
			switch v.RootName() {
			case "self", "path", "terraform":
				valid = true
			case "count":
				// count must use "index"
				if len(v) == 2 {
					if t, ok := v[1].(hcl.TraverseAttr); ok && t.Name == "index" {
						valid = true
					}
				}

			case "each":
				if len(v) == 2 {
					if t, ok := v[1].(hcl.TraverseAttr); ok && t.Name == "key" {
						valid = true
					}
				}
			}

			if !valid {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid reference from destroy provisioner",
					Detail: "Destroy-time provisioners and their connection configurations may only " +
						"reference attributes of the related resource, via 'self', 'count.index', " +
						"or 'each.key'.\n\nReferences to other resources during the destroy phase " +
						"can cause dependency cycles and interact poorly with create_before_destroy.",
					Subject: attr.Expr.Range().Ptr(),
				})
			}
		}
	}
	return diags
}

// Connection represents a "connection" block when used within either a
// "resource" or "provisioner" block in a module or file.
type Connection struct {
	Config hcl.Body

	DeclRange hcl.Range
}

// ProvisionerWhen is an enum for valid values for when to run provisioners.
type ProvisionerWhen int

//go:generate go run golang.org/x/tools/cmd/stringer -type ProvisionerWhen

const (
	ProvisionerWhenInvalid ProvisionerWhen = iota
	ProvisionerWhenCreate
	ProvisionerWhenDestroy
)

// ProvisionerOnFailure is an enum for valid values for on_failure options
// for provisioners.
type ProvisionerOnFailure int

//go:generate go run golang.org/x/tools/cmd/stringer -type ProvisionerOnFailure

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
