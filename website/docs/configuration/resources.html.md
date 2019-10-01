---
layout: "docs"
page_title: "Resources - Configuration Language"
sidebar_current: "docs-config-resources"
description: |-
  Resources are the most important element in a Terraform configuration.
  Each resource corresponds to an infrastructure object, such as a virtual
  network or compute instance.
---

# Resources

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Resources](../configuration-0-11/resources.html).

_Resources_ are the most important element in the Terraform language.
Each resource block describes one or more infrastructure objects, such
as virtual networks, compute instances, or higher-level components such
as DNS records.

## Resource Syntax

Resource declarations can include a number of advanced features, but only
a small subset are required for initial use. More advanced syntax features,
such as single resource declarations that produce multiple similar remote
objects, are described later in this page.

```hcl
resource "aws_instance" "web" {
  ami           = "ami-a1b2c3d4"
  instance_type = "t2.micro"
}
```

A `resource` block declares a resource of a given type ("aws_instance")
with a given local name ("web"). The name is used to refer to this resource
from elsewhere in the same Terraform module, but has no significance outside
of the scope of a module.

The resource type and name together serve as an identifier for a given
resource and so must be unique within a module.

Within the block body (between `{` and `}`) are the configuration arguments
for the resource itself. Most arguments in this section depend on the
resource type, and indeed in this example both `ami` and `instance_type` are
arguments defined specifically for [the `aws_instance` resource type](/docs/providers/aws/r/instance.html).

-> **Note:** Resource names must start with a letter or underscore, and may
contain only letters, digits, underscores, and dashes.

## Resource Types and Arguments

Each resource is associated with a single _resource type_, which determines
the kind of infrastructure object it manages and what arguments and other
attributes the resource supports.

Each resource type in turn belongs to a [provider](./providers.html),
which is a plugin for Terraform that offers a collection of resource types. A
provider usually provides resources to manage a single cloud or on-premises
infrastructure platform.

Most of the items within the body of a `resource` block are specific to the
selected resource type. These arguments can make full use of
[expressions](./expressions.html) and other dynamic Terraform
language features.

There are also some _meta-arguments_ that are defined by Terraform itself
and apply across all resource types. (See [Meta-Arguments](#meta-arguments) below.)

### Documentation for Resource Types

[Terraform's provider documentation][providers] is the primary place to
learn which resource types are available and which arguments to use for each
resource type. Once you understand Terraform's basic syntax, the provider
documentation will be where you spend the majority of your time on this website.

The "[Providers][]" link at the top level of the navigation sidebar will take
you to an alphabetical list of all of the providers distributed by HashiCorp.
You can find a specific provider in this master list, or choose a category from
the navigation sidebar to browse a more focused list of providers.

You can also search GitHub or other sources for third-party providers, which can
be installed as plugins to enable an even broader selection of resource types.

[providers]: /docs/providers/index.html

## Resource Behavior

A `resource` block describes your intent for a particular infrastructure object
to exist with the given settings. If you are writing a new configuration for
the first time, the resources it defines will exist _only_ in the configuration,
and will not yet represent real infrastructure objects in the target platform.

_Applying_ a Terraform configuration is the process of creating, updating,
and destroying real infrastructure objects in order to make their settings
match the configuration.

When Terraform creates a new infrastructure object represented by a `resource`
block, the identifier for that real object is saved in Terraform's
[state](/docs/state/index.html), allowing it to be updated and destroyed
in response to future changes. For resource blocks that already have an
associated infrastructure object in the state, Terraform compares the
actual configuration of the object with the arguments given in the
configuration and, if necessary, updates the object to match the configuration.

This general behavior applies for all resources, regardless of type. The
details of what it means to create, update, or destroy a resource are different
for each resource type, but this standard set of verbs is common across them
all.

The meta-arguments within `resource` blocks, documented in the
sections below, allow some details of this standard resource behavior to be
customized on a per-resource basis.

### Resource Dependencies

Most resources in a configuration don't have any particular relationship, and
Terraform can make changes to several unrelated resources in parallel.

However, some resources must be processed after other specific resources;
sometimes this is because of how the resource works, and sometimes the
resource's configuration just requires information generated by another
resource.

Most resource dependencies are handled automatically. Terraform analyses any
[expressions](./expressions.html) within a `resource` block to find references
to other objects, and treats those references as implicit ordering requirements
when creating, updating, or destroying resources. Since most resources with
behavioral dependencies on other resources also refer to those resources' data,
it's usually not necessary to manually specify dependencies between resources.

However, some dependencies cannot be recognized implicitly in configuration. For
example, if Terraform must manage access control policies _and_ take actions
that require those policies to be present, there is a hidden dependency between
the access policy and a resource whose creation depends on it. In these rare
cases, [the `depends_on` meta-argument][inpage-depend] can explicitly specify a
dependency.

## Meta-Arguments

Terraform CLI defines the following meta-arguments, which can be used with
any resource type to change the behavior of resources:

- [`depends_on`, for specifying hidden dependencies][inpage-depend]
- [`count`, for creating multiple resource instances according to a count][inpage-count]
- [`for_each`, to create multiple instances according to a map, or set of strings][inpage-for_each]
- [`provider`, for selecting a non-default provider configuration][inpage-provider]
- [`lifecycle`, for lifecycle customizations][inpage-lifecycle]
- [`provisioner` and `connection`, for taking extra actions after resource creation][inpage-provisioner]

These arguments often have additional restrictions on what language features can
be used with them, which are described in each

### `depends_on`: Explicit Resource Dependencies

[inpage-depend]: #depends_on-explicit-resource-dependencies

Use the `depends_on` meta-argument to handle hidden resource dependencies that
Terraform can't automatically infer.

Explicitly specifying a dependency is only necessary when a resource relies on
some other resource's behavior but _doesn't_ access any of that resource's data
in its arguments.

This argument is available in all `resource` blocks, regardless of resource
type. For example:

```hcl
resource "aws_iam_role" "example" {
  name = "example"

  # assume_role_policy is omitted for brevity in this example. See the
  # documentation for aws_iam_role for a complete example.
  assume_role_policy = "..."
}

resource "aws_iam_instance_profile" "example" {
  # Because this expression refers to the role, Terraform can infer
  # automatically that the role must be created first.
  role = aws_iam_role.example.name
}

resource "aws_iam_role_policy" "example" {
  name   = "example"
  role   = aws_iam_role.example.name
  policy = jsonencode({
    "Statement" = [{
      # This policy allows software running on the EC2 instance to
      # access the S3 API.
      "Action" = "s3:*",
      "Effect" = "Allow",
    }],
  })
}

resource "aws_instance" "example" {
  ami           = "ami-a1b2c3d4"
  instance_type = "t2.micro"

  # Terraform can infer from this that the instance profile must
  # be created before the EC2 instance.
  iam_instance_profile = aws_iam_instance_profile.example

  # However, if software running in this EC2 instance needs access
  # to the S3 API in order to boot properly, there is also a "hidden"
  # dependency on the aws_iam_role_policy that Terraform cannot
  # automatically infer, so it must be declared explicitly:
  depends_on = [
    aws_iam_role_policy.example,
  ]
}
```

The `depends_on` meta-argument, if present, must be a list of references
to other resources in the same module. Arbitrary expressions are not allowed
in the `depends_on` argument value, because its value must be known before
Terraform knows resource relationships and thus before it can safely
evaluate expressions.

The `depends_on` argument should be used only as a last resort. When using it,
always include a comment explaining why it is being used, to help future
maintainers understand the purpose of the additional dependency.

### `count`: Multiple Resource Instances By Count

[inpage-count]: #count-multiple-resource-instances-by-count

-> **Note:** A given resource block cannot use both `count` and `for_each`.

By default, a `resource` block configures one real infrastructure object.
However, sometimes you want to manage several similar objects, such as a fixed
pool of compute instances. Terraform has two ways to do this:
`count` and [`for_each`][inpage-for_each].

The `count` meta-argument accepts a whole number, and creates that many
instances of the resource. Each instance has a distinct infrastructure object
associated with it (as described above in
[Resource Behavior](#resource-behavior)), and each is separately created,
updated, or destroyed when the configuration is applied.

```hcl
resource "aws_instance" "server" {
  count = 4 # create four similar EC2 instances

  ami           = "ami-a1b2c3d4"
  instance_type = "t2.micro"

  tags {
    Name = "Server ${count.index}"
  }
}
```

#### The `count` Object

In resource blocks where `count` is set, an additional `count` object is
available in expressions, so you can modify the configuration of each instance.
This object has one attribute:

- `count.index` — The distinct index number (starting with `0`) corresponding
  to this instance.

#### Referring to Instances

When `count` is set, Terraform distinguishes between the resource block itself
and the multiple _resource instances_ associated with it. Instances are
identified by an index number, starting with `0`.

- `<TYPE>.<NAME>` (for example, `aws_instance.server`) refers to the resource block.
- `<TYPE>.<NAME>[<INDEX>]` (for example, `aws_instance.server[0]`,
  `aws_instance.server[1]`, etc.) refers to individual instances.

This is different from resources without `count` or `for_each`, which can be
referenced without an index or key.

-> **Note:** Within nested `provisioner` or `connection` blocks, the special
`self` object refers to the current _resource instance,_ not the resource block
as a whole.

#### Using Expressions in `count`

The `count` meta-argument accepts numeric [expressions](./expressions.html).
However, unlike most resource arguments, the `count` value must be known
_before_ Terraform performs any remote resource actions. This means `count`
can't refer to any resource attributes that aren't known until after a
configuration is applied (such as a unique ID generated by the remote API when
an object is created).

#### When to Use `for_each` Instead of `count`

If your resource instances are almost identical, `count` is appropriate. If some
of their arguments need distinct values that can't be directly derived from an
integer, it's safer to use `for_each`.

Before `for_each` was available, it was common to derive `count` from the
length of a list and use `count.index` to look up the original list value:

```hcl
variable "subnet_ids" {
  type = list(string)
}

resource "aws_instance" "server" {
  # Create one instance for each subnet
  count = length(var.subnet_ids)

  ami           = "ami-a1b2c3d4"
  instance_type = "t2.micro"
  subnet_id     = var.subnet_ids[count.index]

  tags {
    Name = "Server ${count.index}"
  }
}
```

This was fragile, because the resource instances were still identified by their
_index_ instead of the string values in the list. If an element was removed from
the middle of the list, every instance _after_ that element would see its
`subnet_id` value change, resulting in more remote object changes than intended.
Using `for_each` gives the same flexibility without the extra churn.

### `for_each`: Multiple Resource Instances Defined By a Map, or Set of Strings

[inpage-for_each]: #for_each-multiple-resource-instances-defined-by-a-map-or-set-of-strings

-> **Version note:** `for_each` was added in Terraform 0.12.6.

-> **Note:** A given resource block cannot use both `count` and `for_each`.

By default, a `resource` block configures one real infrastructure object.
However, sometimes you want to manage several similar objects, such as a fixed
pool of compute instances. Terraform has two ways to do this:
[`count`][inpage-count] and `for_each`.

The `for_each` meta-argument accepts a map or a set of strings, and creates an
instance for each item in that map or set. Each instance has a distinct
infrastructure object associated with it (as described above in
[Resource Behavior](#resource-behavior)), and each is separately created,
updated, or destroyed when the configuration is applied.

```hcl
resource "azurerm_resource_group" "rg" {
  for_each = {
    a_group = "eastus"
    another_group = "westus2"
  }
  name     = each.key
  location = each.value
}
```

#### The `each` Object

In resource blocks where `for_each` is set, an additional `each` object is
available in expressions, so you can modify the configuration of each instance.
This object has two attributes:

- `each.key` — The map key (or set member) corresponding to this instance.
- `each.value` — The map value corresponding to this instance. (If a set was
  provided, this is the same as `each.key`.)

#### Referring to Instances

When `for_each` is set, Terraform distinguishes between the resource block itself
and the multiple _resource instances_ associated with it. Instances are
identified by a map key (or set member) from the value provided to `for_each`.

- `<TYPE>.<NAME>` (for example, `azurerm_resource_group.rg`) refers to the resource block.
- `<TYPE>.<NAME>[<KEY>]` (for example, `azurerm_resource_group.rg["a_group"]`,
  `azurerm_resource_group.rg["another_group"]`, etc.) refers to individual instances.

This is different from resources without `count` or `for_each`, which can be
referenced without an index or key.

-> **Note:** Within nested `provisioner` or `connection` blocks, the special
`self` object refers to the current _resource instance,_ not the resource block
as a whole.

#### Using Sets

The Terraform language doesn't have a literal syntax for
[sets](./types.html#collection-types), but you can use the `toset` function to
convert a list of strings to a set:

```hcl
variable "subnet_ids" {
  type = list(string)
}

resource "aws_instance" "server" {
  for_each = toset(var.subnet_ids)

  ami           = "ami-a1b2c3d4"
  instance_type = "t2.micro"
  subnet_id     = each.key # note: each.key and each.value are the same for a set

  tags {
    Name = "Server ${each.key}"
  }
}
```

#### Using Expressions in `for_each`

The `for_each` meta-argument accepts map or set [expressions](./expressions.html).
However, unlike most resource arguments, the `for_each` value must be known
_before_ Terraform performs any remote resource actions. This means `for_each`
can't refer to any resource attributes that aren't known until after a
configuration is applied (such as a unique ID generated by the remote API when
an object is created).

### `provider`: Selecting a Non-default Provider Configuration

[inpage-provider]: #provider-selecting-a-non-default-provider-configuration

As described in [the Providers page](./providers.html),
Terraform optionally allows the definition of multiple alternative ("aliased")
configurations for a single provider, to allow management of resources
in different regions in multi-region services, etc.
The `provider` meta-argument overrides Terraform's default behavior of
selecting a provider configuration based on the resource type name.

By default, Terraform takes the initial word in the resource type name
(separated by underscores) and selects the default configuration for that
named provider. For example, the resource type `google_compute_instance`
is associated automatically with the default configuration for the provider
named `google`.

By using the `provider` meta-argument, an aliased provider configuration
can be selected:

```hcl
# default configuration
provider "google" {
  region = "us-central1"
}

# alternative, aliased configuration
provider "google" {
  alias  = "europe"
  region = "europe-west1"
}

resource "google_compute_instance" "example" {
  # This "provider" meta-argument selects the google provider
  # configuration whose alias is "europe", rather than the
  # default configuration.
  provider = google.europe

  # ...
}
```

A resource always has an implicit dependency on its associated provider, to
ensure that the provider is fully configured before any resource actions
are taken.

The `provider` meta-argument expects [a `<PROVIDER>.<ALIAS>` reference](./providers.html#referring-to-alternate-providers), which
does not need to be quoted. Arbitrary expressions are not permitted for
`provider` because it must be resolved while Terraform is constructing the
dependency graph, before it is safe to evaluate expressions.

### `lifecycle`: Lifecycle Customizations

[inpage-lifecycle]: #lifecycle-lifecycle-customizations

The general lifecycle for resources is described above in the
[Resource Behavior](#resource-behavior) section. Some details of that behavior
can be customized using the special nested `lifecycle` block within a resource
block body:

```
resource "azurerm_resource_group" "example" {
  # ...

  lifecycle {
    create_before_destroy = true
  }
}
```

The `lifecycle` block and its contents are meta-arguments, available
for all `resource` blocks regardless of type. The following lifecycle
meta-arguments are supported:

* `create_before_destroy` (bool) - By default, when Terraform must make a
  change to a resource argument that cannot be updated in-place due to
  remote API limitations, Terraform will instead destroy the existing object
  and then create a new replacement object with the new configured arguments.

    The `create_before_destroy` meta-argument changes this behavior so that
    the new replacement object is created _first,_ and then the prior object
    is destroyed only once the replacement is created.

    This is an opt-in behavior because many remote object types have unique
    name requirements or other constraints that must be accommodated for
    both a new and an old object to exist concurrently. Some resource types
    offer special options to append a random suffix onto each object name to
    avoid collisions, for example. Terraform CLI cannot automatically activate
    such features, so you must understand the constraints for each resource
    type before using `create_before_destroy` with it.

* `prevent_destroy` (bool) - This meta-argument, when set to `true`, will
  cause Terraform to reject with an error any plan that would destroy the
  infrastructure object associated with the resource, as long as the argument
  remains present in the configuration.

    This can be used as a measure of safety against the accidental replacement
    of objects that may be costly to reproduce, such as database instances.
    However, it will make certain configuration changes impossible to apply,
    and will prevent the use of the `terraform destroy` command once such
    objects are created, and so this option should be used sparingly.

    Since this argument must be present in configuration for the protection to
    apply, note that this setting does not prevent the remote object from
    being destroyed if the `resource` block were removed from configuration
    entirely: in that case, the `prevent_destroy` setting is removed along
    with it, and so Terraform will allow the destroy operation to succeed.

* `ignore_changes` (list of attribute names) - By default, Terraform detects
  any difference in the current settings of a real infrastructure object
  and plans to update the remote object to match configuration.

    In some rare cases, settings of a remote object are modified by processes
    outside of Terraform, which Terraform would then attempt to "fix" on the
    next run. In order to make Terraform share management responsibilities
    of a single object with a separate process, the `ignore_changes`
    meta-argument specifies resource attributes that Terraform should ignore
    when planning updates to the associated remote object.

    The arguments corresponding to the given attribute names are considered
    when planning a _create_ operation, but are ignored when planning an
    _update_.

    ```hcl
    resource "aws_instance" "example" {
      # ...

      lifecycle {
        ignore_changes = [
          # Ignore changes to tags, e.g. because a management agent
          # updates these based on some ruleset managed elsewhere.
          tags,
        ]
      }
    }
    ```

    You can also ignore specific map elements by writing references like
    `tags["Name"]` in the `ignore_changes` list, though with an important
    caveat: the ignoring applies only to in-place updates to an existing
    key. Adding or removing a key is treated by Terraform as a change to the
    containing map itself rather than to the individual key, and so if you
    wish to ignore changes to a particular tag made by an external system
    you must ensure that the Terraform configuration creates a placeholder
    element for that tag name so that the external system changes will be
    understood as an in-place edit of that key:

    ```hcl
    resource "aws_instance" "example" {
      # ...

      tags = {
        # Initial value for Name is overridden by our automatic scheduled
        # re-tagging process; changes to this are ignored by ignore_changes
        # below.
        Name = "placeholder"
      }

      lifecycle {
        ignore_changes = [
          tags["Name"],
        ]
      }
    }
    ```

    Instead of a list, the special keyword `all` may be used to instruct
    Terraform to ignore _all_ attributes, which means that Terraform can
    create and destroy the remote object but will never propose updates to it.

    Only attributes defined by the resource type can be ignored.
    `ignore_changes` cannot be applied to itself or to any other meta-arguments.

The `lifecycle` settings all effect how Terraform constructs and traverses
the dependency graph. As a result, only literal values can be used because
the processing happens too early for arbitrary expression evaluation.

### `provisioner` and `connection`: Resource Provisioners

[inpage-provisioner]: #provisioner-and-connection-resource-provisioners

Some infrastructure objects require some special actions to be taken after they
are created before they can become fully functional. For example, compute
instances may require configuration to be uploaded or a configuration management
program to be run before they can begin their intended operation.

Create-time actions like these can be described using _resource provisioners_.
A provisioner is another type of plugin supported by Terraform, and each
provisioner takes a different kind of action in the context of a resource
being created.

Provisioning steps should be used sparingly, since they represent
non-declarative actions taken during the creation of a resource and so
Terraform is not able to model changes to them as it can for the declarative
portions of the Terraform language.

Provisioners can also be defined to run when a resource is _destroyed_, with
certain limitations.

The `provisioner` and `connection` block types within `resource` blocks are
meta-arguments available across all resource types. Provisioners and their
usage are described in more detail in
[the Provisioners section](/docs/provisioners/index.html).

## Local-only Resources

While most resource types correspond to an infrastructure object type that
is managed via a remote network API, there are certain specialized resource
types that operate only within Terraform itself, calculating some results and
saving those results in the state for future use.

For example, local-only resource types exist for
[generating private keys](/docs/providers/tls/r/private_key.html),
[issuing self-signed TLS certificates](/docs/providers/tls/r/self_signed_cert.html),
and even [generating random ids](/docs/providers/random/r/id.html).
While these resource types often have a more marginal purpose than those
managing "real" infrastructure objects, they can be useful as glue to help
connect together other resources.

The behavior of local-only resources is the same as all other resources, but
their result data exists only within the Terraform state. "Destroying" such
a resource means only to remove it from the state, discarding its data.

## Operation Timeouts

Some resource types provide a special `timeouts` nested block argument that
allows you to customize how long certain operations are allowed to take
before being considered to have failed.
For example, [`aws_db_instance`](/docs/providers/aws/r/db_instance.html)
allows configurable timeouts for `create`, `update` and `delete` operations.

Timeouts are handled entirely by the resource type implementation in the
provider, but resource types offering these features follow the convention
of defining a child block called `timeouts` that has a nested argument
named after each operation that has a configurable timeout value.
Each of these arguments takes a string representation of a duration, such
as `"60m"` for 60 minutes, `"10s"` for ten seconds, or `"2h"` for two hours.

```hcl
resource "aws_db_instance" "example" {
  # ...

  timeouts {
    create = "60m"
    delete = "2h"
  }
}
```

The set of configurable operations is chosen by each resource type. Most
resource types do not support the `timeouts` block at all. Consult the
documentation for each resource type to see which operations it offers
for configuration, if any.

