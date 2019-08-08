---
layout: "commands-providers"
page_title: "Command: providers schema"
sidebar_current: "docs-commands-providers-schema"
description: |-
  The `terraform providers schema` command prints detailed schemas for the providers used
  in the current configuration.
---

# Command: terraform providers schema

The `terraform providers schema` command is used to print detailed schemas for the providers used in the current configuration.

-> `terraform providers schema` requires **Terraform v0.12 or later**.

## Usage

Usage: `terraform providers schema [options]`

The list of available flags are:

* `-json` - Displays the schemas in a machine-readble, JSON format.

Please note that, at this time, the `-json` flag is a _required_ option. In future releases, this command will be extended to allow for additional options. 

-> **Note:** The output includes a `format_version` key, which currently has major version zero to indicate that the format is experimental and subject to change. A future version will assign a non-zero major version and make stronger promises about compatibility. We do not anticipate any significant breaking changes to the format before its first major version, however.

## Format Summary

The following sections describe the JSON output format by example, using a pseudo-JSON notation.
Important elements are described with comments, which are prefixed with //.
To avoid excessive repetition, we've split the complete format into several discrete sub-objects, described under separate headers. References wrapped in angle brackets (like `<block-representation>`) are placeholders which, in the real output, would be replaced by an instance of the specified sub-object.

The JSON output format consists of the following objects and sub-objects:

- [Providers Schema Representation](#providers-schema-representation) - the top-level object returned by `terraform providers schema -json`
- [Schema Representation](#schema-representation) - a sub-object of providers, resources, and data sources that describes their schema
- [Block Representation](#block-representation) - a sub-object of schemas that describes attributes and nested blocks

## Providers Schema Representation 

```javascript
{
  "format_version": "0.1",
  
  // "provider_schemas" describes the provider schemas for all 
  // providers throughout the configuration tree. 
  "provider_schemas": {
    // keys in this map are the provider type, such as "random"
    "example_provider_name": {
      // "provider" is the schema for the provider configuration
      "provider": <schema-representation>,
    
      // "resource_schemas" map the resource type name to the resource's schema
      "resource_schemas": {
        "example_resource_name": <schema-representation>
      },

      // "data_source_schemas" map the data source type name to the
      // data source's schema
      "data_source_schemas": {
        "example_datasource_name": <schema-representation>,
      }
    },
    "example_provider_two": { … }
  }
}
```

## Schema Representation

A schema representation pairs a provider or resource schema (in a "block") with that schema's version.

```javascript
{
  // "version" is the schema version, not the provider version
  "version": int64,
  "block": <block-representation>
}
```

## Block Representation

A block representation contains "attributes" and "block_types" (which represent nested blocks).

```javascript
{
  // "attributes" describes any attributes that appear directly inside the 
  // block. Keys in this map are the attribute names.
  "attributes":  {
    "example_attribute_name": {
      // "type" is a representation of a type specification
      // that the attribute's value must conform to.
      "type": "string",

      // "description" is an English-language description of 
      // the purpose and usage of the attribute.
      "description": "string",

      // "required", if set to true, specifies that an 
      // omitted or null value is not permitted.
      "required": bool,

      // "optional", if set to true, specifies that an 
      // omitted or null value is permitted.  
      "optional": bool,

      // "computed", if set to true, indicates that the 
      // value comes from the provider rather than the 
      // configuration.
      "computed": bool,

      // "sensitive", if set to true, indicates that the
      // attribute may contain sensitive information.
      "sensitive": bool
    },
  },
  // "block_types" describes any nested blocks that appear directly
  // inside the block.
  // Keys in this map are the names of the block_type.
  "block_types": { 
    "example_block_name": {	
      // "nesting_mode" describes the nesting mode for the 
      // child block, and can be one of the following:
      // 	single
      // 	list
      // 	set
      // 	map
    "nesting_mode": "list",
    "block": <block-representation>,

    // "min_items" and "max_items" set lower and upper 
    // limits on the number of child blocks allowed for 
    // the list and set modes. These are 
    // omitted for other modes. 
    "min_items": 1,
    "max_items": 3
  }
}
```
