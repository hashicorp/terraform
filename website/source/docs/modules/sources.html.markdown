---
layout: "docs"
page_title: "Module Sources"
sidebar_current: "docs-modules-sources"
description: |-
  As documented in usage, the only required parameter when using a module is the `source` parameter which tells Terraform where the module can be found and what constraints to put on the module if any (such as branches for Git, versions, etc.).
---

# Module Sources

As documented in [usage](/docs/modules/usage.html), the only required
parameter when using a module is the `source` parameter which tells Terraform
where the module can be found and what constraints to put on the module
if any (such as branches for Git, versions, etc.).

Terraform manages modules for you: it downloads them, organizes them
on disk, checks for updates, etc. Terraform uses this source parameter for
the download/update of modules.

Terraform supports the following sources:

  * Local file paths

  * GitHub

  * BitBucket

  * Generic Git, Mercurial repositories

  * HTTP URLs

Each is documented further below.

## Local File Paths

The easiest source is the local file path. For maximum portability, this
should be a relative file path into a subdirectory. This allows you to
organize your Terraform configuration into modules within one repository,
for example.

An example is shown below:

```
module "consul" {
	source = "./consul"
}
```

Updates for file paths are automatic: when "downloading" the module
using the [get command](/docs/commands/get.html), Terraform will create
a symbolic link to the original directory. Therefore, any changes are
automatically instantly available.

## GitHub

Terraform will automatically recognize GitHub URLs and turn them into
the proper Git repository. The syntax is simple:

```
module "consul" {
	source = "github.com/hashicorp/example"
}
```

Subdirectories within the repository can also be referenced:

```
module "consul" {
	source = "github.com/hashicorp/example//subdir"
}
```

**Note:** The double-slash is important. It is what tells Terraform that
that is the separator for a subdirectory, and not part of the repository
itself.

GitHub source URLs will require that Git is installed on your system
and that you have the proper access to the repository.

You can use the same parameters to GitHub repositories as you can generic
Git repositories (such as tags or branches). See the documentation for generic
Git repositories for more information.

#### Private GitHub Repos<a id="private-github-repos"></a>

If you need Terraform to be able to fetch modules from private GitHub repos on
a remote machine (like a Atlas or a CI server), you'll need to provide
Terraform with credentials that can be used to authenticate as a user with read
access to the private repo.

First, create a [machine
user](https://developer.github.com/guides/managing-deploy-keys/#machine-users)
with access to read from the private repo in question, then embed this user's
credentials into the source field:

```
module "private-infra" {
  source = "git::https://MACHINE-USER:MACHINE-PASS@github.com/org/privatemodules//modules/foo"
}
```

Note that Terraform does not yet support interpolations in the `source` field,
so the machine username and password will have to be embedded directly into the
source string. You can track
[GH-1439](https://github.com/hashicorp/terraform/issues/1439) to learn when this
limitation is lifted.

## BitBucket

Terraform will automatically recognize BitBucket URLs and turn them into
the proper Git or Mercurial repository. An example:

```
module "consul" {
	source = "bitbucket.org/hashicorp/example"
}
```

Subdirectories within the repository can also be referenced:

```
module "consul" {
	source = "bitbucket.org/hashicorp/example//subdir"
}
```

**Note:** The double-slash is important. It is what tells Terraform that
that is the separator for a subdirectory, and not part of the repository
itself.

BitBucket URLs will require that Git or Mercurial is installed on your
system, depending on the source URL.

## Generic Git Repository

Generic Git repositories are also supported. The value of `source` in this
case should be a complete Git-compatible URL. Using Git requires that
Git is installed on your system. Example:

```
module "consul" {
	source = "git://hashicorp.com/module.git"
}
```

You can also use protocols such as HTTP or SSH, but you'll have to hint
to Terraform (using the forced source type syntax documented below) to use
Git:

```
// force https source
module "consul" {
	source = "git::https://hashicorp.com/module.git"
}

// force ssh source
module "ami" {
	source = "git::ssh://git@github.com/owner/repo.git"
}
```

URLs for Git repositories (of any protocol) support the following query
parameters:

  * `ref` - The ref to checkout. This can be a branch, tag, commit, etc.

An example of using these parameters is shown below:

```
module "consul" {
	source = "git::https://hashicorp.com/module.git?ref=master"
}
```

## Generic Mercurial Repository

Generic Mercurial repositories are supported. The value of `source` in this
case should be a complete Mercurial-compatible URL. Using Mercurial requires that
Mercurial is installed on your system. Example:

```
module "consul" {
	source = "hg::http://hashicorp.com/module.hg"
}
```

In the case of above, we used the forced source type syntax documented below.
Mercurial repositories require this.

URLs for Mercurial repositories (of any protocol) support the following query
parameters:

  * `rev` - The rev to checkout. This can be a branch, tag, commit, etc.

## HTTP URLs

Any HTTP endpoint can serve up Terraform modules. For HTTP URLs (SSL is
supported, as well), Terraform will make a GET request to the given URL.
An additional GET parameter `terraform-get=1` will be appended, allowing
you to optionally render the page differently when Terraform is requesting it.

Terraform then looks for the resulting module URL in the following order.

First, if a header `X-Terraform-Get` is present, then it should contain
the source URL of the actual module. This will be used.

If the header isn't present, Terraform will look for a `<meta>` tag
with the name of "terraform-get". The value will be used as the source
URL.

## Forced Source Type

In a couple places above, we've referenced "forced source type." Forced
source type is a syntax added to URLs that allow you to force a specific
method for download/updating the module. It is used to disambiguate URLs.

For example, the source "http://hashicorp.com/foo.git" could just as
easily be a plain HTTP URL as it might be a Git repository speaking the
HTTP protocol. The forced source type syntax is used to force Terraform
one way or the other.

Example:

```
module "consul" {
	source = "git::http://hashicorp.com/foo.git"
}
```

The above will force Terraform to get the module using Git, despite it
being an HTTP URL.

If a forced source type isn't specified, Terraform will match the exact
protocol if it supports it. It will not try multiple methods. In the case
above, it would've used the HTTP method.
