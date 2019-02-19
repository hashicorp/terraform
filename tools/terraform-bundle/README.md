# terraform-bundle

`terraform-bundle` is a helper program to create "bundle archives", which are
zip files that contain both a particular version of Terraform and a number
of provider plugins.

Normally `terraform init` will download and install the plugins necessary to
work with a particular configuration, but sometimes Terraform is deployed in
a network that, for one reason or another, cannot access the official
plugin repository for automatic download.

In some cases, this can be solved by installing provider plugins into the
[user plugins directory](https://www.terraform.io/docs/configuration/providers.html#third-party-plugins).
However, this doesn't always meet the needs of automated deployments.

`terraform-bundle` provides an alternative, by allowing the auto-download
process to be run out-of-band on a separate machine that _does_ have access
to the repository. The result is a zip file that can be extracted onto the
target system to install both the desired Terraform version and a selection
of providers, thus avoiding the need for on-the-fly plugin installation.

## Building

To build `terraform-bundle` from source, set up a Terraform development
environment per [Terraform's own README](../../README.md) and then install
this tool from within it:

```
$ go install ./tools/terraform-bundle
```

This will install `terraform-bundle` in `$GOPATH/bin`, which is assumed by
the rest of this README to be in `PATH`.

`terraform-bundle` is a repackaging of the module installation functionality
from Terraform itself, so for best results you should build from the tag
relating to the version of Terraform you plan to use. There is some slack in
this requirement due to the fact that the module installation behavior changes
rarely, but please note that in particular bundles for versions of
Terraform before v0.12 must be built from a `terraform-bundle` built against
a Terraform v0.11 tag at the latest, since Terraform v0.12 installs plugins
in a different way that is not compatible.

## Usage

`terraform-bundle` uses a simple configuration file to define what should
be included in a bundle. This is designed so that it can be checked into
version control and used by an automated build and deploy process.

The configuration file format works as follows:

```hcl
terraform {
  # Version of Terraform to include in the bundle. An exact version number
  # is required.
  version = "0.10.0"
}

# Define which provider plugins are to be included
providers {
  # Include the newest "aws" provider version in the 1.0 series.
  aws = ["~> 1.0"]

  # Include both the newest 1.0 and 2.0 versions of the "google" provider.
  # Each item in these lists allows a distinct version to be added. If the
  # two expressions match different versions then _both_ are included in
  # the bundle archive.
  google = ["~> 1.0", "~> 2.0"]

  # Include a custom plugin to the bundle. Will search for the plugin in the
  # plugins directory, and package it with the bundle archive. Plugin must have
  # a name of the form: terraform-provider-*, and must be build with the operating
  # system and architecture that terraform enterprise is running, e.g. linux and amd64
  customplugin = ["0.1"]
}

```

The `terraform` block defines which version of Terraform will be included
in the bundle. An exact version is required here.

The `providers` block defines zero or more providers to include in the bundle
along with core Terraform. Each attribute in this block is a provider name,
and its value is a list of version constraints. For each given constraint,
`terraform-bundle` will find the newest available version matching the
constraint and include it in the bundle.

It is allowed to specify multiple constraints for the same provider, in which
case multiple versions can be included in the resulting bundle. Each constraint
string given results in a separate plugin in the bundle, unless two constraints
resolve to the same concrete plugin.

Including multiple versions of the same provider allows several configurations
running on the same system to share an installation of the bundle and to
choose a version using version constraints within the main Terraform
configuration. This avoids the need to upgrade all configurations to newer
versions in lockstep.

After creating the configuration file, e.g. `terraform-bundle.hcl`, a bundle
zip file can be produced as follows:

```
$ terraform-bundle package terraform-bundle.hcl
```

By default the bundle package will target the operating system and CPU
architecture where the tool is being run. To override this, use the `-os` and
`-arch` options. For example, to build a bundle for on-premises Terraform
Enterprise:

```
$ terraform-bundle package -os=linux -arch=amd64 terraform-bundle.hcl
```

The bundle file is assigned a name that includes the core Terraform version
number, a timestamp to the nearest hour of when the bundle was built, and the
target OS and CPU architecture. It is recommended to refer to a bundle using
this composite version number so that bundle archives can be easily
distinguished from official release archives and from each other when multiple
bundles contain the same core Terraform version.

To include custom plugins in the bundle file, create a local directory "./plugins"
and put all the plugins you want to include there. Optionally, you can use the
`-plugin-dir` flag to specify a location where to find the plugins. To be recognized
as a valid plugin, the file must have a name of the form
`terraform-provider-<NAME>_v<VERSION>`. In
addition, ensure that the plugin is built using the same operating system and
architecture used for Terraform Enterprise. Typically this will be `linux` and `amd64`.

## Provider Resolution Behavior

Terraform's provider resolution behavior is such that if a given constraint
can be resolved by any plugin already installed on the system it will use
the newest matching plugin and not attempt automatic installation.

Therefore if automatic installation is not desired, it is important to ensure
that version constraints within Terraform configurations do not exclude all
of the versions available from the bundle. If a suitable version cannot be
found in the bundle, Terraform _will_ attempt to satisfy that dependency by
automatic installation from the official repository. If you want
`terraform init` to explicitly fail instead of contacting the repository, pass
the `-get-plugins=false` option.

For full details about provider resolution, see
[How Terraform Works: Plugin Discovery](https://www.terraform.io/docs/extend/how-terraform-works.html#discovery).

The downloaded provider archives are verified using the same signature check
that is used for auto-installed plugins, using Hashicorp's release key. At
this time, the core Terraform archive itself is _not_ verified in this way;
that may change in a future version of this tool.

## Installing a Bundle in On-premises Terraform Enterprise

If using a private install of Terraform Enterprise in an "air-gapped"
environment, this tool can produce a custom Terraform version package, which
includes a set of provider plugins along with core Terraform.

To create a suitable bundle, use the `-os` and `-arch` options as described
above to produce a bundle targeting `linux_amd64`. You can then place this
archive on an HTTP server reachable by the Terraform Enterprise hosts and
install it as per
[Administration: Managing Terraform Versions](https://www.terraform.io/docs/enterprise/private/admin/resources.html#managing-terraform-versions).

After clicking the "Add Terraform Version" button:

1. In the "Version" field, enter the generated bundle version from the bundle
   filename, which will be of the form `N.N.N-bundleYYYYMMDDHH`.
2. In the "URL" field, enter the URL where the generated bundle archive can be found.
3. In the "SHA256 Checksum" field, enter the SHA256 hash of the file, which can
   be found by running `sha256sum <FILE>` or `shasum -a256 <FILE>`.

The new bundle version can then be selected as the Terraform version for
any workspace. When selected, configurations that require only plugins
included in the bundle will run without trying to auto-install.

Note that the above does _not_ apply to Terraform Pro, or to Terraform Premium
when not running a private install. In these packages, Terraform versions
are managed centrally across _all_ organizations and so custom bundles are not
supported.

For more information on the available Terraform Enterprise packages, see
[the Terraform product site](https://www.hashicorp.com/products/terraform/).
