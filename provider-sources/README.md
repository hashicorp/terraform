# Terraform v0.13 Beta: Automatic installation of third-party providers

In Terraform v0.12 and earlier, `terraform init` was able to automatically
download and install providers packaged and distributed by HashiCorp, but to
use any other provider required manual installation.

Terraform v0.13 introduces a new hierarchical provider naming scheme, allowing
the providers developed by HashiCorp to occupy separate namespaces than
providers developed or distributed by others in the community, and thus to
allow third-party providers to be indexed in
[Terraform Registry](https://registry.terraform.io/) and to be automatically installed
by Terraform.

> **Note:** The features for publishing providers in Terraform Registry are
> in closed beta at the time of writing, separately from the Terraform v0.13 beta.
> There are already some third-party providers available for installation, but
> due to the Terraform Registry beta they are not yet discoverable on the
> Terraform Registry website.

The new provider naming scheme includes a registry hostname and a namespace in
addition to the provider name. The existing `azurerm` provider is, for example,
now known as `hashicorp/azurerm`, which is in turn a shorthand for
`registry.terraform.io/hashicorp/azurerm`. Providers not developed by HashiCorp
can be selected from their own namespaces, using a new provider requirements
syntax added in Terraform v0.13:

```hcl
terraform {
  required_providers {
    jetstream = {
      source  = "nats-io/jetstream"
      version = "0.0.5"
    }
  }
}
```

The address `nats-io/jetstream` is a shorthand for
`registry.terraform.io/nats-io/jetstream`, indicating that this is a third-party
provider published in the public Terraform Registry for general use.

The provider registry protocol will eventually be published so that others can
implement it, in which case other hostnames will become usable in source
addresses. At the time of writing this guide, only the public Terraform Registry
at `registry.terraform.io` is available for general testing.

This directory contains a simple example of a Terraform module that declares
some provider dependencies, but it's a contrived example focused on showing the
new syntax. To test provider installation during the v0.13 beta we suggest
instead taking some configurations you already have in your environment and
trying to run `terraform init` against them under the beta to see what happens.

With that said, please **do not run `terraform apply` or other similar operations against your real configurations**:
once Terraform has applied changes to an existing state it will be hard to
revert to using Terraform v0.12 again. If you make any local changes to your
configuration while testing, use your version control system to undo those
changes and avoid committing them to your main version control branches.

If you'd like to do deeper testing of _using_ providers installed under the
new scheme, please do that only with separate configurations you've written
for testing only. The contrived example in this directory may help as a
starting point.

## Usage Notes

### Implied source addresses for the "hashicorp" namespace

As a measure of backward-compatibility for commonly-used existing providers,
Terraform 0.13 includes a special case that if no explicit `source` is selected
for a provider Terraform will construct one by selecting `registry.terraform.io`
as the origin registry and `hashicorp` as the namespace.

For example, if you write a `provider "aws"` in your configuration without
also having a `required_providers` entry for `aws`, Terraform will currently
assume that you meant `hashicorp/aws`, which is short for
`registry.terraform.io/hashicorp/aws`.

If you try testing with existing examples that use providers that now belong
to other namespaces, you will see one of the following errors:

```
Error: Failed to install providers

Could not find required providers, but found possible alternatives:

  hashicorp/datadog -> terraform-providers/datadog
  hashicorp/fastly -> terraform-providers/fastly

If these suggestions look correct, upgrade your configuration with the
following command:
    terraform 0.13upgrade
```

```
Error: Failed to install provider

Error while installing hashicorp/happycloud: provider registry
registry.terraform.io does not have a provider named
registry.terraform.io/hashicorp/happycloud
```

As noted in the first error message, providers that were previously distributed
by HashiCorp but maintained by third-parties can potentially have requirement
declarations generated automatically by running the upgrade tool
`terraform 0.13upgrade`. This will modify the configuration of your current
module, allowing you to review the proposed changes using your version control
system. During the beta we expect you would not want to commit the result to
your primary branch, but you can use the updated configuration for local-only
testing on temporary test infrastructure.

The second error message will appear if you are using providers that previously
required manual installation under Terraform 0.12. The upgrade tool cannot
automatically infer a source address for those, so you'll need to adapt your
local filesystem directory of providers to use a new multi-level directory
structure so Terraform 0.13 can find it. There's more information on that in
the following section.

### Provider plugins in the local filesystem

While `terraform init` has previously been able to support automatic
installation of HashiCorp-distributed providers, third-party-packaged providers
had to be installed manually in the local filesystem. Some users also chose
to create local copies of the HashiCorp-distributed providers in order to
avoid repeatedly re-downloading them.

Terraform v0.13 still supports local copies of providers -- now officially
called "local mirrors" -- but the new multi-level addressing scheme for providers
means that the expected directory structure in these local directories has
now changed to include each provider's origin registry hostname and namespace,
giving a directory structure like this:

```
registry.terraform.io/hashicorp/azurerm/2.0.0/linux_amd64/terraform-provider-azurerm_v2.0.0
```

In the above, `terraform-provider-azurerm_v2.0.0` is whatever executable is
inside the provider's distribution zip file, and the containing directory
structure allows Terraform to see that this is a plugin intended to serve
the provider `hashicorp/azurerm`, which is short for
`registry.terraform.io/hashicorp/azurerm`, at version 2.0.0 on the platform
linux_amd64.

If you use local copies of providers that `terraform init` would normally be
able to auto-install, you can use the new `terraform providers mirror` command
to automatically construct the above directory structure for the providers
required in the current configuration:

```
terraform providers mirror ~/.terraform.d/plugins
```

The above will create local mirrors in one of the directories Terraform consults
by default on non-Windows systems. This same directory structure is used for
all of the directories Terraform searches for plugins.

Note that due to the directory structure now being multi-level, Terraform no
longer looks for provider plugins in the same directory where the `terraform`
executable is installed, because it's not conventional for there to be
subdirectories under directories like `/usr/bin` on a Unix system.
