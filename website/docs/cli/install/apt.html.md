---
layout: "downloads"
page_title: "APT Packages for Debian and Ubuntu"
sidebar_current: "docs-cli-install-apt"
description: |-
  The HashiCorp APT repositories contain distribution-specific Terraform packages for both Debian and Ubuntu systems.
---

# APT Packages for Debian and Ubuntu

The primary distribution packages for Terraform are `.zip` archives containing
single executable files that you can extract anywhere on your system. However,
for easier integration with configuration management tools and other systematic
system configuration strategies, we also offer package repositories for
Debian and Ubuntu systems, which allow you to install Terraform using the
`apt install` command or any other APT frontend.

If you are instead using Red Hat Enterprise Linux, CentOS, or Fedora, you
might prefer to [install Terraform from our Yum repositories](yum.html).

-> **Note:** The APT repositories discussed on this page are generic HashiCorp
repositories that contain packages for a variety of different HashiCorp
products, rather than just Terraform. Adding these repositories to your
system will, by default, therefore make several other non-Terraform
packages available for installation. That might then mask some packages that
are available for some HashiCorp products in the main Debian and Ubuntu
package repositories.

## Repository Configuration

The Terraform packages are signed using a private key controlled by HashiCorp,
so in most situations the first step would be to configure your system to trust
that HashiCorp key for package authentication. For example:

```bash
curl -fsSL https://apt.releases.hashicorp.com/gpg | sudo apt-key add -
```

After registering the key, you can add the official HashiCorp repository to
your system:

```bash
sudo apt-add-repository "deb [arch=$(dpkg --print-architecture)] https://apt.releases.hashicorp.com $(lsb_release -cs) main"
```

The above command line uses the following sub-shell commands:

* `dpkg --print-architecture` to determine your system's primary APT
  architecture/ABI, such as `amd64`.
* `lsb_release -cs` to find the distribution release codename for your current
  system, such as `buster`, `groovy`, or `sid`.

To install Terraform from the new repository:

```bash
sudo apt update
sudo apt install terraform
```

## Supported Architectures

The HashiCorp APT server currently has packages only for the `amd64`
architecture, which is also sometimes known as `x86_64`.

There are no official packages available for other architectures, such as
`arm64`. If you wish to use Terraform on a non-`amd64` system,
[download a normal release `.zip` file](/downloads.html) instead.

## Supported Debian and Ubuntu Releases

The HashiCorp APT server currently contains release repositories for the
following distribution releases:

* Debian 8 (`jessie`)
* Debian 9 (`stretch`)
* Debian 10 (`buster`)
* Debian 11 (`bullseye`)
* Ubuntu 16.04 (`xenial`)
* Ubuntu 18.04 (`bionic`)
* Ubuntu 19.10 (`eoam`)
* Ubuntu 20.04 (`focal`)
* Ubuntu 20.10 (`groovy`)
* Ubuntu 21.04 (`hirsute`)
* Ubuntu 21.10 (`impish`)

No repositories are available for other Debian or Ubuntu versions or
any other APT-based Linux distributions. If you add the repository using
the above commands on other systems then `apt update` will report the
repository index as missing.

Terraform executables are statically linked and so they depend only on the
Linux system call interface, not on any system libraries. Because of that,
you may be able to use one of the above release codenames when adding a
repository to your system, even if that codename doesn't match your current
distribution release.

Over time we will change the set of supported distributions, including both
adding support for new releases and ceasing to publish new Terraform versions
under older releases.

## Choosing Terraform Versions

The HashiCorp APT repositories contain multiple versions of Terraform, but
because the packages are all named `terraform` it is impossible to install
more than one version at a time, and `apt install` will default to selecting
the latest version.

It's often necessary to match your Terraform version with what a particular
configuration is currently expecting. You can use the following command to
see which versions are currently available in the repository index:

```bash
apt policy terraform
```

There may be multiple package releases for a particular Terraform version if
we need to publish an updated package for any reason. In that case, the
subsequent releases will have an additional suffix, like `0.13.4-2`. In these
cases, the Terraform executable inside the package should be unchanged, but its
metadata and other contents may be different.

You can select a specific version to install by including it in the
`apt install` command line, as follows:

```bash
sudo apt install terraform=0.14.0
```

If your workflow requires using multiple versions of Terraform at the same
time, for example when working through a gradual upgrade where not all
of your configurations are upgraded yet, we recommend that you use the
official release `.zip` files instead of the APT packages, so you can install
multiple versions at once and then select which to use for each command you
run.
