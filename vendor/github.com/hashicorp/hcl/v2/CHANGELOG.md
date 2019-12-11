# HCL Changelog

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
