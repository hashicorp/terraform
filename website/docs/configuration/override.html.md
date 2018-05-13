---
layout: "docs"
page_title: "Override Files"
sidebar_current: "docs-config-override"
description: |-
  Override files allow additional settings to be merged into existing
  configuration objects.
---

# Override Files

Terraform normally loads all of the `.tf` and `.tf.json` files within a
directory and expects each one to define a distinct set of configuration
objects. If two files attempt to define the same object, Terraform returns
an error.

In some rare cases, it is convenient to be able to override specific portions
of an existing configuration object in a separate file. For example, a
human-edited configuration file in the Terraform language native syntax
could be partially overridden using a programmatically-generated file
in JSON syntax.

For these rare situations, Terraform has special handling of any configuration
file whose name ends in `_override.tf` or `_override.tf.json`. This special
handling also applies to a file named literally `override.tf` or
`override.tf.json`.

Terraform initially skips these _override files_ when loading configuration,
and then afterwards processes each one in turn (in lexicographical order). For
each top-level block defined in an override file, Terraform attempts to find
an already-defined object corresponding to that block and then merges the
override block contents into the existing object.

Use override files only in special circumstances. Over-use of override files
hurts readability, since a reader looking only at the original files cannot
easily see that some portions of those files have been overridden without
consulting all of the override files that are present. When using override
files, use comments in the original files to warn future readers about which
override files apply changes to each block.

## Example

If you have a Terraform configuration `example.tf` with the following contents:

```hcl
resource "aws_instance" "web" {
  instance_type = "t2.micro"
  ami           = "ami-408c7f28"
}
```

...and you created a file `override.tf` containing the following:

```hcl
resource "aws_instance" "web" {
  ami = "foo"
}
```

Terraform will merge the latter into the former, behaving as if the original
configuration had been as follows:

```hcl
resource "aws_instance" "web" {
  instance_type = "t2.micro"
  ami           = "foo"
}
```

## Merging Behavior

The merging behavior is slightly different for each block type, and some
special constructs within certain blocks are merged in a special way.

The general rule, which applies in most cases, is:

* A top-level block in an override file merges with a block in a normal
  configuration file that has the same block header. The block _header_ is the
  block type and any quoted labels that follow it.

* Within a top-level block, an attribute argument within an override block
  replaces any argument of the same name in the original block.

* Within a top-level block, any nested blocks within an override block replace
  _all_ blocks of the same type in the original block. Any block types that
  do not appear in the override block remain from the original block.

* The contents of nested configuration blocks are not merged.

* The resulting _merged block_ must still comply with any validation rules
  that apply to the given block type.

If more than one override file defines the same top-level block, the overriding
effect is compounded, with later blocks taking precedence over earlier blocks.
Overrides are processed in order first by filename (in lexicographical order)
and then by position in each file.

The following sections describe the special merging behaviors that apply to
specific arguments within certain top-level block types.

### Merging `resource` and `data` blocks

Within a `resource` block, the contents of any `lifecycle` nested block are
merged on an argument-by-argument basis. For example, if an override block
sets only the `create_before_destroy` argument then any `ignore_changes`
argument in the original block will be preserved.

If an overriding `resource` block contains one or more `provisioner` blocks
then any `provisioner` blocks in the original block are ignored.

If an overriding `resource` block contains a `connection` block then it
completely overrides any `connection` block present in the original block.

The `depends_on` meta-argument may not be used in override blocks, and will
produce an error.

### Merging `variable` blocks

The arguments within a `variable` block are merged in the standard way
described above, but some special considerations apply due to the interactions
between the `type` and `default` arguments.

If the original block defines a `default` value and an override block changes
the variable's `type`, Terraform attempts to convert the default value to
the overridden type, producing an error if this conversion is not possible.

Conversely, if the original block defines a `type` and an override block changes
the `default`, the overridden default value must be compatible with the
original type specification.

### Merging `output` blocks

The `depends_on` meta-argument may not be used in override blocks, and will
produce an error.

### Merging `locals` blocks

Each `locals` block defines a number of named values. Overrides are applied
on a value-by-value basis, ignoring which `locals` block they are defined in.

### Merging `terraform` blocks

The settings within `terraform` blocks are considered individually when
merging.

If the `required_providers` argument is set, its value is merged on an
element-by-element basis, which allows an override block to adjust the
constraint for a single provider without affecting the constraints for
other providers.

In both the `required_version` and `required_providers` settings, each override
constraint entirely replaces the constraints for the same component in the
original block. If both the base block and the override block both set
`required_version` then the constraints in the base block are entirely ignored.
