---
layout: "docs"
page_title: "Provider Sources - Configuration Language"
---

## Provider Source

-> **Note:** The provider `source` attribute was introduced in Terraform v0.13.

The `required_providers` setting in can be used to declare the source for a
terraform provider. You do not need to declare the source of any provider owned
by HashiCorp(official provider / link?), but it is required for all other
providers, including locally-installed third-party providers.

To declare a provider's source, add a `required_providers` setting inside a `terrafrom` block:

```hcl
terraform {
  required_providers {
    mycloud = {
      version = "~> 1.0"
      source = "mycorp/mycloud"
    }
  }
}
```

The map keys in the required_providers block will be used as the provider
[localname](#localname) (for instance, `mycloud` in the example above). The best
practice is to use the type as the local name, however in configurations with
provider type collisions choose whatever localname makes the most sense for you.


For more information on the `required_providers` block, see
[Specifying Required Provider Versions and Source](https://www.terraform.io/docs/configuration/terraform.html#specifying-required-provider-versions-and-source).



## Specifying Required Provider Versions and Source

[inpage-source]: #specifying-required-provider-versions-and-source

-> **Note:** The provider `source` attribute was introduced in Terraform v0.13.

The `required_providers` setting is a map specifying a version constraint and source for
each provider required by your configuration.

```hcl
terraform {
  required_providers {
    aws = {
      version = ">= 2.7.0"
      source = "hashicorp/aws"
    }
  }
}
```

You may omit the `source` attribute for providers in the `hashicorp` namespace.
In those cases, an optional, simplified syntax may also be used:

```hcl
terraform {
  required_providers {
    aws = ">= 2.7.0"
  }
}
```

### Third-Party Providers

If you have a third-party provider that is not in the public registry,
you will need to make up an arbitrary source for that provider and copy (or
link) the binary to a directory corresponding to that source.

Once you've chosen a source, the binary needs to be installed into the following directory heirarchy:

```
$PLUGINDIR/$SOURCEHOST/$NAMESPACE/$TYPE/$VERSION/$OS_$ARCH/
```

The $OS_$ARCH must be the same operating system and architecture you are
currently using for Terraform.

For example, consider a provider called `terraform-provider-mycloud`. You can
use any source, though a best practice is to choose something logical to you:

```hcl
terraform {
  required_providers {
    mycloud = {
      source = "example.com/mycompany/mycloud"
      version = "1.0"
    }
  }
}
```

Terraform will look for the binary in the following directory (replace `$OS_$ARCH` with the appropriate operating system and architecture which you are using to run Terraform):

```
$PLUGINDIR/example.com/mycompany/mycloud/1.0/$OS_$ARCH/terraform-provider-mycloud
```

### Version Constraint Strings

Version constraint strings within the `required_providers` block use the
same version constraint syntax as for
[the `required_version` argument](#specifying-a-required-terraform-version)
described above.

When a configuration contains multiple version constraints for a single
provider -- for example, if you're using multiple modules and each one has
its own constraint -- _all_ of the constraints must hold to select a single
provider version for the whole configuration.

Re-usable modules should constrain only the minimum allowed version, such
as `>= 1.0.0`. This specifies the earliest version that the module is
compatible with while leaving the user of the module flexibility to upgrade
to newer versions of the provider without altering the module.

Root modules should use a `~>` constraint to set both a lower and upper bound
on versions for each provider they depend on, as described in
[Provider Versions](providers.html#provider-versions).

### Source Constraint Strings

A source constraint string within the `required_providers` is a string made up
of one to three parts, separated by a forward-slash (`/`). The parts are:

* `hostname`: The `hostname` is the registry host which indexes the provider.
  `hostname` may be omitted if the provider is in HashiCorp's public registry
  (`registry.terraform.io`).

* `namespace`: The registry namespace that the provider is in. This may be
  omitted if the provider is in HashiCorp's namesapce (`hashicorp`). `namespace`
  is required when `hostname` is set.

* `type`: The provider type.


The following are all valid source strings for the `random` provider in the
HashiCorp namespace:
```
"random"
"hashicorp/random"
"registry.terraform.io/hashicorp/random"
```

The following is _not_ a valid source string, since namespace is required when
hostname is provided:
```
"registry.terraform.io/random"
```