---
layout: "docs"
page_title: "Internals: JSON Output Format"
sidebar_current: "docs-internals-json"
description: |-
  Terraform provides a machine-readable JSON representation of state, configuration and plan.
---

# JSON Output Format

-> **Note:** This format is available in Terraform 0.12 and later.

When Terraform plans to make changes, it prints a human-readable summary to the terminal. It can also, when run with `-out=<PATH>`, write a much more detailed binary plan file, which can later be used to apply those changes.

Since the format of plan files isn't suited for use with external tools (and likely never will be), Terraform can output a machine-readable JSON representation of a plan file's changes. It can also convert state files to the same format, to simplify data loading and provide better long-term compatibility.

Use `terraform show -json <FILE>` to generate a JSON representation of a plan or state file. See [the `terraform show` documentation](/docs/commands/show.html) for more details.

-> **Note:** The output includes a `format_version` key, which currently has major version zero to indicate that the format is experimental and subject to change. A future version will assign a non-zero major version and make stronger promises about compatibility. We do not anticipate any significant breaking changes to the format before its first major version, however.

## Format Summary

The following sections describe the JSON output format by example, using a pseudo-JSON notation.

Important elements are described with comments, which are prefixed with `//`.

To avoid excessive repetition, we've split the complete format into several discrete sub-objects, described under separate headers. References wrapped in angle brackets (like `<values-representation>`) are placeholders which, in the real output, would be replaced by an instance of the specified sub-object.

The JSON output format consists of the following objects and sub-objects:

- [State Representation](#state-representation) — The complete top-level object returned by `terraform show -json <STATE FILE>`.
- [Plan Representation](#plan-representation) — The complete top-level object returned by `terraform show -json <PLAN FILE>`.
- [Values Representation](#values-representation) — A sub-object of both plan and state output that describes current state or planned state.
- [Configuration Representation](#configuration-representation) — A sub-object of plan output that describes a parsed Terraform configuration.
    - [Expression Representation](#expression-representation) — A sub-object of a configuration representation that describes an unevaluated expression.
    - [Block Expressions Representation](#block-expressions-representation) — A sub-object of a configuration representation that describes the expressions nested inside a block.
- [Change Representation](#change-representation) — A sub-object of plan output that describes planned changes to an object.

## State Representation

Because state does not currently have any significant metadata not covered by the common values representation ([described below](#values-representation)), the `<state-representation>` is straightforward:

```javascript
{
  // "values" is a values representation object derived from the values in the
  // state. Because the state is always fully known, this is always complete.
  "values": <values-representation>

  "terraform_version": "version.string"
}
```

The extra wrapping object here will allow for any extension we may need to add in future versions of this format.

## Plan Representation

A plan consists of a prior state, the configuration that is being applied to that state, and the set of changes Terraform plans to make to achieve that.

For ease of consumption by callers, the plan representation includes a partial representation of the values in the final state (using a [value representation](#value-representation)), allowing callers to easily analyze the planned outcome using similar code as for analyzing the prior state.

```javascript
{
  "format_version": "0.1",

  // "prior_state" is a representation of the state that the configuration is
  // being applied to, using the state representation described above.
  "prior_state": <state-representation>,

  // "config" is a representation of the configuration being applied to the
  // prior state, using the configuration representation described above.
  "config": <config-representation>,

  // "planned_values" is a description of what is known so far of the outcome in
  // the standard value representation, with any as-yet-unknown values omitted.
  "planned_values": <values-representation>,

  // "proposed_unknown" is a representation of the attributes, including any
  // potentially-unknown attributes. Each value is replaced with "true" or
  // "false" depending on whether it is known in the proposed plan.
  "proposed_unknown": <values-representation>,

  // "variables" is a representation of all the variables provided for the given
  // plan. This is structured as a map similar to the output map so we can add
  // additional fields in later.
  "variables": {
    "varname": {
      "value": "varvalue"
    },
  },

  // "changes" is a description of the individual change actions that Terraform
  // plans to use to move from the prior state to a new state matching the
  // configuration.
  "resource_changes": [
    // Each element of this array describes the action to take
    // for one instance object. All resources in the
    // configuration are included in this list.
    {
      // "address" is the full absolute address of the resource instance this
      // change applies to, in the same format as addresses in a value
      // representation
      "address": "module.child.aws_instance.foo[0]",

      // "module_address", if set, is the module portion of the above address.
      // Omitted if the instance is in the root module.
      "module_address": "module.child",

      // "mode", "type", "name", and "index" have the same meaning as in a
      // value representation.
      "mode": "managed",
      "type": "aws_instance",
      "name": "foo",
      "index": 0,

      // "deposed", if set, indicates that this action applies to a "deposed"
      // object of the given instance rather than to its "current" object.
      // Omitted for changes to the current object. "address" and "deposed"
      // together form a unique key across all change objects in a particular
      // plan. The value is an opaque key representing the specific deposed
      // object.
      "deposed": "deadbeef",

      // "change" describes the change that will be made to the indicated
      // object. The <change-representation> is detailed in a section below.
      "change": <change-representation>
    }
  ],

  // "output_changes" describes the planned changes to the output values of the
  // root module.
  "output_changes": {
    // Keys are the defined output value names.
    "foo": {

      // "change" describes the change that will be made to the indicated output
      // value, using the same representation as for resource changes except
      // that the only valid actions values are:
      //   ["create"]
      //   ["update"]
      //   ["delete"]
      // In the Terraform CLI 0.12.0 release, Terraform is not yet fully able to
      // track changes to output values, so the actions indicated may not be
      // fully accurate, but the "after" value will always be correct.
      "change": <change-representation>,
    }
  }
}
```

This overall plan structure, fully expanded, is what will be printed by the `terraform show -json <planfile>` command.

## Values Representation

A values representation is used in both state and plan output to describe current state (which is always complete) and planned state (which omits values not known until apply).

The following example illustrates the structure of a `<values-representation>`:

```javascript
{
  // "outputs" describes the outputs from the root module. Outputs from
  // descendent modules are not available because they are not retained in all
  // of the underlying structures we will build this values representation from.
  "outputs": {
    "private_ip": {
      "value": "192.168.3.2",
      "sensitive": false
    }
  },

  // "root_module" describes the resources and child modules in the root module.
  "root_module": {
    "resources": [
      {
        // "address" is the absolute resource address, which callers must consider
        // opaque but may do full string comparisons with other address strings or
        // pass this verbatim to other Terraform commands that are documented to
        // accept absolute resource addresses. The module-local portions of this
        // address are extracted in other properties below.
        "address": "aws_instance.example[1]",

        // "mode" can be "managed", for resources, or "data", for data resources
        "mode": "managed",
        "type": "aws_instance",
        "name": "example",

        // If the count or for_each meta-arguments are set for this resource, the
        // additional key "index" is present to give the instance index key. This
        // is omitted for the single instance of a resource that isn't using count
        // or for_each.
        "index": 1,

        // "provider_name" is the name of the provider that is responsible for
        // this resource. This is only the provider name, not a provider
        // configuration address, and so no module path nor alias will be
        // indicated here. This is included to allow the property "type" to be
        // interpreted unambiguously in the unusual situation where a provider
        // offers a resource type whose name does not start with its own name,
        // such as the "googlebeta" provider offering "google_compute_instance".
        "provider_name": "aws",

        // "schema_version" indicates which version of the resource type schema
        // the "values" property conforms to.
        "schema_version": 2,

        // "values" is the JSON representation of the attribute values of the
        // resource, whose structure depends on the resource type schema. Any
        // unknown values are omitted or set to null, making them
        // indistinguishable from absent values; callers which need to distinguish
        // unknown from unset must use the plan-specific or config-specific
        // structures described in later sections.
        "values": {
          "id": "i-abc123",
          "instance_type": "t2.micro",
          // etc, etc
        }
      }
    ]

    "child_modules": [
      // Each entry in "child_modules" has the same structure as the root_module
      // object, with the additional "address" property shown below.
      {
        // "address" is the absolute module address, which callers must treat as
        // opaque but may do full string comparisons with other module address
        // strings and may pass verbatim to other Terraform commands that are
        // documented as accepting absolute module addresses.
        "address": "module.child",

        // "resources" is the same as in "root_module" above
        "resources": [
            {
              "address": "module.child.aws_instance.foo",
              // etc, etc
            }
        ],

        // Each module object can optionally have its own
        // nested "child_modules", recursively describing the
        // full module tree.
        "child_modules": [ ... ],
      }
    ]
  }
}
```

The translation of attribute and output values is the same intuitive mapping from HCL types to JSON types used by Terraform's [`jsonencode`](/docs/configuration/functions/jsonencode.html) function. This mapping does lose some information: lists, sets, and tuples all lower to JSON arrays while maps and objects both lower to JSON objects. Unknown values and null values are both treated as absent or null.

Only the "current" object for each resource instance is described. "Deposed" objects are not reflected in this structure at all; in plan representations, you can refer to the change representations for further details.

The intent of this structure is to give a caller access to a similar level of detail as is available to expressions within the configuration itself. This common representation is not suitable for all use-cases because it loses information compared to the data structures it is built from. For more complex needs, use the more elaborate changes and configuration representations.

## Configuration Representation

Configuration is the most complicated structure in Terraform, since it includes unevaluated expression nodes and other complexities.

Because the configuration models are produced at a stage prior to expression evaluation, it is not possible to produce a values representation for configuration. Instead, we describe the physical structure of the configuration, giving access to constant values where possible and allowing callers to analyze any references to other objects that are present:

```javascript
{
  // "provider_configs" describes all of the provider configurations throughout
  // the configuration tree, flattened into a single map for convenience since
  // provider configurations are the one concept in Terraform that can span
  // across module boundaries.
  "provider_configs": {

    // Keys in the provider_configs map are to be considered opaque by callers,
    // and used just for lookups using the "provider_config_key" property in each
    // resource object.
    "opaque_provider_ref_aws": {

      // "name" is the name of the provider without any alias
      "name": "aws",

      // "alias" is the alias set for a non-default configuration, or unset for
      // a default configuration.
      "alias": "foo",

      // "module_address" is included only for provider configurations that are
      // declared in a descendent module, and gives the opaque address for the
      // module that contains the provider configuration.
      "module_address": "module.child",

      // "expressions" describes the provider-specific content of the
      // configuration block, as a block expressions representation (see section
      // below).
      "expressions": <block-expressions-representation>
    }
  },

  // "root_module" describes the root module in the configuration, and serves
  // as the root of a tree of similar objects describing descendent modules.
  "root_module": {

    // "outputs" describes the output value configurations in the module.
    "outputs": {

      // Property names here are the output value names
      "example": {
        "expression": <expression-representation>,
        "sensitive": false
      }
    },

    // "resources" describes the "resource" and "data" blocks in the module
    // configuration.
    "resources": [
      {
        // "address" is the opaque absolute address for the resource itself.
        "address": "aws_instance.example",

        // "mode", "type", and "name" have the same meaning as for the resource
        // portion of a value representation.
        "mode": "managed",
        "type": "aws_instance",
        "name": "example",

        // "provider_config_key" is the key into "provider_configs" (shown
        // above) for the provider configuration that this resource is
        // associated with.
        "provider_config_key": "opaque_provider_ref_aws",

        // "provisioners" is an optional field which describes any provisioners.
        // Connection info will not be included here.
        "provisioners": [
          {
            "type": "local-exec",

            // "expressions" describes the provisioner configuration
            "expressions": <block-expressions-representation>
          },
        ],

        // "expressions" describes the resource-type-specific content of the
        // configuration block.
        "expressions": <block-expressions-representation>,

        // "schema_version" is the schema version number indicated by the
        // provider for the type-specific arguments described in "expressions".
        "schema_version": 2,

        // "count_expression" and "for_each_expression" describe the expressions
        // given for the corresponding meta-arguments in the resource
        // configuration block. These are omitted if the corresponding argument
        // isn't set.
        "count_expression": <expression-representation>,
        "for_each_expression": <expression-representation>
      },
    ],

    // "module_calls" describes the "module" blocks in the module. During
    // evaluation, a module call with count or for_each may expand to multiple
    // module instances, but in configuration only the block itself is
    // represented.
    "module_calls": {

      // Key is the module call name chosen in the configuration.
      "child": {

        // "resolved_source" is the resolved source address of the module, after
        // any normalization and expansion. This could be either a
        // go-getter-style source address or a local path starting with "./" or
        // "../". If the user gave a registry source address then this is the
        // final location of the module as returned by the registry, after
        // following any redirect indirection.
        "resolved_source": "./child"

        // "expressions" describes the expressions for the arguments within the
        // block that correspond to input variables in the child module.
        "expressions": <block-expressions-representation>,

        // "count_expression" and "for_each_expression" describe the expressions
        // given for the corresponding meta-arguments in the module
        // configuration block. These are omitted if the corresponding argument
        // isn't set.
        "count_expression": <expression-representation>,
        "for_each_expression": <expression-representation>,

        // "module" is a representation of the configuration of the child module
        // itself, using the same structure as the "root_module" object,
        // recursively describing the full module tree.
        "module": <module-config-representation>,
      }
    }
  }
}
```

### Expression Representation

Each unevaluated expression in the configuration is represented with an `<expression-representation>` object with the following structure:

```javascript
{
  // "constant_value" is set only if the expression contains no references to
  // other objects, in which case it gives the resulting constant value. This is
  // mapped as for the individual values in a value representation.
  "constant_value": "hello",

  // Alternatively, "references" will be set to a list of references in the
  // expression. Multi-step references will be unwrapped and duplicated for each
  // significant traversal step, allowing callers to more easily recognize the
  // objects they care about without attempting to parse the expressions.
  // Callers should only use string equality checks here, since the syntax may
  // be extended in future releases.
  "references": [
    "data.template_file.foo[1].vars[\"baz\"]",
    "data.template_file.foo[1].vars", // implied by previous
    "data.template_file.foo[1]", // implied by previous
    "data.template_file.foo", // implied by previous
    "module.foo.bar",
    "module.foo", // implied by the previous
    "var.example[0]",
    "var.example", // implied by the previous

    // Partial references like "data" and "module" are not included, because
    // Terraform considers "module.foo" to be an atomic reference, not an
    // attribute access.
  ]
}
```

### Block Expressions Representation

In some cases, it is the entire content of a block (possibly after certain special arguments have already been handled and removed) that must be represented. For that, we have an `<block-expressions-representation>` structure:

```javascript
{
  // Attribute arguments are mapped directly with the attribute name as key and
  // an <expression-representation> as value.
  "ami": <expression-representation>,
  "instance_type": <expression-representation>,

  // Nested block arguments are mapped as either a single nested
  // <block-expressions-representation> or an array object of these, depending on the
  // block nesting mode chosen in the schema.
  //  - "single" nesting is a direct <block-expressions-representation>
  //  - "list" and "set" produce arrays
  //  - "map" produces an object
  "root_block_device": <expression-representation>,
  "ebs_block_device": [
    <expression-representation>
  ]
}
```

For now we expect callers to just hard-code assumptions about the schemas of particular resource types in order to process these expression representations. In a later release we will add new inspection commands to return machine-readable descriptions of the schemas themselves, allowing for more generic handling in programs such as visualization tools.

## Change Representation

A `<change-representation>` describes the change that will be made to the indicated object.

```javascript
{
  // "actions" are the actions that will be taken on the object selected by the
  // properties below.
  // Valid actions values are:
  //    ["no-op"]
  //    ["create"]
  //    ["read"]
  //    ["update"]
  //    ["delete", "create"]
  //    ["create", "delete"]
  //    ["delete"]
  // The two "replace" actions are represented in this way to allow callers to
  // e.g. just scan the list for "delete" to recognize all three situations
  // where the object will be deleted, allowing for any new deletion
  // combinations that might be added in future.
  "actions": ["update"]

  // "before" and "after" are representations of the object value both before
  // and after the action. For ["create"] and ["delete"] actions, either
  // "before" or "after" is unset (respectively). For ["no-op"], the before and
  // after values are identical. The "after" value will be incomplete if there
  // are values within it that won't be known until after apply.
  "before": <value-representation>,
  "after": <value-representation>
}
```
