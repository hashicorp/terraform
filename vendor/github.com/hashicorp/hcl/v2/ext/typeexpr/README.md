# HCL Type Expressions Extension

This HCL extension defines a convention for describing HCL types using function
call and variable reference syntax, allowing configuration formats to include
type information provided by users.

The type syntax is processed statically from a hcl.Expression, so it cannot
use any of the usual language operators. This is similar to type expressions
in statically-typed programming languages.

```hcl
variable "example" {
  type = list(string)
}
```

The extension is built using the `hcl.ExprAsKeyword` and `hcl.ExprCall`
functions, and so it relies on the underlying syntax to define how "keyword"
and "call" are interpreted. The above shows how they are interpreted in
the HCL native syntax, while the following shows the same information
expressed in JSON:

```json
{
  "variable": {
    "example": {
      "type": "list(string)"
    }
  }
}
```

Notice that since we have additional contextual information that we intend
to allow only calls and keywords the JSON syntax is able to parse the given
string directly as an expression, rather than as a template as would be
the case for normal expression evaluation.

For more information, see [the godoc reference](http://godoc.org/github.com/hashicorp/hcl/v2/ext/typeexpr).

## Type Expression Syntax

When expressed in the native syntax, the following expressions are permitted
in a type expression:

* `string` - string
* `bool` - boolean
* `number` - number
* `any` - `cty.DynamicPseudoType` (in function `TypeConstraint` only)
* `list(<type_expr>)` - list of the type given as an argument
* `set(<type_expr>)` - set of the type given as an argument
* `map(<type_expr>)` - map of the type given as an argument
* `tuple([<type_exprs...>])` - tuple with the element types given in the single list argument
* `object({<attr_name>=<type_expr>, ...}` - object with the attributes and corresponding types given in the single map argument

For example:

* `list(string)`
* `object({name=string,age=number})`
* `map(object({name=string,age=number}))`

Note that the object constructor syntax is not fully-general for all possible
object types because it requires the attribute names to be valid identifiers.
In practice it is expected that any time an object type is being fixed for
type checking it will be one that has identifiers as its attributes; object
types with weird attributes generally show up only from arbitrary object
constructors in configuration files, which are usually treated either as maps
or as the dynamic pseudo-type.

## Type Constraints as Values

Along with defining a convention for writing down types using HCL expression
constructs, this package also includes a mechanism for representing types as
values that can be used as data within an HCL-based language.

`typeexpr.TypeConstraintType` is a
[`cty` capsule type](https://github.com/zclconf/go-cty/blob/master/docs/types.md#capsule-types)
that encapsulates `cty.Type` values. You can construct such a value directly
using the `TypeConstraintVal` function:

```go
tyVal := typeexpr.TypeConstraintVal(cty.String)

// We can unpack the type from a value using TypeConstraintFromVal
ty := typeExpr.TypeConstraintFromVal(tyVal)
```

However, the primary purpose of `typeexpr.TypeConstraintType` is to be
specified as the type constraint for an argument, in which case it serves
as a signal for HCL to treat the argument expression as a type constraint
expression as defined above, rather than as a normal value expression.

"An argument" in the above in practice means the following two locations:

* As the type constraint for a parameter of a cty function that will be
  used in an `hcl.EvalContext`. In that case, function calls in the HCL
  native expression syntax will require the argument to be valid type constraint
  expression syntax and the function implementation will receive a
  `TypeConstraintType` value as the argument value for that parameter.

* As the type constraint for a `hcldec.AttrSpec` or `hcldec.BlockAttrsSpec`
  when decoding an HCL body using `hcldec`. In that case, the attributes
  with that type constraint will be required to be valid type constraint
  expression syntax and the result will be a `TypeConstraintType` value.

Note that the special handling of these arguments means that an argument
marked in this way must use the type constraint syntax directly. It is not
valid to pass in a value of `TypeConstraintType` that has been obtained
dynamically via some other expression result.

`TypeConstraintType` is provided with the intent of using it internally within
application code when incorporating type constraint expression syntax into
an HCL-based language, not to be used for dynamic "programming with types". A
calling application could support programming with types by defining its _own_
capsule type, but that is not the purpose of `TypeConstraintType`.

## The "convert" `cty` Function

Building on the `TypeConstraintType` described in the previous section, this
package also provides `typeexpr.ConvertFunc` which is a cty function that
can be placed into a `cty.EvalContext` (conventionally named "convert") in
order to provide a general type conversion function in an HCL-based language:

```hcl
  foo = convert("true", bool)
```

The second parameter uses the mechanism described in the previous section to
require its argument to be a type constraint expression rather than a value
expression. In doing so, it allows converting with any type constraint that
can be expressed in this package's type constraint syntax. In the above example,
the `foo` argument would receive a boolean true, or `cty.True` in `cty` terms.

The target type constraint must always be provided statically using inline
type constraint syntax. There is no way to _dynamically_ select a type
constraint using this function.
