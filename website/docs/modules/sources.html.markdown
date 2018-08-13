---
layout: "docs"
page_title: "Module Sources"
sidebar_current: "docs-modules-sources"
description: The source argument within a module block specifies the location of the source code of a child module.
---

# Module Sources

As introduced in [the _Usage_ section](/docs/modules/usage.html), the `source`
argument in a `module` block tells Terraform where to find the source code
for the desired child module.

Terraform uses this during the module installation step of `terraform init`
to download the source code to a directory on local disk so that it can be
used by other Terraform commands.

The module installer supports installation from a number of different source
types, as listed below.

  * [Local paths](#local-paths)

  * [Terraform Registry](#terraform-registry)

  * [GitHub](#github)

  * [Bitbucket](#bitbucket)

  * Generic [Git](#generic-git-repository), [Mercurial](#generic-mercurial-repository) repositories

  * [HTTP URLs](#http-urls)

  * [S3 buckets](#s3-bucket)

Each of these is described in the following sections. Module source addresses
use a _URL-like_ syntax, but with extensions to support unambiguous selection
of sources and additional features.

We recommend using local file paths for closely-related modules used primarily
for the purpose of factoring out repeated code elements, and using a native
Terraform module registry for modules intended to be shared by multiple calling
configurations. We support other sources so that you can potentially distribute
Terraform modules internally with existing infrastructure.

Many of the source types will make use of "ambient" credentials available
when Terraform is run, such as from environment variables or credentials files
in your home directory. This is covered in more detail in each of the following
sections.

We recommend placing each module that is intended to be re-usable in the root
of its own repository or archive file, but it is also possible to
[reference modules from subdirectories](#modules-in-package-sub-directories).

## Local Paths

Local path references allow for factoring out portions of a configuration
within a single source repository.

```hcl
module "consul" {
  source = "./consul"
}
```

A local path must begin with either `./` or `../` to indicate that a local
path is intended, to distinguish from
[a module registry address](#terraform-registry).

Local paths are special in that they are not "installed" in the same sense
that other sources are: the files are already present on local disk (possibly
as a result of installing a parent module) and so can just be used directly.
Their source code is automatically updated if the parent module is upgraded.

## Terraform Registry

A module registry is the native way of distributing Terraform modules for use
across multiple configurations, using a Terraform-specific protocol that
has full support for module versioning.

[Terraform Registry](https://registry.terraform.io/) is an index of modules
shared publicly using this protocol. This public registry is the easiest way
to get started with Terraform and find modules created by others in the
community.

You can also use a
[private registry](/docs/registry/private.html), either
via the built-in feature from Terraform Enterprise, or by running a custom
service that implements
[the module registry protocol](/docs/registry/api.html).

Modules on the public Terraform Registry can be referenced using a registry
source address of the form `<NAMESPACE>/<NAME>/<PROVIDER>`, with each
module's information page on the registry site including the exact address
to use.

```hcl
module "consul" {
  source = "hashicorp/consul/aws"
  version = "0.1.0"
}
```

The above example will use the
[Consul module for AWS](https://registry.terraform.io/modules/hashicorp/consul/aws)
from the public registry.

For modules hosted in other registries, prefix the source address with an
additional `<HOSTNAME>/` portion, giving the hostname of the private registry:

```hcl
module "consul" {
  source = "app.terraform.io/example-corp/k8s-cluster/azurerm"
  version = "1.1.0"
}
```

If you are using the managed version of Terraform Enterprise, its private
registry hostname is `app.terraform.io`. If you are using Terraform Enterprise
on-premises, its private registry hostname is the same hostname you use to
access your Terraform Enterprise instance.

Registry modules support versioning. You can provide a specific version as shown
in the above examples, or use flexible
[version constraints](/docs/modules/usage.html#module-versions).

You can learn more about the registry at the
[Terraform Registry documentation](/docs/registry/modules/use.html#using-modules).

To access modules from a private registry, you may need to configure an access
token [in the CLI config](/docs/commands/cli-config.html#credentials). Use the
same hostname as used in the module source string. For a private registry
within Terraform Enterprise, use the same authentication token as you would
use with the Enterprise API or command-line clients.

## GitHub

Terraform will recognize unprefixed `github.com` URLs and interpret them
automatically as Git repository sources.

```hcl
module "consul" {
  source = "github.com/hashicorp/example"
}
```

The above address scheme will clone over HTTPS. To clone over SSH, use the
following form:

```hcl
module "consul" {
  source = "git@github.com:hashicorp/example.git"
}
```

These GitHub schemes are treated as convenient aliases for
[the general Git repository address scheme](#generic-git-repository), and so
they obtain credentials in the same way and support the `ref` argument for
selecting a specific revision. You will need to configure credentials in
particular to access private repositories.

## Bitbucket

Terraform will recognize unprefixed `bitbucket.org` URLs and interpret them
automatically as BitBucket repositories:

```hcl
module "consul" {
  source = "bitbucket.org/hashicorp/terraform-consul-aws"
}
```

This shorthand works only for public repositories, because Terraform must
access the BitBucket API to learn if the given repository uses Git or Mercurial.

Terraform treats the result either as [a Git source](#generic-git-repository)
or [a Mercurial source](#generic-mercurial-repository) depending on the
repository type. See the sections on each version control type for information
on how to configure credentials for private repositories and how to specify
a specific revision to install.

## Generic Git Repository

Arbitrary Git repositories can be used by prefixing the address with the
special `git::` prefix. After this prefix, any valid
[Git URL](https://git-scm.com/docs/git-clone#_git_urls_a_id_urls_a)
can be specified to select one of the protocols supported by Git.

For example, to use HTTPS or SSH:

```hcl
module "vpc" {
  source = "git::https://example.com/vpc.git"
}

module "storage" {
  source = "git::ssh://username@example.com/storage.git"
}
```

Terraform installs modules from Git repositories by running `git clone`, and
so it will respect any local Git configuration set on your system, including
credentials. To access a non-public Git repository, configure Git with
suitable credentials for that repository.

If you use the SSH protocol then any configured SSH keys will be used
automatically. This is the most common way to access non-public Git
repositories from automated systems beacuse it is easy to configure
and allows access to private repositories without interactive prompts.

If using the HTTP/HTTPS protocol, or any other protocol that uses
username/password credentials, configure
[Git Credentials Storage](https://git-scm.com/book/en/v2/Git-Tools-Credential-Storage)
to select a suitable source of credentials for your environment.

If your Terraform configuration will be used within [Terraform Enterprise](https://www.hashicorp.com/products/terraform),
only SSH key authentication is supported, and
[keys can be configured on a per-workspace basis](/docs/enterprise/workspaces/ssh-keys.html).

### Selecting a Revision

By default, Terraform will clone and use the default branch (referenced by
`HEAD`) in the selected repository. You can override this using the
`ref` argument:

```hcl
module "vpc" {
  source = "git::https://example.com/vpc.git?ref=v1.2.0"
}
```

The value of the `ref` argument can be any reference that would be accepted
by the `git checkout` command, including branch and tag names.

## Generic Mercurial Repository

You can use arbitrary Mercurial repositories by prefixing the address with the
special `hg::` prefix. After this prefix, any valid
[Mercurial URL](https://www.mercurial-scm.org/repo/hg/help/urls)
can be specified to select one of the protocols supported by Mercurial.

```hcl
module "vpc" {
  source = "hg::http://example.com/vpc.hg"
}
```

Terraform installs modules from Mercurial repositories by running `hg clone`, and
so it will respect any local Mercurial configuration set on your system,
including credentials. To access a non-public repository, configure Mercurial
with suitable credentials for that repository.

If you use the SSH protocol then any configured SSH keys will be used
automatically. This is the most common way to access non-public Mercurial
repositories from automated systems beacuse it is easy to configure
and allows access to private repositories without interactive prompts.

If your Terraform configuration will be used within [Terraform Enterprise](https://www.hashicorp.com/products/terraform),
only SSH key authentication is supported, and
[keys can be configured on a per-workspace basis](/docs/enterprise/workspaces/ssh-keys.html).

### Selecting a Revision

You can select a non-default branch or tag using the optional `ref` argument:

```hcl
module "vpc" {
  source = "hg::http://example.com/vpc.hg?ref=v1.2.0"
}
```

## HTTP URLs

When you use an HTTP or HTTPS URL, Terraform will make a `GET` request to
the given URL, which can return _another_ source address. This indirection
allows using HTTP URLs as a sort of "vanity redirect" over a more complicated
module source address.

Terraform will append an additional query string argument `terraform-get=1` to
the given URL before sending the `GET` request, allowing the server to
optionally return a different result when Terraform is requesting it.

If the response is successful (`200`-range status code), Terraform looks in
the following locations in order for the next address to access:

* The value of a response header field named `X-Terraform-Get`.

* If the response is an HTML page, a `meta` element with the name `terraform-get`:

  ```html
  <meta name="terraform-get"
        content="github.com/hashicorp/example" />
  ```

In either case, the result is interpreted as another module source address
using one of the forms documented elsewhere on this page.

If an HTTP/HTTPS URL requires authentication credentials, use a `.netrc`
file in your home directory to configure these. For information on this format,
see [the documentation for using it in `curl`](https://ec.haxx.se/usingcurl-netrc.html).

### Fetching archives over HTTP

As a special case, if Terraform detects that the URL has a common file
extension associated with an archive file format then it will bypass the
special `terraform-get=1` redirection described above and instead just use
the contents of the referenced archive as the module source code:

```hcl
module "vpc" {
  source = "https://example.com/vpc-module.zip"
}
```

The extensions that Terraform recognizes for this special behavior are:

* `zip`
* `tar.bz2` and `tbz2`
* `tar.gz` and `tgz`
* `tar.xz` and `txz`

If your URL _doesn't_ have one of these extensions but refers to an archive
anyway, use the `archive` argument to force this interpretation:

```hcl
module "vpc" {
  source = "https://example.com/vpc-module?archive=zip"
}
```

## S3 Bucket

You can use archives stored in S3 as module sources using the special `s3::`
prefix, followed by
[an S3 bucket object URL](http://docs.aws.amazon.com/AmazonS3/latest/dev/UsingBucket.html#access-bucket-intro).

```hcl
module "consul" {
  source = "s3::https://s3-eu-west-1.amazonaws.com/examplecorp-terraform-modules/vpc.zip"
}
```

The `s3::` prefix causes Terraform to use AWS-style authentication when
accessing the given URL. As a result, this scheme may also work for other
services that mimic the S3 API, as long as they handle authentication in the
same way as AWS.

The resulting object must be an archive with one of the same file
extensions as for [archives over standard HTTP](#fetching-archives-over-http).
Terraform will extract the archive to obtain the module source tree.

The module installer looks for AWS credentials in the following locations,
preferring those earlier in the list when multiple are available:

* The `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables.
* The default profile in the `.aws/credentials` file in your home directory.
* If running on an EC2 instance, temporary credentials associated with the
  instance's IAM Instance Profile.

## Modules in Package Sub-directories

When the source of a module is a version control repository or archive file
(generically, a "package"), the module itself may be in a sub-directory relative
to the root of the package.

A special double-slash syntax is interpreted by Terraform to indicate that
the remaining path after that point is a sub-directory within the package.
For example:

* `hashicorp/consul/aws//modules/consul-cluster`
* `git::https://example.com/network.git//modules/vpc`
* `https://example.com/network-module.zip//modules/vpc`
* `s3::https://s3-eu-west-1.amazonaws.com/examplecorp-terraform-modules/network.zip//modules/vpc`

If the source address has arguments, such as the `ref` argument supported for
the version control sources, the sub-directory portion must be _before_ those
arguments:

* `git::https://example.com/network.git//modules/vpc?ref=v1.2.0`

Terraform will still extract the entire package to local disk, but will read
the module from the subdirectory. As a result, it is safe for a module in
a sub-directory of a package to use [a local path](#local-paths) to another
module as long as it is in the _same_ package.
