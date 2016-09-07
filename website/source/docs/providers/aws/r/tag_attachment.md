---
layout: "aws"
page_title: "AWS: aws_tag_attachment"
sidebar_current: "docs-aws-resource-tag-attachment"
description: |-
  Attaches tags to an existing EC2 resource.
---

# aws\_tag\_attachment

Attaches tags to an existing EC2 resource.

This is useful when the tags of a resource need to depend on computed attributes
of the resource, for example a VPC tagged with its own VPC ID.

~> **NOTE** If you use `tags` on a resource, that resource will assume full
ownership of its tags and treat tags created by a `tag_attachment` as drift. For
this reason, a single resource must not have both its own `tags` and a
`tag_attachment` managing its tags.

## Example Usage

```
provider "aws" {
    region = "us-east-1"
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_tag_attachment" "main" {
  resource = "${aws_vpc.main.id}"
  tags {
    ID = "${aws_vpc.main.id}"
  }
}
```

## Argument Reference

The following arguments are supported:

* `resource` - (Required) The ID of any EC2 resource that supports tags.
* `tags` - (Required) A mapping of tags to assign to the resource.
