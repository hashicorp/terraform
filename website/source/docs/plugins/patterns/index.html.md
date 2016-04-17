---
layout: "pluginpatterns"
page_title: "Plugin Design Patterns"
sidebar_current: "docs-plugins-patterns"
description: |-
  Common design patterns for writing Terraform plugins.
---

# Plugin Design Patterns

Terraform has the unenviable task of providing a unified user interface across
many different services and systems, built in very different ways, with
differing naming conventions, workflows, and concepts.

Plugin developers are faced with numerous decisions for how to map
externally-defined concepts into Terraform's model, and each plugin will have
its own unique challenges, but this section aims to ease that burden by
documenting some conventions and patterns that are used by existing plugins,
so those working on new plugins don't need to reinvent the wheel.

When multiple paths are available, consistency with existing usage is
preferable. However, each plugin is a little different and so deviations from
these patterns are expected and accepted when warranted.

The navigation bar includes links to detailed pages devoted to each of the
primary objects plugins can implement. The remainder of this page consists
of some general advice that can potentially apply to all plugins.

## Configuration Structures and Naming

Terraform plugins will define configuration blocks of various kinds, with
attributes whose names depend on what is being modelled.

Generally it is recommended to lean towards using the same names and structures
used by the underlying system as far as is reasonable, so users of that system
can easily map their knowledge of the system's concepts onto the Terraform
syntax. However, making the configuration feel "natural" within Terraform, by
following patterns around naming and data structures, can conversely help
users re-use their experience with one plugin to learn another, so a
good compromise between these two extremes is important.

Here is an example of a resource configuration block which illustrates some
general design patterns that can apply across all plugin object types:

```
resource "aws_instance" "example" {
  ami                    = "ami-408c7f28"
  instance_type          = "t1.micro"
  monitoring             = true
  vpc_security_group_ids = [
      "sg-1436abcf",
  ]
  tags                   = {
    Name        = "Application Server"
    Environment = "production"
  }

  root_block_device {
    delete_on_termination = false
  }
}
```

Attribute names within Terraform configuration blocks are conventionally named
as all-lowercase with underscores separating words, as shown above.

Simple single-value attributes, like `ami` and `instance_type` in the above
example, are given names that are singular nouns, to reflect that only one
value is required and allowed.

Boolean attributes like `monitoring` are usually written also as nouns
describing what is being enabled. However, they can sometimes be named as
verbs if the attribute is specifying whether to take some action, as with the
`delete_on_termination` flag within the `root_block_device` block.
Boolean attributes are ideally oriented so that `true` means to do something
and `false` means not to do it; it can be confusing do have "negative" flags
that prevent something from happening, since they require the user to follow
a double-negative in order to reason about what value should be provided.

Some attributes expect list, set or map values. In the above example,
`vpc_security_group_ids` is a set of strings, while `tags` is a map
from strings to strings. Such attributes should be named with *plural* nouns,
to reflect that multiple values may be provided.

List and set attributes use the same bracket syntax, and differ only in how
they are described to and used by the user. In lists, the ordering is
significant and duplicate values are often accepted. In sets, the ordering is
*not* significant and duplicated values are usually *not* accepted, since
presence or absense is what is important.

Map blocks use the same syntax as other configuration blocks, but the keys in
maps are arbitrary and not explicitly named by the plugin, so in some cases
(as in this `tags` example) they will not conform to the usual "lowercase with
underscores" naming convention.

Configuration blocks may contain other sub-blocks, such as `root_block_device`
in the above example. The patterns described above can also apply to such
sub-blocks. Sub-blocks are usually introduced by a singular noun, even if
multiple instances of the same-named block are accepted, since each distinct
instance represents a single object.
