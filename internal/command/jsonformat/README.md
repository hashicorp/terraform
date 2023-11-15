# jsonformat

This package contains functionality around formatting and displaying the JSON
structured output produced by adding the `-json` flag to various Terraform
commands.

## Terraform Structured Plan Renderer

As of January 2023, this package contains only a single structure: the 
`Renderer`.

The renderer accepts the JSON structured output produced by the 
`terraform show <plan-file> -json` command and writes it in a human-readable
format.

Implementation details and decisions for the `Renderer` are discussed in the
following sections.

### Implementation

There are two subpackages within the `jsonformat` renderer package. The `differ`
package compares the `before` and `after` values of the given plan and produces
`Diff` objects from the `computed` package.

This approach is aimed at ensuring the process by which the plan difference is
calculated is separated from the rendering itself. In this way it should be 
possible to modify the rendering or add new renderer formats without being
concerned with the complex diff calculations.

#### The `differ` package

The `differ` package operates on `Change` objects. These are produced from
`jsonplan.Change` objects (which are produced by the `terraform show` command).
Each `jsonplan.Change` object represents a single resource within the overall
Terraform configuration.

The `differ` package will iterate through the `Change` objects and produce a 
single `Diff` that represents a processed summary of the changes described by 
the `Change`. You will see that the produced changes are nested so a change to a
list attribute will contain a slice of changes, this is discussed in the 
"[The computed package](#the-computed-package)" section.

##### The `Change` object

The `Change` objects contain raw Golang representations of JSON objects (generic
`interface{}` fields). These are produced by parsing the `json.RawMessage` 
objects within the provided changes.

The fields the differ cares about from the provided changes are:

- `Before`: The value before the proposed change.
- `After`: The value after the proposed change.
- `Unknown`: If the value is being computed during the change.
- `BeforeSensitive`: If the value was sensitive before the change.
- `AfterSensitive`: If the value is sensitive after the change.
- `ReplacePaths`: If the change is causing the overall resource to be replaced.

In addition, the changes define two additional meta fields that they set and
manipulate internally:

- `BeforeExplicit`: If the value in `Before` is explicit or an implied result due to a change elsewhere.
- `AfterExplicit`: If the value in `After` is explicit or an implied result due to a change elsewhere.

The actual concrete type of each of the generic fields is determined by the 
overall schema. The changes are also recursive, this means as we iterate through 
the `Change` we create relevant child values based on the schema for the given 
resource.

For example, the initial change is always a `block` type which means the 
`Before` and `After` values will actually be `map[string]interface{}` types 
mapping each attribute and block to their relevant values. The
`Unknown`, `BeforeSensitive`, `AfterSensitive` values will all be either a
`map[string]interface{}` which maps each attribute or nested block to their
unknown and sensitive status, or it could simply be a `boolean` which generally
means the entire block and all children are sensitive or computed.

In total, a `Change` can represent the following types:

- `Attribute`
  - `map`: Values will typically be `map[string]interface{}`.
  - `list`: Values will typically be `[]interface{}`.
  - `set`: Values will typically be `[]interface{}`.
  - `object`: Values will typically be `map[string]interface{}`.
  - `tuple`: Values will typically be `[]interface{}`.
  - `bool`: Values will typically be a `bool`.
  - `number`: Values will typically be a `float64`.
  - `string`: Values will typically be a `string`.
- `Block`: Values will typically be `map[string]interface{}`, but they can be
           split between nested blocks and attributes.
- `Output`
  - Outputs are interesting as we don't have a schema for them, as such they
    can be any JSON type.
  - We also use the Output type to represent dynamic attributes, since in both 
    cases we work out the type based on the JSON representation instead of the
    schema.

The `ReplacePaths` field is unique in that it's value doesn't actually change
based on the schema - it's always a slice of index slices. An index in this
context will either be an integer pointing to a child of a set or a list or a
string pointing to the child of a map, object or block. As we iterate through 
the value we manipulate the outer slice to remove child slices where the index
doesn't match and propagate paths that do match onto the children.

*Quick note on explicit vs implicit:* In practice, it is only possible to get 
implicit changes when you manipulate a collection. That is to say child values 
of a modified collection will insert `nil` entries into the relevant before 
or after fields of their child changes to represent their values being deleted
or created. It is also possible for users to explicitly put null values into 
their collections, and this behaviour is different to deleting an item in the
collection. With the `BeforeExplicit` and `AfterExplicit` values we can tell the
difference between whether this value was removed from a collection or this 
value was set to null in a collection.

*Quick note on the go-cty Value and Type objects:* The `Before` and `After` 
fields are actually go-cty values, but we cannot convert them directly because 
of the  Terraform Cloud redacted endpoint. The redacted endpoint turns sensitive
values into strings regardless of their types. Because of this, we cannot just 
do a direct conversion using the ctyjson package. We would have to iterate 
through the schema first, find the sensitive values and their mapped types, 
update the types inside the schema to strings, and then go back and do the 
overall conversion. This isn't including any of the more complicated parts 
around what happens if something was sensitive before and isn't sensitive after 
or vice versa. This would mean the type would need to change between the before 
and after value. It is in fact just easier to iterate through the values as 
generic JSON interfaces, and obfuscate the sensitive values as we never need to 
print them anyway.

##### Iterating through changes

The `differ` package will recursively create child `Change` objects for the 
complex objects.

There are two key subtypes of a `Change`: `SliceChange` and `MapChange`. 
`SliceChange` values are used by list, set, and tuple attributes. `MapChange` 
values are used by map and object attributes, and blocks. For what it is worth 
outputs and dynamic types can end up using both, but they're kind of special as 
the processing for dynamic types works out the type from the JSON struct and 
then just passes it into the relevant real types for actual processing.

The two subtypes implement `GetChild` functions that retrieve a child change
for a relevant index (`int` for slice, `string` for map). These functions build
an entirely populated `Change` object, and the package will then recursively 
compute the change for the child (and all other children). When a complex change
has all the children changes, it then passes that into the relevant complex 
diff type.

#### The `computed` package

A computed `Diff` should contain all the relevant information it needs to render 
itself.

The `Diff` itself contains the action (eg. `Create`, `Delete`, `Update`), and
whether this change is causing the overall resource to be replaced (read from 
the `ReplacePaths` field discussed in the previous section). The actual content 
of the diffs is passed directly into the internal renderer field. The internal
renderer is then an implementation that knows the actual content of the changes
and what they represent.

For example to instantiate a diff resulting from updating a list of 
primitives:

```go
    listDiff := computed.NewDiff(renderers.List([]computed.Diff{
        computed.NewDiff(renderers.Primitive(0.0, 0.0, cty.Number), plans.NoOp, false),
        computed.NewDiff(renderers.Primitive(1.0, nil, cty.Number), plans.Delete, false),
        computed.NewDiff(renderers.Primitive(nil, 4.0, cty.Number), plans.Create, false),
        computed.NewDiff(renderers.Primitive(2.0, 2.0, cty.Number), plans.NoOp, false)
    }, plans.Update, false))
```

##### The `RenderHuman` function

Currently, there is only one way to render a change, and it is implemented via
the `RenderHuman` function. In the future, there may be additional rendering 
capabilities, but for now the `RenderHuman` function just passes the call 
directly onto the internal renderer.

Rendering the above diff with: `listDiff.RenderHuman(0, RenderOpts{})` would
produce:

```text
[
    0,
  - 1 -> null,
  + 4, 
    2,
]    
```

Note, the render function itself doesn't print out metadata about its own change
(eg. there's no `~` symbol in front of the opening bracket). The expectation is
that parent changes control how child changes are rendered, so are responsible
for deciding on their opening indentation, whether they have a key (as in maps, 
objects, and blocks), or how the action symbol is displayed.

In the above example, the primitive renderer would print out only `1 -> null` 
while the surrounding list renderer is providing the indentation, the symbol and
the line ending commas.

##### Implementing new diff types

To implement a new diff type, you must implement the internal Renderer 
functionality. To do this you create a new implementation of the 
`computed.DiffRenderer`, make sure it accepts all the data you need, and 
implement the `RenderHuman` function (and any other additional render functions 
that may exist).

Some changes publish warnings that should be displayed alongside them. 
If your new change has no warnings you can use the `NoWarningsRenderer` to avoid
implementing the additional `Warnings` function.

If/when new Renderer types are implemented, additional `Render` like functions
will be added. You should implement all of these with your new change type.

##### Implementing new renderer types for changes

As of January 2023, there is only a single type of renderer (the human-readable)
renderer. As such, the `Diff` structure provides a single `RenderHuman` 
function.

To implement a new renderer:

1. Add a new render function onto the internal `DiffRenderer` interface.
2. Add a new render function onto the `Diff` struct that passes the call onto
   the internal renderer.
3. Implement the new function on all the existing internal interfaces.

Since each internal renderer contains all the information it needs to provide
change information about itself, your new Render function should pass in 
anything it needs.

### New types of Renderer

In the future, we may wish to add in different kinds of renderer, such as a 
compact renderer, or an interactive renderer. To do this, you'll need to modify
the Renderer struct or create a new type of Renderer.

The logic around creating the `Diff` structures will be shared (ie. calling 
into the differ package should be consistent across renderers). But when it 
comes to rendering the changes, I'd expect the `Diff` structures to implement
additional functions that allow them to internally organise the data as required
and return a relevant object. For the existing human-readable renderer that is 
simply a string, but for a future interactive renderer it might be a model from
an MVC pattern.
