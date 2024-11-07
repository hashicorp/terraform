// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mocking

import (
	"math/rand"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
)

var (
	normalAttributes = map[string]*configschema.Attribute{
		"id": {
			Type: cty.String,
		},
		"value": {
			Type: cty.String,
		},
	}

	computedAttributes = map[string]*configschema.Attribute{
		"id": {
			Type:     cty.String,
			Computed: true,
		},
		"value": {
			Type: cty.String,
		},
	}

	normalBlock = configschema.Block{
		Attributes: normalAttributes,
	}

	computedBlock = configschema.Block{
		Attributes: computedAttributes,
	}
)

func TestComputedValuesForDataSource(t *testing.T) {
	tcs := map[string]struct {
		target           cty.Value
		with             cty.Value
		schema           *configschema.Block
		expected         cty.Value
		expectedFailures []string
	}{
		"nil_target_no_unknowns": {
			target: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("kj87eb9"),
				"value": cty.StringVal("Hello, world!"),
			}),
			with:   cty.NilVal,
			schema: &normalBlock,
			expected: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("kj87eb9"),
				"value": cty.StringVal("Hello, world!"),
			}),
		},
		"empty_target_no_unknowns": {
			target: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("kj87eb9"),
				"value": cty.StringVal("Hello, world!"),
			}),
			with:   cty.EmptyObjectVal,
			schema: &normalBlock,
			expected: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("kj87eb9"),
				"value": cty.StringVal("Hello, world!"),
			}),
		},
		"basic_computed_attribute_preset": {
			target: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("kj87eb9"),
				"value": cty.StringVal("Hello, world!"),
			}),
			with:   cty.NilVal,
			schema: &computedBlock,
			expected: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("kj87eb9"),
				"value": cty.StringVal("Hello, world!"),
			}),
		},
		"basic_computed_attribute_random": {
			target: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.NullVal(cty.String),
				"value": cty.StringVal("Hello, world!"),
			}),
			with:   cty.NilVal,
			schema: &computedBlock,
			expected: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("ssnk9qhr"),
				"value": cty.StringVal("Hello, world!"),
			}),
		},
		"basic_computed_attribute_supplied": {
			target: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.NullVal(cty.String),
				"value": cty.StringVal("Hello, world!"),
			}),
			with: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("myvalue"),
			}),
			schema: &computedBlock,
			expected: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("myvalue"),
				"value": cty.StringVal("Hello, world!"),
			}),
		},
		"nested_single_block_preset": {
			target: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ObjectVal(map[string]cty.Value{
					"id":    cty.NullVal(cty.String),
					"value": cty.StringVal("Hello, world!"),
				}),
			}),
			with: cty.NilVal,
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Block:   computedBlock,
						Nesting: configschema.NestingSingle,
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("ssnk9qhr"),
					"value": cty.StringVal("Hello, world!"),
				}),
			}),
		},
		"nested_single_block_supplied": {
			target: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ObjectVal(map[string]cty.Value{
					"id":    cty.NullVal(cty.String),
					"value": cty.StringVal("Hello, world!"),
				}),
			}),
			with: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("myvalue"),
				}),
			}),
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Block:   computedBlock,
						Nesting: configschema.NestingSingle,
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("myvalue"),
					"value": cty.StringVal("Hello, world!"),
				}),
			}),
		},
		"nested_list_block_preset": {
			target: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.NullVal(cty.String),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.NullVal(cty.String),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
			with: cty.NilVal,
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Block:   computedBlock,
						Nesting: configschema.NestingList,
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("ssnk9qhr"),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("amyllmyg"),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
		},
		"nested_list_block_supplied": {
			target: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.NullVal(cty.String),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.NullVal(cty.String),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
			with: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("myvalue"),
				}),
			}),
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Block:   computedBlock,
						Nesting: configschema.NestingList,
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
		},
		"nested_set_block_preset": {
			target: cty.ObjectVal(map[string]cty.Value{
				"block": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.NullVal(cty.String),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.NullVal(cty.String),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
			with: cty.NilVal,
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Block:   computedBlock,
						Nesting: configschema.NestingSet,
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"block": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("ssnk9qhr"),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("amyllmyg"),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
		},
		"nested_set_block_supplied": {
			target: cty.ObjectVal(map[string]cty.Value{
				"block": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.NullVal(cty.String),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.NullVal(cty.String),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
			with: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("myvalue"),
				}),
			}),
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Block:   computedBlock,
						Nesting: configschema.NestingSet,
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"block": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
		},
		"nested_map_block_preset": {
			target: cty.ObjectVal(map[string]cty.Value{
				"block": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.NullVal(cty.String),
						"value": cty.StringVal("one"),
					}),
					"two": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.NullVal(cty.String),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
			with: cty.NilVal,
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Block:   computedBlock,
						Nesting: configschema.NestingMap,
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"block": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("ssnk9qhr"),
						"value": cty.StringVal("one"),
					}),
					"two": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("amyllmyg"),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
		},
		"nested_map_block_supplied": {
			target: cty.ObjectVal(map[string]cty.Value{
				"block": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.NullVal(cty.String),
						"value": cty.StringVal("one"),
					}),
					"two": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.NullVal(cty.String),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
			with: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("myvalue"),
				}),
			}),
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Block:   computedBlock,
						Nesting: configschema.NestingMap,
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"block": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("one"),
					}),
					"two": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
		},
		"nested_single_attribute": {
			target: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ObjectVal(map[string]cty.Value{
					"id":    cty.NullVal(cty.String),
					"value": cty.StringVal("Hello, world!"),
				}),
			}),
			with: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("myvalue"),
				}),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"nested": {
						NestedType: &configschema.Object{
							Attributes: computedAttributes,
							Nesting:    configschema.NestingSingle,
						},
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("myvalue"),
					"value": cty.StringVal("Hello, world!"),
				}),
			}),
		},
		"nested_single_attribute_generated": {
			target: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.NullVal(cty.Object(map[string]cty.Type{
					"id":    cty.String,
					"value": cty.String,
				})),
			}),
			with: cty.NilVal,
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"nested": {
						NestedType: &configschema.Object{
							Attributes: computedAttributes,
							Nesting:    configschema.NestingSingle,
						},
						Computed: true,
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("ssnk9qhr"),
					"value": cty.StringVal("amyllmyg"),
				}),
			}),
		},
		"nested_single_attribute_computed": {
			target: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.NullVal(cty.Object(map[string]cty.Type{
					"id":    cty.String,
					"value": cty.String,
				})),
			}),
			with: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("hello"),
				}),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"nested": {
						NestedType: &configschema.Object{
							Attributes: computedAttributes,
							Nesting:    configschema.NestingSingle,
						},
						Computed: true,
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("hello"),
					"value": cty.StringVal("ssnk9qhr"),
				}),
			}),
		},
		"nested_list_attribute": {
			target: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.NullVal(cty.String),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.NullVal(cty.String),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
			with: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("myvalue"),
				}),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"nested": {
						NestedType: &configschema.Object{
							Attributes: computedAttributes,
							Nesting:    configschema.NestingList,
						},
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
		},
		"nested_list_attribute_generated": {
			target: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"id":    cty.String,
					"value": cty.String,
				}))),
			}),
			with: cty.NilVal,
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"nested": {
						NestedType: &configschema.Object{
							Attributes: computedAttributes,
							Nesting:    configschema.NestingList,
						},
						Computed: true,
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ListValEmpty(cty.Object(map[string]cty.Type{
					"id":    cty.String,
					"value": cty.String,
				})),
			}),
		},
		"nested_list_attribute_computed": {
			target: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
					"id":    cty.String,
					"value": cty.String,
				}))),
			}),
			with: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("myvalue"),
				}),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"nested": {
						NestedType: &configschema.Object{
							Attributes: computedAttributes,
							Nesting:    configschema.NestingList,
						},
						Computed: true,
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ListValEmpty(cty.Object(map[string]cty.Type{
					"id":    cty.String,
					"value": cty.String,
				})),
			}),
		},
		"nested_set_attribute": {
			target: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.NullVal(cty.String),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.NullVal(cty.String),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
			with: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("myvalue"),
				}),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"nested": {
						NestedType: &configschema.Object{
							Attributes: computedAttributes,
							Nesting:    configschema.NestingSet,
						},
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
		},
		"nested_set_attribute_generated": {
			target: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"id":    cty.String,
					"value": cty.String,
				}))),
			}),
			with: cty.NilVal,
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"nested": {
						NestedType: &configschema.Object{
							Attributes: computedAttributes,
							Nesting:    configschema.NestingSet,
						},
						Computed: true,
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"id":    cty.String,
					"value": cty.String,
				})),
			}),
		},
		"nested_set_attribute_computed": {
			target: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"id":    cty.String,
					"value": cty.String,
				}))),
			}),
			with: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("myvalue"),
				}),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"nested": {
						NestedType: &configschema.Object{
							Attributes: computedAttributes,
							Nesting:    configschema.NestingSet,
						},
						Computed: true,
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"id":    cty.String,
					"value": cty.String,
				})),
			}),
		},
		"nested_map_attribute": {
			target: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.NullVal(cty.String),
						"value": cty.StringVal("one"),
					}),
					"two": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.NullVal(cty.String),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
			with: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("myvalue"),
				}),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"nested": {
						NestedType: &configschema.Object{
							Attributes: computedAttributes,
							Nesting:    configschema.NestingMap,
						},
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("one"),
					}),
					"two": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
		},
		"nested_map_attribute_generated": {
			target: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.NullVal(cty.Map(cty.Object(map[string]cty.Type{
					"id":    cty.String,
					"value": cty.String,
				}))),
			}),
			with: cty.NilVal,
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"nested": {
						NestedType: &configschema.Object{
							Attributes: computedAttributes,
							Nesting:    configschema.NestingMap,
						},
						Computed: true,
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.MapValEmpty(cty.Object(map[string]cty.Type{
					"id":    cty.String,
					"value": cty.String,
				})),
			}),
		},
		"nested_map_attribute_computed": {
			target: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.NullVal(cty.Map(cty.Object(map[string]cty.Type{
					"id":    cty.String,
					"value": cty.String,
				}))),
			}),
			with: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("myvalue"),
				}),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"nested": {
						NestedType: &configschema.Object{
							Attributes: computedAttributes,
							Nesting:    configschema.NestingMap,
						},
						Computed: true,
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.MapValEmpty(cty.Object(map[string]cty.Type{
					"id":    cty.String,
					"value": cty.String,
				})),
			}),
		},
		"invalid_replacement_path": {
			target: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("kj87eb9"),
				"value": cty.StringVal("Hello, world!"),
			}),
			with:   cty.StringVal("Hello, world!"),
			schema: &normalBlock,
			expected: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("kj87eb9"),
				"value": cty.StringVal("Hello, world!"),
			}),
			expectedFailures: []string{
				"The requested replacement value must be an object type, but was string.",
			},
		},
		"invalid_replacement_path_nested": {
			target: cty.ObjectVal(map[string]cty.Value{
				"nested_object": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id": cty.NullVal(cty.String),
					}),
				}),
			}),
			with: cty.ObjectVal(map[string]cty.Value{
				"nested_object": cty.StringVal("Hello, world!"),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"nested_object": {
						NestedType: &configschema.Object{
							Attributes: map[string]*configschema.Attribute{
								"id": {
									Type:     cty.String,
									Computed: true,
								},
							},
							Nesting: configschema.NestingSet,
						},
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"nested_object": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal("ssnk9qhr"),
					}),
				}),
			}),
			expectedFailures: []string{
				"Terraform expected an object type for attribute \"nested_object\" defined within the mocked data at :0,0-0, but found string.",
			},
		},
		"invalid_replacement_path_nested_block": {
			target: cty.ObjectVal(map[string]cty.Value{
				"nested_object": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id": cty.NullVal(cty.String),
					}),
				}),
			}),
			with: cty.ObjectVal(map[string]cty.Value{
				"nested_object": cty.StringVal("Hello, world!"),
			}),
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"nested_object": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"id": {
									Type:     cty.String,
									Computed: true,
								},
							},
						},
						Nesting: configschema.NestingSet,
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"nested_object": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal("ssnk9qhr"),
					}),
				}),
			}),
			expectedFailures: []string{
				"Terraform expected an object type for attribute \"nested_object\" defined within the mocked data at :0,0-0, but found string.",
			},
		},
		"invalid_replacement_type": {
			target: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.NullVal(cty.String),
				"value": cty.StringVal("Hello, world!"),
			}),
			with: cty.ObjectVal(map[string]cty.Value{
				"id": cty.ListValEmpty(cty.String),
			}),
			schema: &computedBlock,
			expected: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("ssnk9qhr"),
				"value": cty.StringVal("Hello, world!"),
			}),
			expectedFailures: []string{
				"Terraform could not compute a value for the target type string with the mocked data defined at :0,0-0 with the attribute \"id\": string required.",
			},
		},
		"invalid_replacement_type_nested": {
			target: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.NullVal(cty.String),
						"value": cty.StringVal("one"),
					}),
				}),
			}),
			with: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ObjectVal(map[string]cty.Value{
					"id": cty.EmptyObjectVal,
				}),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"nested": {
						NestedType: &configschema.Object{
							Attributes: computedAttributes,
							Nesting:    configschema.NestingMap,
						},
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("ssnk9qhr"),
						"value": cty.StringVal("one"),
					}),
				}),
			}),
			expectedFailures: []string{
				"Terraform could not compute a value for the target type string with the mocked data defined at :0,0-0 with the attribute \"nested.id\": string required.",
			},
		},
		"invalid_replacement_type_nested_block": {
			target: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.NullVal(cty.String),
						"value": cty.StringVal("one"),
					}),
				}),
			}),
			with: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ObjectVal(map[string]cty.Value{
					"id": cty.EmptyObjectVal,
				}),
			}),
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Block:   computedBlock,
						Nesting: configschema.NestingList,
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("ssnk9qhr"),
						"value": cty.StringVal("one"),
					}),
				}),
			}),
			expectedFailures: []string{
				"Terraform could not compute a value for the target type string with the mocked data defined at :0,0-0 with the attribute \"block.id\": string required.",
			},
		},
		"dynamic_attribute_unset": {
			target: cty.ObjectVal(map[string]cty.Value{
				"dynamic_attribute": cty.NullVal(cty.DynamicPseudoType),
			}),
			with: cty.EmptyObjectVal,
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"dynamic_attribute": {
						Type:     cty.DynamicPseudoType,
						Computed: true,
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"dynamic_attribute": cty.NullVal(cty.DynamicPseudoType),
			}),
		},
		"dynamic_attribute_set": {
			target: cty.ObjectVal(map[string]cty.Value{
				"dynamic_attribute": cty.NullVal(cty.DynamicPseudoType),
			}),
			with: cty.ObjectVal(map[string]cty.Value{
				"dynamic_attribute": cty.StringVal("Hello, world!"),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"dynamic_attribute": {
						Type:     cty.DynamicPseudoType,
						Computed: true,
					},
				},
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"dynamic_attribute": cty.StringVal("Hello, world!"),
			}),
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {

			// We'll just make sure that any random strings are deterministic.
			testRand = rand.New(rand.NewSource(0))
			defer func() {
				testRand = nil
			}()

			actual, diags := ComputedValuesForDataSource(tc.target, MockedData{
				Value: tc.with,
			}, tc.schema)

			var actualFailures []string
			for _, diag := range diags {
				actualFailures = append(actualFailures, diag.Description().Detail)
			}
			if diff := cmp.Diff(tc.expectedFailures, actualFailures); len(diff) > 0 {
				t.Errorf("unexpected failures\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", tc.expectedFailures, actualFailures, diff)
			}

			if actual.Equals(tc.expected).False() {
				t.Errorf("\nexpected: (%s)\nactual:   (%s)", tc.expected.GoString(), actual.GoString())
			}
		})
	}
}
