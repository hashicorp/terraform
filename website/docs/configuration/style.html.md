---
layout: "docs"
page_title: "Style Conventions - Configuration Language"
sidebar_current: "docs-config-style"
description: |-
  The Terraform language has some idiomatic style conventions, which we
  recommend users always follow for consistency between files and modules
  written by different teams.
---

# Style Conventions

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language](../configuration-0-11/index.html).

The Terraform parser allows you some flexibility in how you lay out the
elements in your configuration files, but the Terraform language also has some
idiomatic style conventions which we recommend users always follow
for consistency between files and modules written by different teams.
Automatic source code formatting tools may apply these conventions
automatically.

* Indent two spaces for each nesting level.

* When multiple arguments with single-line values appear on consecutive lines
  at the same nesting level, align their equals signs:

    ```hcl
    ami           = "abc123"
    instance_type = "t2.micro"
    ```

* When both arguments and blocks appear together inside a block body,
  place all of the arguments together at the top and then place nested
  blocks below them. Use one blank line to separate the arguments from
  the blocks.

* Use empty lines to separate logical groups of arguments within a block.

* For blocks that contain both arguments and "meta-arguments" (as defined by
  the Terraform language semantics), list meta-arguments first
  and separate them from other arguments with one blank line. Place
  meta-argument blocks _last_ and separate them from other blocks with
  one blank line.

    ```hcl
    resource "aws_instance" "example" {
      count = 2 # meta-argument first

      ami           = "abc123"
      instance_type = "t2.micro"

      network_interface {
        # ...
      }

      lifecycle { # meta-argument block last
        create_before_destroy = true
      }
    }
    ```

* Top-level blocks should always be separated from one another by one
  blank line. Nested blocks should also be separated by blank lines, except
  when grouping together related blocks of the same type (like multiple
  `provisioner` blocks in a resource).

* Avoid separating multiple blocks of the same type with other blocks of
  a different type, unless the block types are defined by semantics to
  form a family.
  (For example: `root_block_device`, `ebs_block_device` and
  `ephemeral_block_device` on `aws_instance` form a family of block types
  describing AWS block devices, and can therefore be grouped together and
  mixed.)

