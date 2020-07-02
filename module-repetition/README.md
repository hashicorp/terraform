# Terraform v0.13 Beta: Module `for_each` and `count`

The `for_each` and `count` features have, in previous Terraform versions,
allowed for systematically creating multiple resource instances from a single
resource block based on data from elsewhere in the module:

```hcl
variable "vpc_id" {
    type = string
}

variable "subnets" {
  type = map(object({
    cidr_block        = string
    availability_zone = string
  }))
}

resource "aws_subnet" "example" {
  for_each = var.subnets

  cidr_block        = each.value.cidr_block
  availability_zone = each.value.availability_zone
  tags = {
    Name = each.key
  }
}
```

Terraform v0.13 introduces a similar capability for entire modules, allowing
a single `module` block to produce multiple module _instances_ systematically:

```hcl
variable "project_id" {
  type = string
}

variable "regions" {
  type = map(object({
    region            = string
    network           = string
    subnetwork        = string
    ip_range_pods     = string
    ip_range_services = string
  }))
}

module "kubernetes_cluster" {
  source   = "terraform-google-modules/kubernetes-engine/google"
  for_each = var.regions

  project_id        = var.project_id
  name              = each.key
  region            = each.value.region
  network           = each.value.network
  subnetwork        = each.value.subnetwork
  ip_range_pods     = each.value.ip_range_pods
  ip_range_services = each.value.ip_range_services
}
```

This directory contains an example Terraform configuration demonstrating the
module `for_each` mechanism using some contrived, local-only resource types.

Try adapting this example to be a hypothetical solution to some real-world
problems you've seen in your systems, perhaps using resource types that
correspond to more common infrastructure objects in your chosen cloud
platform(s). We'd love to hear about your experiences, including any bugs or
rough edges you encounter.

## Usage Notes

These features were a long time coming because it required some quite
significant changes to how Terraform represents modules internally.
Consequently, the behavior of these new features inevitably interacts with the
behaviors of other language features. We felt it was important to release an
initial version of these features soon to meet as many needs as possible, but
we do know that there are some interactions with existing Terraform features
that are not yet ideal, and we'll work to improve these other features in
later releases.

### Associating provider configurations with modules

From Terraform 0.11 onwards the Terraform team has recommended placing provider
configurations only in the _root_ module of a configuration, and then having
child modules either implicitly inherit or explicitly receive provider
configurations from the root:

```hcl
provider "azurerm" {
  # ...

  alias = "special"
}

module "uses_azurerm" {
  # ...

  providers = {
    azurerm = azurerm.special
  }
}
```

We retained the ability to include `provider` blocks in child modules to give
some time to migrate away from that approach, but the introduction of `for_each`
and `count` for modules forced us to finally constrain that legacy usage
pattern. Terraform v0.13 will not allow `provider` blocks inside any module
that has been instantiated with either `for_each` or `count` arguments.

For now, nested provider blocks are still permitted in modules that _do not_
use `for_each` or `count`, because we know we still need to make some
improvements to the way resources and provider configurations are associated in
the Terraform language and intend to be pragmatic about nested module `provider`
blocks in the meantime.

In particular though, we'd like to note that for providers where the provider
configuration is the primary or only way to select a target region or similar
for declared objects, such as in the AWS provider, it will not yet be possible
to systematically assign different regions to different instances of a module
created using `for_each` or `count`.
