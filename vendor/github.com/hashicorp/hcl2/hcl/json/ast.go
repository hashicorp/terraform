package json

import (
	"math/big"

	"github.com/hashicorp/hcl2/hcl"
)

type node interface {
	Range() hcl.Range
	StartRange() hcl.Range
}

type objectVal struct {
	Attrs      map[string]*objectAttr
	SrcRange   hcl.Range // range of the entire object, brace-to-brace
	OpenRange  hcl.Range // range of the opening brace
	CloseRange hcl.Range // range of the closing brace
}

func (n *objectVal) Range() hcl.Range {
	return n.SrcRange
}

func (n *objectVal) StartRange() hcl.Range {
	return n.OpenRange
}

type objectAttr struct {
	Name      string
	Value     node
	NameRange hcl.Range // range of the name string
}

func (n *objectAttr) Range() hcl.Range {
	return n.NameRange
}

func (n *objectAttr) StartRange() hcl.Range {
	return n.NameRange
}

type arrayVal struct {
	Values    []node
	SrcRange  hcl.Range // range of the entire object, bracket-to-bracket
	OpenRange hcl.Range // range of the opening bracket
}

func (n *arrayVal) Range() hcl.Range {
	return n.SrcRange
}

func (n *arrayVal) StartRange() hcl.Range {
	return n.OpenRange
}

type booleanVal struct {
	Value    bool
	SrcRange hcl.Range
}

func (n *booleanVal) Range() hcl.Range {
	return n.SrcRange
}

func (n *booleanVal) StartRange() hcl.Range {
	return n.SrcRange
}

type numberVal struct {
	Value    *big.Float
	SrcRange hcl.Range
}

func (n *numberVal) Range() hcl.Range {
	return n.SrcRange
}

func (n *numberVal) StartRange() hcl.Range {
	return n.SrcRange
}

type stringVal struct {
	Value    string
	SrcRange hcl.Range
}

func (n *stringVal) Range() hcl.Range {
	return n.SrcRange
}

func (n *stringVal) StartRange() hcl.Range {
	return n.SrcRange
}

type nullVal struct {
	SrcRange hcl.Range
}

func (n *nullVal) Range() hcl.Range {
	return n.SrcRange
}

func (n *nullVal) StartRange() hcl.Range {
	return n.SrcRange
}

// invalidVal is used as a placeholder where a value is needed for a valid
// parse tree but the input was invalid enough to prevent one from being
// created.
type invalidVal struct {
	SrcRange hcl.Range
}

func (n invalidVal) Range() hcl.Range {
	return n.SrcRange
}

func (n invalidVal) StartRange() hcl.Range {
	return n.SrcRange
}
