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

For more information, see [the godoc reference](http://godoc.org/github.com/hashicorp/hcl2/ext/typeexpr).

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
