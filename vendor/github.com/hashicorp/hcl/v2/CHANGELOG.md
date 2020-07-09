# HCL Changelog

## v2.6.0 (June 4, 2020)

### Enhancements

* hcldec: Add a new `Spec`, `ValidateSpec`, which allows custom validation of values at decode-time. ([#387](https://github.com/hashicorp/hcl/pull/387))

### Bugs Fixed

* hclsyntax: Fix panic with combination of sequences and null arguments ([#386](https://github.com/hashicorp/hcl/pull/386))
* hclsyntax: Fix handling of unknown values and sequences ([#386](https://github.com/hashicorp/hcl/pull/386))

## v2.5.1 (May 14, 2020)

### Bugs Fixed

* hclwrite: handle legacy dot access of numeric indexes. ([#369](https://github.com/hashicorp/hcl/pull/369))
* hclwrite: Fix panic for dotted full splat (`foo.*`) ([#374](https://github.com/hashicorp/hcl/pull/374))

## v2.5.0 (May 6, 2020)

### Enhancements

* hclwrite: Generate multi-line objects and maps. ([#372](https://github.com/hashicorp/hcl/pull/372))

## v2.4.0 (Apr 13, 2020)

### Enhancements

* The Unicode data tables that HCL uses to produce user-perceived "column" positions in diagnostics and other source ranges are now updated to Unicode 12.0.0, which will cause HCL to produce more accurate column numbers for combining characters introduced to Unicode since Unicode 9.0.0.

### Bugs Fixed

* json: Fix panic when parsing malformed JSON. ([#358](https://github.com/hashicorp/hcl/pull/358))

## v2.3.0 (Jan 3, 2020)

### Enhancements

* ext/tryfunc: Optional functions `try` and `can` to include in your `hcl.EvalContext` when evaluating expressions, which allow users to make decisions based on the success of expressions. ([#330](https://github.com/hashicorp/hcl/pull/330))
* ext/typeexpr: Now has an optional function `convert` which you can include in your `hcl.EvalContext` when evaluating expressions, allowing users to convert values to specific type constraints using the type constraint expression syntax. ([#330](https://github.com/hashicorp/hcl/pull/330))
* ext/typeexpr: A new `cty` capsule type `typeexpr.TypeConstraintType` which, when used as either a type constraint for a function parameter or as a type constraint for a `hcldec` attribute specification will cause the given expression to be interpreted as a type constraint expression rather than a value expression. ([#330](https://github.com/hashicorp/hcl/pull/330))
* ext/customdecode: An optional extension that allows overriding the static decoding behavior for expressions either in function arguments or `hcldec` attribute specifications. ([#330](https://github.com/hashicorp/hcl/pull/330))
* ext/customdecode: New `cty` capsuletypes `customdecode.ExpressionType` and `customdecode.ExpressionClosureType` which, when used as either a type constraint for a function parameter or as a type constraint for a `hcldec` attribute specification will cause the given expression (and, for the closure type, also the `hcl.EvalContext` it was evaluated in) to be captured for later analysis, rather than immediately evaluated. ([#330](https://github.com/hashicorp/hcl/pull/330))

## v2.2.0 (Dec 11, 2019)

### Enhancements

* hcldec: Attribute evaluation (as part of `AttrSpec` or `BlockAttrsSpec`) now captures expression evaluation metadata in any errors it produces during type conversions, allowing for better feedback in calling applications that are able to make use of this metadata when printing diagnostic messages. ([#329](https://github.com/hashicorp/hcl/pull/329))

### Bugs Fixed

* hclsyntax: `IndexExpr`, `SplatExpr`, and `RelativeTraversalExpr` will now report a source range that covers all of their child expression  nodes. Previously they would report only the operator part, such as `["foo"]`, `[*]`, or `.foo`, which was problematic for callers using source ranges for code analysis. ([#328](https://github.com/hashicorp/hcl/pull/328))
* hclwrite: Parser will no longer panic when the input includes index, splat, or relative traversal syntax.  ([#328](https://github.com/hashicorp/hcl/pull/328))

## v2.1.0 (Nov 19, 2019)

### Enhancements

* gohcl: When decoding into a struct value with some fields already populated, those values will be retained if not explicitly overwritten in the given HCL body, with similar overriding/merging behavior as `json.Unmarshal` in the Go standard library.
* hclwrite: New interface to set the expression for an attribute to be a raw token sequence, with no special processing. This has some caveats, so if you intend to use it please refer to the godoc comments. ([#320](https://github.com/hashicorp/hcl/pull/320))

### Bugs Fixed

* hclwrite: The `Body.Blocks` method was returing the blocks in an indefined order, rather than preserving the order of declaration in the source input. ([#313](https://github.com/hashicorp/hcl/pull/313))
* hclwrite: The `TokensForTraversal` function (and thus in turn the `Body.SetAttributeTraversal` method) was not correctly handling index steps in traversals, and thus producing invalid results. ([#319](https://github.com/hashicorp/hcl/pull/319))

## v2.0.0 (Oct 2, 2019)

Initial release of HCL 2, which is a new implementating combining the HCL 1
language with the HIL expression language to produce a single language
supporting both nested configuration structures and arbitrary expressions.

HCL 2 has an entirely new Go library API and so is _not_ a drop-in upgrade
relative to HCL 1. It's possible to import both versions of HCL into a single
program using Go's _semantic import versioning_ mechanism:

```
import (
    hcl1 "github.com/hashicorp/hcl"
    hcl2 "github.com/hashicorp/hcl/v2"
)
```

---

Prior to v2.0.0 there was not a curated changelog. Consult the git history
from the latest v1.x.x tag for information on the changes to HCL 1.
