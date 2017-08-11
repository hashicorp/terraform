package json

import (
	"math/big"

	"github.com/zclconf/go-zcl/zcl"
)

type node interface {
	Range() zcl.Range
	StartRange() zcl.Range
}

type objectVal struct {
	Attrs      map[string]*objectAttr
	SrcRange   zcl.Range // range of the entire object, brace-to-brace
	OpenRange  zcl.Range // range of the opening brace
	CloseRange zcl.Range // range of the closing brace
}

func (n *objectVal) Range() zcl.Range {
	return n.SrcRange
}

func (n *objectVal) StartRange() zcl.Range {
	return n.OpenRange
}

type objectAttr struct {
	Name      string
	Value     node
	NameRange zcl.Range // range of the name string
}

func (n *objectAttr) Range() zcl.Range {
	return n.NameRange
}

func (n *objectAttr) StartRange() zcl.Range {
	return n.NameRange
}

type arrayVal struct {
	Values    []node
	SrcRange  zcl.Range // range of the entire object, bracket-to-bracket
	OpenRange zcl.Range // range of the opening bracket
}

func (n *arrayVal) Range() zcl.Range {
	return n.SrcRange
}

func (n *arrayVal) StartRange() zcl.Range {
	return n.OpenRange
}

type booleanVal struct {
	Value    bool
	SrcRange zcl.Range
}

func (n *booleanVal) Range() zcl.Range {
	return n.SrcRange
}

func (n *booleanVal) StartRange() zcl.Range {
	return n.SrcRange
}

type numberVal struct {
	Value    *big.Float
	SrcRange zcl.Range
}

func (n *numberVal) Range() zcl.Range {
	return n.SrcRange
}

func (n *numberVal) StartRange() zcl.Range {
	return n.SrcRange
}

type stringVal struct {
	Value    string
	SrcRange zcl.Range
}

func (n *stringVal) Range() zcl.Range {
	return n.SrcRange
}

func (n *stringVal) StartRange() zcl.Range {
	return n.SrcRange
}

type nullVal struct {
	SrcRange zcl.Range
}

func (n *nullVal) Range() zcl.Range {
	return n.SrcRange
}

func (n *nullVal) StartRange() zcl.Range {
	return n.SrcRange
}

// invalidVal is used as a placeholder where a value is needed for a valid
// parse tree but the input was invalid enough to prevent one from being
// created.
type invalidVal struct {
	SrcRange zcl.Range
}

func (n invalidVal) Range() zcl.Range {
	return n.SrcRange
}

func (n invalidVal) StartRange() zcl.Range {
	return n.SrcRange
}
