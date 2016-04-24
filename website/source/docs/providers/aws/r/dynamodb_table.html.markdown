---
layout: "aws"
page_title: "AWS: dynamodb_table"
sidebar_current: "docs-aws-resource-dynamodb-table"
description: |-
  Provides a DynamoDB table resource
---

# aws\_dynamodb\_table

Provides a DynamoDB table resource

## Example Usage

The following dynamodb table description models the table and GSI shown
in the [AWS SDK example documentation](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/GSI.html)

```
resource "aws_dynamodb_table" "basic-dynamodb-table" {
    name = "GameScores"
    read_capacity = 20
    write_capacity = 20
    hash_key = "UserId"
    range_key = "GameTitle"
    attribute {
      name = "UserId"
      type = "S"
    }
    attribute {
      name = "GameTitle"
      type = "S"
    }
    attribute {
      name = "TopScore"
      type = "N"
    }
    global_secondary_index {
      name = "GameTitleIndex"
      hash_key = "GameTitle"
      range_key = "TopScore"
      write_capacity = 10
      read_capacity = 10
      projection_type = "INCLUDE"
      non_key_attributes = [ "UserId" ]
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the table, this needs to be unique
  within a region.
* `read_capacity` - (Required) The number of read units for this table
* `write_capacity` - (Required) The number of write units for this table
* `hash_key` - (Required) The attribute to use as the hash key (the
  attribute must also be defined as an attribute record
* `range_key` - (Optional) The attribute to use as the range key (must
  also be defined)
* `attribute` - Define an attribute, has two properties:
  * `name` - The name of the attribute
  * `type` - One of: S, N, or B for (S)tring, (N)umber or (B)inary data
* `stream_enabled` - (Optional) Indicates whether Streams are to be enabled (true) or disabled (false).
* `stream_view_type` - (Optional) When an item in the table is modified, StreamViewType determines what information is written to the table's stream. Valid values are KEYS_ONLY, NEW_IMAGE, OLD_IMAGE, NEW_AND_OLD_IMAGES.
* `local_secondary_index` - (Optional) Describe an LSI on the table;
  these can only be allocated *at creation* so you cannot change this
definition after you have created the resource.
* `global_secondary_index` - (Optional) Describe a GSO for the table;
  subject to the normal limits on the number of GSIs, projected
attributes, etc.

For both `local_secondary_index` and `global_secondary_index` objects,
the following properties are supported:

* `name` - (Required) The name of the LSI or GSI
* `hash_key` - (Required for GSI) The name of the hash key in the index; must be
defined as an attribute in the resource. Only applies to
  `global_secondary_index`
* `range_key` - (Required) The name of the range key; must be defined
* `projection_type` - (Required) One of "ALL", "INCLUDE" or "KEYS_ONLY"
   where *ALL* projects every attribute into the index, *KEYS_ONLY*
    projects just the hash and range key into the index, and *INCLUDE*
    projects only the keys specified in the _non_key_attributes_
parameter.
* `non_key_attributes` - (Optional) Only required with *INCLUDE* as a
  projection type; a list of attributes to project into the index. These
do not need to be defined as attributes on the table.

For `global_secondary_index` objects only, you need to specify
`write_capacity` and `read_capacity` in the same way you would for the
table as they have separate I/O capacity.

### A note about attributes

Only define attributes on the table object that are going to be used as:

* Table hash key or range key
* LSI or GSI hash key or range key

The DynamoDB API expects attribute structure (name and type) to be
passed along when creating or updating GSI/LSIs or creating the initial
table. In these cases it expects the Hash / Range keys to be provided;
because these get re-used in numerous places (i.e the table's range key
could be a part of one or more GSIs), they are stored on the table
object to prevent duplication and increase consistency. If you add
attributes here that are not used in these scenarios it can cause an
infinite loop in planning.


## Attributes Reference

The following attributes are exported:

* `arn` - The arn of the table
* `id` - The name of the table
* `stream_arn` - The ARN of the Table Stream. Only available when `stream_enabled = true`

