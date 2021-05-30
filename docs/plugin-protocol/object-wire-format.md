# Wire Format for Terraform Objects and Associated Values

The provider wire protocol (as of major version 5) includes a protobuf message
type `DynamicValue` which Terraform uses to represent values from the Terraform
Language type system, which result from evaluating the content of `resource`,
`data`, and `provider` blocks, based on a schema defined by the corresponding
provider.

Because the structure of these values is determined at runtime, `DynamicValue`
uses one of two possible dynamic serialization formats for the values
themselves: MessagePack or JSON. Terraform most commonly uses MessagePack,
because it offers a compact binary representation of a value. However, a server
implementation of the provider protocol should fall back to JSON if the
MessagePack field is not populated, in order to support both formats.

The remainder of this document describes how Terraform translates from its own
type system into the type system of the two supported serialization formats.
A server implementation of the Terraform provider protocol can use this
information to decode `DynamicValue` values from incoming messages into
whatever representation is convenient for the provider implementation.

A server implementation must also be able to _produce_ `DynamicValue` messages
as part of various response messages. When doing so, servers should always
use MessagePack encoding, because Terraform does not consistently support
JSON responses across all request types and all Terraform versions.

Both the MessagePack and JSON serializations are driven by information the
provider previously returned in a `Schema` message. Terraform will encode each
value depending on the type constraint given for it in the corresponding schema,
using the closest possible MessagePack or JSON type to the Terraform language
type. Therefore a server implementation can decode a serialized value using a
standard MessagePack or JSON library and assume it will conform to the
serialization rules described below.

## MessagePack Serialization Rules

The MessagePack types referenced in this section are those defined in
[The MessagePack type system specification](https://github.com/msgpack/msgpack/blob/master/spec.md#type-system).

Note that MessagePack defines several possible serialization formats for each
type, and Terraform may choose any of the formats of a specified type.
The exact serialization chosen for a given value may vary between Terraform
versions, but the types given here are contractual.

Conversely, server implementations that are _producing_ MessagePack-encoded
values are free to use any of the valid serialization formats for a particular
type. However, we recommend choosing the most compact format that can represent
the value without a loss of range.

### `Schema.Block` Mapping Rules for MessagePack

To represent the content of a block as MessagePack, Terraform constructs a
MessagePack map that contains one key-value pair per attribute and one
key-value pair per distinct nested block described in the `Schema.Block` message.

The key-value pairs representing attributes have values based on
[the `Schema.Attribute` mapping rules](#Schema.Attribute-mapping-rules-for-messagepack).
The key-value pairs representing nested block types have values based on
[the `Schema.NestedBlock` mapping rules](#Schema.NestedBlock-mapping-rules-for-messagepack).

### `Schema.Attribute` Mapping Rules for MessagePack

The MessagePack serialization of an attribute value depends on the value of the
`type` field of the corresponding `Schema.Attribute` message. The `type` field is
a compact JSON serialization of a
[Terraform type constraint](https://www.terraform.io/docs/configuration/types.html),
which consists either of a single
string value (for primitive types) or a two-element array giving a type kind
and a type argument.

The following table describes the type-specific mapping rules. Along with those
type-specific rules there are two special rules that override the mappings
in the table below, regardless of type:

* A null value is represented as a MessagePack nil value.
* An unknown value (that is, a placeholder for a value that will be decided
  only during the apply operation) is represented as a
  [MessagePack extension](https://github.com/msgpack/msgpack/blob/master/spec.md#extension-types)
  value whose type identifier is zero and whose value is unspecified and
  meaningless.

| `type` Pattern | MessagePack Representation |
|---|---|
| `"string"` | A MessagePack string containing the Unicode characters from the string value serialized as normalized UTF-8. |
| `"number"` | Either MessagePack integer, MessagePack float, or MessagePack string representing the number. If a number is represented as a string then the string contains a decimal representation of the number which may have a larger mantissa than can be represented by a 64-bit float. |
| `"bool"` | A MessagePack boolean value corresponding to the value. |
| `["list",T]` | A MessagePack array with the same number of elements as the list value, each of which is represented by the result of applying these same mapping rules to the nested type `T`. |
| `["set",T]` | Identical in representation to `["list",T]`, but the order of elements is undefined because Terraform sets are unordered. |
| `["map",T]` | A MessagePack map with one key-value pair per element of the map value, where the element key is serialized as the map key (always a MessagePack string) and the element value is represented by a value constructed by applying these same mapping rules to the nested type `T`. |
| `["object",ATTRS]` | A MessagePack map with one key-value pair per attribute defined in the `ATTRS` object. The attribute name is serialized as the map key (always a MessagePack string) and the attribute value is represented by a value constructed by applying these same mapping rules to each attribute's own type. |
| `["tuple",TYPES]` | A MessagePack array with one element per element described by the `TYPES` array. The element values are constructed by applying these same mapping rules to the corresponding element of `TYPES`. |
| `"dynamic"` | A MessagePack array with exactly two elements. The first element is a MessagePack binary value containing a JSON-serialized type constraint in the same format described in this table. The second element is the result of applying these same mapping rules to the value with the type given in the first element. This special type constraint represents values whose types will be decided only at runtime. |

### `Schema.NestedBlock` Mapping Rules for MessagePack

The MessagePack serialization of a collection of blocks of a particular type
depends on the `nesting` field of the corresponding `Schema.NestedBlock` message.
The `nesting` field is a value from the `Schema.NestingBlock.NestingMode`
enumeration.

All `nesting` values cause the individual blocks of a type to be represented
by applying
[the `Schema.Block` mapping rules](#Schema.Block-mapping-rules-for-messagepack)
to the block's contents based on the `block` field, producing what we'll call
a _block value_ in the table below.

The `nesting` value then in turn defines how Terraform will collect all of the
individual block values together to produce a single property value representing
the nested block type. For all `nesting` values other than `MAP`, blocks may
not have any labels. For the `nesting` value `MAP`, blocks must have exactly
one label, which is a string we'll call a _block label_ in the table below.

| `nesting` Value | MessagePack Representation |
|---|---|
| `SINGLE` | The block value of the single block of this type, or nil if there is no block of that type. |
| `LIST` | A MessagePack array of all of the block values, preserving the order of definition of the blocks in the configuration. |
| `SET` | A MessagePack array of all of the block values in no particular order. |
| `MAP` | A MessagePack map with one key-value pair per block value, where the key is the block label and the value is the block value. |
| `GROUP` | The same as with `SINGLE`, except that if there is no block of that type Terraform will synthesize a block value by pretending that all of the declared attributes are null and that there are zero blocks of each declared block type. |

For the `LIST` and `SET` nesting modes, Terraform guarantees that the
MessagePack array will have a number of elements between the `min_items` and
`max_items` values given in the schema, _unless_ any of the block values contain
nested unknown values. When unknown values are present, Terraform considers
the value to be potentially incomplete and so Terraform defers validation of
the number of blocks. For example, if the configuration includes a `dynamic`
block whose `for_each` argument is unknown then the final number of blocks is
not predictable until the apply phase.

## JSON Serialization Rules

The JSON serialization is a secondary representation for `DynamicValue`, with
MessagePack preferred due to its ability to represent unknown values via an
extension.

The JSON encoding described in this section is also used for the `json` field
of the `RawValue` message that forms part of an `UpgradeResourceState` request.
However, in that case the data is serialized per the schema of the provider
version that created it, which won't necessarily match the schema of the
_current_ version of that provider.

### `Schema.Block` Mapping Rules for JSON

To represent the content of a block as JSON, Terraform constructs a
JSON object that contains one property per attribute and one property per
distinct nested block described in the `Schema.Block` message.

The properties representing attributes have property values based on
[the `Schema.Attribute` mapping rules](#Schema.Attribute-mapping-rules-for-json).
The properties representing nested block types have property values based on
[the `Schema.NestedBlock` mapping rules](#Schema.NestedBlock-mapping-rules-for-json).

### `Schema.Attribute` Mapping Rules for JSON

The JSON serialization of an attribute value depends on the value of the `type`
field of the corresponding `Schema.Attribute` message. The `type` field is
a compact JSON serialization of a
[Terraform type constraint](https://www.terraform.io/docs/configuration/types.html),
which consists either of a single
string value (for primitive types) or a two-element array giving a type kind
and a type argument.

The following table describes the type-specific mapping rules. Along with those
type-specific rules there is one special rule that overrides the rules in the
table regardless of type:

* A null value is always represented as JSON `null`.

| `type` Pattern | JSON Representation |
|---|---|
| `"string"` | A JSON string containing the Unicode characters from the string value. |
| `"number"` | A JSON number representing the number value. Terraform numbers are arbitrary-precision floating point, so the value may have a larger mantissa than can be represented by a 64-bit float. |
| `"bool"` | Either JSON `true` or JSON `false`, depending on the boolean value. |
| `["list",T]` | A JSON array with the same number of elements as the list value, each of which is represented by the result of applying these same mapping rules to the nested type `T`. |
| `["set",T]` | Identical in representation to `["list",T]`, but the order of elements is undefined because Terraform sets are unordered. |
| `["map",T]` | A JSON object with one property per element of the map value, where the element key is serialized as the property name string and the element value is represented by a property value constructed by applying these same mapping rules to the nested type `T`. |
| `["object",ATTRS]` | A JSON object with one property per attribute defined in the `ATTRS` object. The attribute name is serialized as the property name string and the attribute value is represented by a property value constructed by applying these same mapping rules to each attribute's own type. |
| `["tuple",TYPES]` | A JSON array with one element per element described by the `TYPES` array. The element values are constructed by applying these same mapping rules to the corresponding element of `TYPES`. |
| `"dynamic"` | A JSON object with two properties: `"type"` specifying one of the `type` patterns described in this table in-band, giving the exact runtime type of the value, and `"value"` specifying the result of applying these same mapping rules to the table for the specified runtime type. This special type constraint represents values whose types will be decided only at runtime. |

### `Schema.NestedBlock` Mapping Rules for JSON

The JSON serialization of a collection of blocks of a particular type depends
on the `nesting` field of the corresponding `Schema.NestedBlock` message.
The `nesting` field is a value from the `Schema.NestingBlock.NestingMode`
enumeration.

All `nesting` values cause the individual blocks of a type to be represented
by applying
[the `Schema.Block` mapping rules](#Schema.Block-mapping-rules-for-json)
to the block's contents based on the `block` field, producing what we'll call
a _block value_ in the table below.

The `nesting` value then in turn defines how Terraform will collect all of the
individual block values together to produce a single property value representing
the nested block type. For all `nesting` values other than `MAP`, blocks may
not have any labels. For the `nesting` value `MAP`, blocks must have exactly
one label, which is a string we'll call a _block label_ in the table below.

| `nesting` Value | JSON Representation |
|---|---|
| `SINGLE` | The block value of the single block of this type, or `null` if there is no block of that type. |
| `LIST` | A JSON array of all of the block values, preserving the order of definition of the blocks in the configuration. |
| `SET` | A JSON array of all of the block values in no particular order. |
| `MAP` | A JSON object with one property per block value, where the property name is the block label and the value is the block value. |
| `GROUP` | The same as with `SINGLE`, except that if there is no block of that type Terraform will synthesize a block value by pretending that all of the declared attributes are null and that there are zero blocks of each declared block type. |

For the `LIST` and `SET` nesting modes, Terraform guarantees that the JSON
array will have a number of elements between the `min_items` and `max_items`
values given in the schema.
