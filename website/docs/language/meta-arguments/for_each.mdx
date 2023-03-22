---
page_title: The for_each Meta-Argument - Configuration Language
description: >-
  The for_each meta-argument allows you to manage similar infrastructure
  resources without writing a separate block for each one.
---

# The `for_each` Meta-Argument

By default, a [resource block](/terraform/language/resources/syntax) configures one real
infrastructure object (and similarly, a
[module block](/terraform/language/modules/syntax) includes a
child module's contents into the configuration one time).
However, sometimes you want to manage several similar objects (like a fixed
pool of compute instances) without writing a separate block for each one.
Terraform has two ways to do this:
[`count`](/terraform/language/meta-arguments/count) and `for_each`.

> **Hands-on:** Try the [Manage Similar Resources With For Each](/terraform/tutorials/configuration-language/for-each) tutorial.

If a resource or module block includes a `for_each` argument whose value is a map or
a set of strings, Terraform creates one instance for each member of
that map or set.

-> **Version note:** `for_each` was added in Terraform 0.12.6. Module support
for `for_each` was added in Terraform 0.13; previous versions can only use
it with resources.

-> **Note:** A given resource or module block cannot use both `count` and `for_each`.

## Basic Syntax

`for_each` is a meta-argument defined by the Terraform language. It can be used
with modules and with every resource type.

The `for_each` meta-argument accepts a map or a set of strings, and creates an
instance for each item in that map or set. Each instance has a distinct
infrastructure object associated with it, and each is separately created,
updated, or destroyed when the configuration is applied.

Map:

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

Set of strings:

```hcl
resource "aws_iam_user" "the-accounts" {
  for_each = toset( ["Todd", "James", "Alice", "Dottie"] )
  name     = each.key
}
```

Child module:

```hcl
# my_buckets.tf
module "bucket" {
  for_each = toset(["assets", "media"])
  source   = "./publish_bucket"
  name     = "${each.key}_bucket"
}
```

```hcl
# publish_bucket/bucket-and-cloudfront.tf
variable "name" {} # this is the input parameter of the module

resource "aws_s3_bucket" "example" {
  # Because var.name includes each.key in the calling
  # module block, its value will be different for
  # each instance of this module.
  bucket = var.name

  # ...
}

resource "aws_iam_user" "deploy_user" {
  # ...
}
```

## The `each` Object

In blocks where `for_each` is set, an additional `each` object is
available in expressions, so you can modify the configuration of each instance.
This object has two attributes:

- `each.key` — The map key (or set member) corresponding to this instance.
- `each.value` — The map value corresponding to this instance. (If a set was
  provided, this is the same as `each.key`.)

## Limitations on values used in `for_each`

The keys of the map (or all the values in the case of a set of strings) must
be _known values_, or you will get an error message that `for_each` has dependencies
that cannot be determined before apply, and a `-target` may be needed.

`for_each` keys cannot be the result (or rely on the result of) of impure functions,
including `uuid`, `bcrypt`, or `timestamp`, as their evaluation is deferred during the
main evaluation step.

Sensitive values, such as [sensitive input variables](/terraform/language/values/variables#suppressing-values-in-cli-output),
[sensitive outputs](/terraform/language/values/outputs#sensitive-suppressing-values-in-cli-output),
or [sensitive resource attributes](/terraform/language/expressions/references#sensitive-resource-attributes),
cannot be used as arguments to `for_each`. The value used in `for_each` is used
to identify the resource instance and will always be disclosed in UI output,
which is why sensitive values are not allowed.
Attempts to use sensitive values as `for_each` arguments will result in an error.

If you transform a value containing sensitive data into an argument to be used in `for_each`, be aware that
[most functions in Terraform will return a sensitive result if given an argument with any sensitive content](/terraform/language/expressions/function-calls#using-sensitive-data-as-function-arguments).
In many cases, you can achieve similar results to a function used for this purpose by
using a `for` expression. For example, if you would like to call `keys(local.map)`, where
`local.map` is an object with sensitive values (but non-sensitive keys), you can create a
value to pass to  `for_each` with `toset([for k,v in local.map : k])`.

## Using Expressions in `for_each`

The `for_each` meta-argument accepts map or set [expressions](/terraform/language/expressions).
However, unlike most arguments, the `for_each` value must be known
_before_ Terraform performs any remote resource actions. This means `for_each`
can't refer to any resource attributes that aren't known until after a
configuration is applied (such as a unique ID generated by the remote API when
an object is created).

The `for_each` value must be a map or set with one element per desired resource
instance. To use a sequence as the `for_each` value, you must use an expression
that explicitly returns a set value, like the [toset](/terraform/language/functions/toset)
function. To prevent unwanted surprises during conversion, the `for_each` argument
does not implicitly convert lists or tuples to sets.
If you need to declare resource instances based on a nested
data structure or combinations of elements from multiple data structures you
can use Terraform expressions and functions to derive a suitable value.
For example:

- Transform a multi-level nested structure into a flat list by
  [using nested `for` expressions with the `flatten` function](/terraform/language/functions/flatten#flattening-nested-structures-for-for_each).
- Produce an exhaustive list of combinations of elements from two or more
  collections by
  [using the `setproduct` function inside a `for` expression](/terraform/language/functions/setproduct#finding-combinations-for-for_each).

### Chaining `for_each` Between Resources

Because a resource using `for_each` appears as a map of objects when used in
expressions elsewhere, you can directly use one resource as the `for_each`
of another in situations where there is a one-to-one relationship between
two sets of objects.

For example, in AWS an `aws_vpc` object is commonly associated with a number
of other objects that provide additional services to that VPC, such as an
"internet gateway". If you are declaring multiple VPC instances using `for_each`
then you can chain that `for_each` into another resource to declare an
internet gateway for each VPC:

```hcl
variable "vpcs" {
  type = map(object({
    cidr_block = string
  }))
}

resource "aws_vpc" "example" {
  # One VPC for each element of var.vpcs
  for_each = var.vpcs

  # each.value here is a value from var.vpcs
  cidr_block = each.value.cidr_block
}

resource "aws_internet_gateway" "example" {
  # One Internet Gateway per VPC
  for_each = aws_vpc.example

  # each.value here is a full aws_vpc object
  vpc_id = each.value.id
}

output "vpc_ids" {
  value = {
    for k, v in aws_vpc.example : k => v.id
  }

  # The VPCs aren't fully functional until their
  # internet gateways are running.
  depends_on = [aws_internet_gateway.example]
}
```

This chaining pattern explicitly and concisely declares the relationship
between the internet gateway instances and the VPC instances, which tells
Terraform to expect the instance keys for both to always change together,
and typically also makes the configuration easier to understand for human
maintainers.

## Referring to Instances

When `for_each` is set, Terraform distinguishes between the block itself
and the multiple _resource or module instances_ associated with it. Instances are
identified by a map key (or set member) from the value provided to `for_each`.

- `<TYPE>.<NAME>` or `module.<NAME>` (for example, `azurerm_resource_group.rg`) refers to the block.
- `<TYPE>.<NAME>[<KEY>]` or `module.<NAME>[<KEY>]` (for example, `azurerm_resource_group.rg["a_group"]`,
  `azurerm_resource_group.rg["another_group"]`, etc.) refers to individual instances.

This is different from resources and modules without `count` or `for_each`, which can be
referenced without an index or key.

Similarly, resources from child modules with multiple instances are prefixed
with `module.<NAME>[<KEY>]` when displayed in plan output and elsewhere in the UI.
For a module without `count` or `for_each`, the address will not contain
the module index as the module's name suffices to reference the module.

-> **Note:** Within nested `provisioner` or `connection` blocks, the special
`self` object refers to the current _resource instance,_ not the resource block
as a whole.

## Using Sets

The Terraform language doesn't have a literal syntax for
[set values](/terraform/language/expressions/type-constraints#collection-types), but you can use the `toset`
function to explicitly convert a list of strings to a set:

```hcl
locals {
  subnet_ids = toset([
    "subnet-abcdef",
    "subnet-012345",
  ])
}

resource "aws_instance" "server" {
  for_each = local.subnet_ids

  ami           = "ami-a1b2c3d4"
  instance_type = "t2.micro"
  subnet_id     = each.key # note: each.key and each.value are the same for a set

  tags = {
    Name = "Server ${each.key}"
  }
}
```

Conversion from list to set discards the ordering of the items in the list and
removes any duplicate elements. `toset(["b", "a", "b"])` will produce a set
containing only `"a"` and `"b"` in no particular order; the second `"b"` is
discarded.

If you are writing a module with an [input variable](/terraform/language/values/variables) that
will be used as a set of strings for `for_each`, you can set its type to
`set(string)` to avoid the need for an explicit type conversion:

```hcl
variable "subnet_ids" {
  type = set(string)
}

resource "aws_instance" "server" {
  for_each = var.subnet_ids

  # (and the other arguments as above)
}
```
