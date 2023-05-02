// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configschema

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	multierror "github.com/hashicorp/go-multierror"
)

func TestBlockInternalValidate(t *testing.T) {
	tests := map[string]struct {
		Block *Block
		Errs  []string
	}{
		"empty": {
			&Block{},
			[]string{},
		},
		"valid": {
			&Block{
				Attributes: map[string]*Attribute{
					"foo": {
						Type:     cty.String,
						Required: true,
					},
					"bar": {
						Type:     cty.String,
						Optional: true,
					},
					"baz": {
						Type:     cty.String,
						Computed: true,
					},
					"baz_maybe": {
						Type:     cty.String,
						Optional: true,
						Computed: true,
					},
				},
				BlockTypes: map[string]*NestedBlock{
					"single": {
						Nesting: NestingSingle,
						Block:   Block{},
					},
					"single_required": {
						Nesting:  NestingSingle,
						Block:    Block{},
						MinItems: 1,
						MaxItems: 1,
					},
					"list": {
						Nesting: NestingList,
						Block:   Block{},
					},
					"list_required": {
						Nesting:  NestingList,
						Block:    Block{},
						MinItems: 1,
					},
					"set": {
						Nesting: NestingSet,
						Block:   Block{},
					},
					"set_required": {
						Nesting:  NestingSet,
						Block:    Block{},
						MinItems: 1,
					},
					"map": {
						Nesting: NestingMap,
						Block:   Block{},
					},
				},
			},
			[]string{},
		},
		"attribute with no flags set": {
			&Block{
				Attributes: map[string]*Attribute{
					"foo": {
						Type: cty.String,
					},
				},
			},
			[]string{"foo: must set Optional, Required or Computed"},
		},
		"attribute required and optional": {
			&Block{
				Attributes: map[string]*Attribute{
					"foo": {
						Type:     cty.String,
						Required: true,
						Optional: true,
					},
				},
			},
			[]string{"foo: cannot set both Optional and Required"},
		},
		"attribute required and computed": {
			&Block{
				Attributes: map[string]*Attribute{
					"foo": {
						Type:     cty.String,
						Required: true,
						Computed: true,
					},
				},
			},
			[]string{"foo: cannot set both Computed and Required"},
		},
		"attribute optional and computed": {
			&Block{
				Attributes: map[string]*Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
						Computed: true,
					},
				},
			},
			[]string{},
		},
		"attribute with missing type": {
			&Block{
				Attributes: map[string]*Attribute{
					"foo": {
						Optional: true,
					},
				},
			},
			[]string{"foo: either Type or NestedType must be defined"},
		},
		/* FIXME: This caused errors when applied to existing providers (oci)
		and cannot be enforced without coordination.

		"attribute with invalid name": {&Block{Attributes:
		    map[string]*Attribute{"fooBar": {Type:     cty.String, Optional:
		    true,
		            },
		        },
		    },
		    []string{"fooBar: name may contain only lowercase letters, digits and underscores"},
		},
		*/
		"attribute with invalid NestedType attribute": {
			&Block{
				Attributes: map[string]*Attribute{
					"foo": {
						NestedType: &Object{
							Nesting: NestingSingle,
							Attributes: map[string]*Attribute{
								"foo": {
									Type:     cty.String,
									Required: true,
									Optional: true,
								},
							},
						},
						Optional: true,
					},
				},
			},
			[]string{"foo: cannot set both Optional and Required"},
		},
		"block type with invalid name": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"fooBar": {
						Nesting: NestingSingle,
					},
				},
			},
			[]string{"fooBar: name may contain only lowercase letters, digits and underscores"},
		},
		"colliding names": {
			&Block{
				Attributes: map[string]*Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
				},
				BlockTypes: map[string]*NestedBlock{
					"foo": {
						Nesting: NestingSingle,
					},
				},
			},
			[]string{"foo: name defined as both attribute and child block type"},
		},
		"nested block with badness": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"bad": {
						Nesting: NestingSingle,
						Block: Block{
							Attributes: map[string]*Attribute{
								"nested_bad": {
									Type:     cty.String,
									Required: true,
									Optional: true,
								},
							},
						},
					},
				},
			},
			[]string{"bad.nested_bad: cannot set both Optional and Required"},
		},
		"nested list block with dynamically-typed attribute": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"bad": {
						Nesting: NestingList,
						Block: Block{
							Attributes: map[string]*Attribute{
								"nested_bad": {
									Type:     cty.DynamicPseudoType,
									Optional: true,
								},
							},
						},
					},
				},
			},
			[]string{},
		},
		"nested set block with dynamically-typed attribute": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"bad": {
						Nesting: NestingSet,
						Block: Block{
							Attributes: map[string]*Attribute{
								"nested_bad": {
									Type:     cty.DynamicPseudoType,
									Optional: true,
								},
							},
						},
					},
				},
			},
			[]string{"bad: NestingSet blocks may not contain attributes of cty.DynamicPseudoType"},
		},
		"nil": {
			nil,
			[]string{"top-level block schema is nil"},
		},
		"nil attr": {
			&Block{
				Attributes: map[string]*Attribute{
					"bad": nil,
				},
			},
			[]string{"bad: attribute schema is nil"},
		},
		"nil block type": {
			&Block{
				BlockTypes: map[string]*NestedBlock{
					"bad": nil,
				},
			},
			[]string{"bad: block schema is nil"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			errs := multierrorErrors(test.Block.InternalValidate())
			if got, want := len(errs), len(test.Errs); got != want {
				t.Errorf("wrong number of errors %d; want %d", got, want)
				for _, err := range errs {
					t.Logf("- %s", err.Error())
				}
			} else {
				if len(errs) > 0 {
					for i := range errs {
						if errs[i].Error() != test.Errs[i] {
							t.Errorf("wrong error: got %s, want %s", errs[i].Error(), test.Errs[i])
						}
					}
				}
			}
		})
	}
}

func multierrorErrors(err error) []error {
	// A function like this should really be part of the multierror package...

	if err == nil {
		return nil
	}

	switch terr := err.(type) {
	case *multierror.Error:
		return terr.Errors
	default:
		return []error{err}
	}
}
