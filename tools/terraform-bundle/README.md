# terraform-bundle

`terraform-bundle` was a solution intended to help with the problem
of distributing Terraform providers to environments where direct registry
access is impossible or undesirable, created in response to the Terraform v0.10
change to distribute providers separately from Terraform CLI.

The Terraform v0.13 series introduced our intended longer-term solutions
to this need:

* [Alternative provider installation methods](https://www.terraform.io/docs/cli/config/config-file.html#provider-installation),
  including the possibility of running server containing a local mirror of
  providers you intend to use which Terraform can then use instead of the
  origin registry.
* [The `terraform providers mirror` command](https://www.terraform.io/docs/cli/commands/providers/mirror.html),
  built in to Terraform v0.13.0 and later, can automatically construct a
  suitable directory structure to serve from a local mirror based on your
  current Terraform configuration, serving a similar (though not identical)
  purpose than `terraform-bundle` had served.

For those using Terraform CLI alone, without Terraform Cloud, we recommend
planning to transition to the above features instead of using
`terraform-bundle`.

## How to use `terraform-bundle`

However, if you need to continue using `terraform-bundle`
during a transitional period then you can use the version of the tool included
in the Terraform v0.15 branch to build bundles compatible with
Terraform v0.13.0 and later.

If you have a working toolchain for the Go programming language, you can
build a `terraform-bundle` executable as follows:

* `git clone --single-branch --branch=v0.15 --depth=1 https://github.com/hashicorp/terraform.git`
* `cd terraform`
* `go build -o ../terraform-bundle ./tools/terraform-bundle`

After running these commands, your original working directory will have an
executable named `terraform-bundle`, which you can then run.


For information
on how to use `terraform-bundle`, see
[the README from the v0.15 branch](https://github.com/hashicorp/terraform/blob/v0.15/tools/terraform-bundle/README.md).

You can follow a similar principle to build a `terraform-bundle` release
compatible with Terraform v0.12 by using `--branch=v0.12` instead of
`--branch=v0.15` in the command above. Terraform CLI versions prior to
v0.13 have different expectations for plugin packaging due to them predating
Terraform v0.13's introduction of automatic third-party provider installation.

## Terraform Enterprise Users

If you use Terraform Enterprise, the self-hosted distribution of
Terraform Cloud, you can use `terraform-bundle` as described above to build
custom Terraform packages with bundled provider plugins.

For more information, see
[Installing a Bundle in Terraform Enterprise](https://github.com/hashicorp/terraform/blob/v0.15/tools/terraform-bundle/README.md#installing-a-bundle-in-terraform-enterprise).
