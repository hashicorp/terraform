# HCL Dynamic Blocks Extension

This HCL extension implements a special block type named "dynamic" that can
be used to dynamically generate blocks of other types by iterating over
collection values.

Normally the block structure in an HCL configuration file is rigid, even
though dynamic expressions can be used within attribute values. This is
convenient for most applications since it allows the overall structure of
the document to be decoded easily, but in some applications it is desirable
to allow dynamic block generation within certain portions of the configuration.

Dynamic block generation is performed using the `dynamic` block type:

```hcl
toplevel {
  nested {
    foo = "static block 1"
  }

  dynamic "nested" {
    for_each = ["a", "b", "c"]
    iterator = nested
    content {
      foo = "dynamic block ${nested.value}"
    }
  }

  nested {
    foo = "static block 2"
  }
}
```

The above is interpreted as if it were written as follows:

```hcl
toplevel {
  nested {
    foo = "static block 1"
  }

  nested {
    foo = "dynamic block a"
  }

  nested {
    foo = "dynamic block b"
  }

  nested {
    foo = "dynamic block c"
  }

  nested {
    foo = "static block 2"
  }
}
```

Since HCL block syntax is not normally exposed to the possibility of unknown
values, this extension must make some compromises when asked to iterate over
an unknown collection. If the length of the collection cannot be statically
recognized (because it is an unknown value of list, map, or set type) then
the `dynamic` construct will generate a _single_ dynamic block whose iterator
key and value are both unknown values of the dynamic pseudo-type, thus causing
any attribute values derived from iteration to appear as unknown values. There
is no explicit representation of the fact that the length of the collection may
eventually be different than one.

## Usage

Pass a body to function `Expand` to obtain a new body that will, on access
to its content, evaluate and expand any nested `dynamic` blocks.
Dynamic block processing is also automatically propagated into any nested
blocks that are returned, allowing users to nest dynamic blocks inside
one another and to nest dynamic blocks inside other static blocks.

HCL structural decoding does not normally have access to an `EvalContext`, so
any variables and functions that should be available to the `for_each`
and `labels` expressions must be passed in when calling `Expand`. Expressions
within the `content` block are evaluated separately and so can be passed a
separate `EvalContext` if desired, during normal attribute expression
evaluation.

## Detecting Variables

Some applications dynamically generate an `EvalContext` by analyzing which
variables are referenced by an expression before evaluating it.

This unfortunately requires some extra effort when this analysis is required
for the context passed to `Expand`: the HCL API requires a schema to be
provided in order to do any analysis of the blocks in a body, but the low-level
schema model provides a description of only one level of nested blocks at
a time, and thus a new schema must be provided for each additional level of
nesting.

To make this arduous process as convenient as possbile, this package provides
a helper function `WalkForEachVariables`, which returns a `WalkVariablesNode`
instance that can be used to find variables directly in a given body and also
determine which nested blocks require recursive calls. Using this mechanism
requires that the caller be able to look up a schema given a nested block type.
For _simple_ formats where a specific block type name always has the same schema
regardless of context, a walk can be implemented as follows:

```go
func walkVariables(node dynblock.WalkVariablesNode, schema *hcl.BodySchema) []hcl.Traversal {
	vars, children := node.Visit(schema)

	for _, child := range children {
		var childSchema *hcl.BodySchema
		switch child.BlockTypeName {
		case "a":
			childSchema = &hcl.BodySchema{
				Blocks: []hcl.BlockHeaderSchema{
					{
						Type:       "b",
						LabelNames: []string{"key"},
					},
				},
			}
		case "b":
			childSchema = &hcl.BodySchema{
				Attributes: []hcl.AttributeSchema{
					{
						Name:     "val",
						Required: true,
					},
				},
			}
		default:
			// Should never happen, because the above cases should be exhaustive
			// for the application's configuration format.
			panic(fmt.Errorf("can't find schema for unknown block type %q", child.BlockTypeName))
		}

		vars = append(vars, testWalkAndAccumVars(child.Node, childSchema)...)
	}
}
```

### Detecting Variables with `hcldec` Specifications

For applications that use the higher-level `hcldec` package to decode nested
configuration structures into `cty` values, the same specification can be used
to automatically drive the recursive variable-detection walk described above.

The helper function `ForEachVariablesHCLDec` allows an entire recursive
configuration structure to be analyzed in a single call given a `hcldec.Spec`
that describes the nested block structure. This means a `hcldec`-based
application can support dynamic blocks with only a little additional effort:

```go
func decodeBody(body hcl.Body, spec hcldec.Spec) (cty.Value, hcl.Diagnostics) {
	// Determine which variables are needed to expand dynamic blocks
	neededForDynamic := dynblock.ForEachVariablesHCLDec(body, spec)

	// Build a suitable EvalContext and expand dynamic blocks
	dynCtx := buildEvalContext(neededForDynamic)
	dynBody := dynblock.Expand(body, dynCtx)

	// Determine which variables are needed to fully decode the expanded body
	// This will analyze expressions that came both from static blocks in the
	// original body and from blocks that were dynamically added by Expand.
	neededForDecode := hcldec.Variables(dynBody, spec)

	// Build a suitable EvalContext and then fully decode the body as per the
	// hcldec specification.
	decCtx := buildEvalContext(neededForDecode)
	return hcldec.Decode(dynBody, spec, decCtx)
}

func buildEvalContext(needed []hcl.Traversal) *hcl.EvalContext {
	// (to be implemented by your application)
}
```

# Performance

This extension is going quite harshly against the grain of the HCL API, and
so it uses lots of wrapping objects and temporary data structures to get its
work done. HCL in general is not suitable for use in high-performance situations
or situations sensitive to memory pressure, but that is _especially_ true for
this extension.
