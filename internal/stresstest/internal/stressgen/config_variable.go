package stressgen

import (
	"log"
	"math/rand"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/states"
	"github.com/zclconf/go-cty/cty"
)

// ConfigVariable is an implementation of ConfigObject representing the
// declaration of an input variable.
type ConfigVariable struct {
	Addr           addrs.InputVariable
	TypeConstraint cty.Type
	DefaultValue   cty.Value
	CallerWillSet  bool
}

var _ ConfigObject = (*ConfigVariable)(nil)

// ConfigVariableInstance represents the binding between a ConfigVariable and
// a particular module instance.
type ConfigVariableInstance struct {
	Addr          addrs.AbsInputVariableInstance
	Obj           *ConfigVariable
	ExpectedValue cty.Value
}

var _ ConfigObjectInstance = (*ConfigVariableInstance)(nil)

// DisplayName implements ConfigObject.DisplayName.
func (v *ConfigVariable) DisplayName() string {
	return v.Addr.String()
}

// AppendConfig implements ConfigObject.AppendConfig.
func (v *ConfigVariable) AppendConfig(to *hclwrite.Body) {
	block := hclwrite.NewBlock("variable", []string{v.Addr.Name})
	body := block.Body()
	if v.TypeConstraint != cty.NilType {
		body.SetAttributeRaw("type", tokensForTypeConstraint(v.TypeConstraint))
	}
	if v.DefaultValue != cty.NilVal {
		body.SetAttributeValue("default", v.DefaultValue)
	}
	to.AppendBlock(block)
}

// GenerateModified implements ConfigObject.GenerateModified.
func (v *ConfigVariable) GenerateModified(rnd *rand.Rand, ns *Namespace) ConfigObject {
	declareConfigVariable(v, ns)
	return v
}

// Instantiate implements ConfigObject.Instantiate.
func (v *ConfigVariable) Instantiate(reg *Registry) ConfigObjectInstance {
	var expectedVal cty.Value
	if v.CallerWillSet {
		// If our caller is correctly implemented, we should have a
		// caller-chosen value for this variable recorded in the registry.
		// If this panics, that suggests a bug in the caller where it failed
		// to register a chosen value.
		expectedVal = reg.VariableValue(v.Addr)
	} else {
		// Variables not set by the caller take on their default value.
		// v.DefaultValue should always be set if we get here, because
		// non-optional variables should always set v.CallerWillSet.
		expectedVal = v.DefaultValue
	}

	reg.RegisterRefValue(v.Addr, expectedVal)
	return &ConfigVariableInstance{
		Addr:          v.Addr.Absolute(reg.ModuleAddr),
		Obj:           v,
		ExpectedValue: expectedVal,
	}
}

// DisplayName implements ConfigObjectInstance.DisplayName.
func (v *ConfigVariableInstance) DisplayName() string {
	return v.Addr.String()
}

// Object implements ConfigObjectInstance.Object.
func (v *ConfigVariableInstance) Object() ConfigObject {
	return v.Obj
}

// CheckState implements ConfigObjectInstance.CheckState.
func (v *ConfigVariableInstance) CheckState(prior, new *states.State) []error {
	log.Printf("Checking %s", v.DisplayName())

	// Input variables are not recorded in the state, so we have
	// nothing to check here. We generate input variables only
	// so other config objects might refer to them.
	return nil
}

func tokensForTypeConstraint(ty cty.Type) hclwrite.Tokens {
	// This is, in a sense, a type-expression-flavored version of
	// hclwrite.TokensForValue. If we find ourselves doing this in several
	// other situations then it might be worth upstreaming it into HCL, but
	// this seems like a reasonable place for it to live for now.
	return appendTokensForTypeConstraint(make(hclwrite.Tokens, 0, 4), ty)
}

func appendTokensForTypeConstraint(into hclwrite.Tokens, ty cty.Type) hclwrite.Tokens {
	switch {
	case ty == cty.String:
		into = append(into, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("string"),
		})
	case ty == cty.Number:
		into = append(into, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("number"),
		})
	case ty == cty.Bool:
		into = append(into, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("bool"),
		})
	case ty == cty.DynamicPseudoType:
		into = append(into, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("any"),
		})
	case ty.IsCollectionType():
		switch {
		case ty.IsListType():
			into = append(into, &hclwrite.Token{
				Type:  hclsyntax.TokenIdent,
				Bytes: []byte("list"),
			})
		case ty.IsMapType():
			into = append(into, &hclwrite.Token{
				Type:  hclsyntax.TokenIdent,
				Bytes: []byte("map"),
			})
		case ty.IsSetType():
			into = append(into, &hclwrite.Token{
				Type:  hclsyntax.TokenIdent,
				Bytes: []byte("set"),
			})
		default:
			panic("unsupported collection type") // the above should be exhaustive
		}
		into = append(into, &hclwrite.Token{
			Type:  hclsyntax.TokenOParen,
			Bytes: []byte{'('},
		})
		into = appendTokensForTypeConstraint(into, ty.ElementType())
		into = append(into, &hclwrite.Token{
			Type:  hclsyntax.TokenCParen,
			Bytes: []byte{')'},
		})
	case ty.IsObjectType():
		into = append(into, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("object"),
		})
		into = append(into, &hclwrite.Token{
			Type:  hclsyntax.TokenOParen,
			Bytes: []byte{'('},
		})
		into = append(into, &hclwrite.Token{
			Type:  hclsyntax.TokenOBrace,
			Bytes: []byte{'{'},
		})
		i := 0
		for k, aty := range ty.AttributeTypes() {
			if i > 0 {
				into = append(into, &hclwrite.Token{
					Type:  hclsyntax.TokenComma,
					Bytes: []byte{','},
				})
			}
			into = append(into, hclwrite.TokensForValue(cty.StringVal(k))...)
			into = append(into, &hclwrite.Token{
				Type:  hclsyntax.TokenEqual,
				Bytes: []byte{'='},
			})
			into = appendTokensForTypeConstraint(into, aty)
			i++
		}
		into = append(into, &hclwrite.Token{
			Type:  hclsyntax.TokenCBrace,
			Bytes: []byte{'}'},
		})
		into = append(into, &hclwrite.Token{
			Type:  hclsyntax.TokenCParen,
			Bytes: []byte{')'},
		})
	case ty.IsTupleType():
		into = append(into, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("tuple"),
		})
		into = append(into, &hclwrite.Token{
			Type:  hclsyntax.TokenOParen,
			Bytes: []byte{'('},
		})
		into = append(into, &hclwrite.Token{
			Type:  hclsyntax.TokenOBrack,
			Bytes: []byte{'['},
		})
		for i, ety := range ty.TupleElementTypes() {
			if i > 0 {
				into = append(into, &hclwrite.Token{
					Type:  hclsyntax.TokenComma,
					Bytes: []byte{','},
				})
			}
			into = appendTokensForTypeConstraint(into, ety)
		}
		into = append(into, &hclwrite.Token{
			Type:  hclsyntax.TokenCBrack,
			Bytes: []byte{']'},
		})
		into = append(into, &hclwrite.Token{
			Type:  hclsyntax.TokenCParen,
			Bytes: []byte{')'},
		})
	default:
		// The above should be exhaustive for all types that Terraform uses.
		// If we add new capsule types that can be used as variable type
		// constraints in future, we'll need to add new cases to the above
		// to serialize those constraints.
		panic("unsupported type")
	}

	return into
}
