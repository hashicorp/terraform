---
layout: "intro"
page_title: "Installing Terraform"
sidebar_current: "gettingstarted-install"
---

# Install Terraform

Terraform must first be installed on every node that will be a member of a
Terraform cluster. To make installation easy, Terraform is distributed as a
[binary package](/downloads.html) for all supported platforms and
architectures. This page will not cover how to compile Terraform from
source.

## Installing Terraform

To install Terraform, find the [appropriate package](/downloads.html) for
your system and download it. Terraform is packaged as a "zip" archive.

After downloading Terraform, unzip the package. Copy the `terraform` binary to
somewhere on the PATH so that it can be executed. On Unix systems,
`~/bin` and `/usr/local/bin` are common installation directories,
depending on if you want to restrict the install to a single user or
expose it to the entire system. On Windows systems, you can put it wherever
you would like.

### OS X

If you are using [homebrew](http://brew.sh/#install) as a package manager,
than you can install terraform as simple as:
```
brew cask install terraform
```

if you are missing the [cask plugin](http://caskroom.io/) you can install it with:
```
brew install caskroom/cask/brew-cask
```

## Verifying the Installation

After installing Terraform, verify the installation worked by opening a new
terminal session and checking that `terraform` is available. By executing
`terraform` you should see help output similar to that below:

```
$ terraform
usage: terraform [--version] [--help] <command> [<args>]

Available commands are:
    agent          Runs a Terraform agent
    force-leave    Forces a member of the cluster to enter the "left" state
    info           Provides debugging information for operators
    join           Tell Terraform agent to join cluster
    keygen         Generates a new encryption key
    leave          Gracefully leaves the Terraform cluster and shuts down
    members        Lists the members of a Terraform cluster
    monitor        Stream logs from a Terraform agent
    version        Prints the Terraform version
```

If you get an error that `terraform` could not be found, then your PATH
environment variable was not setup properly. Please go back and ensure
that your PATH variable contains the directory where Terraform was installed.

Otherwise, Terraform is installed and ready to go!
