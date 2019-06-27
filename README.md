Terraform
=========

- Website: https://www.terraform.io
- [![Gitter chat](https://badges.gitter.im/hashicorp-terraform/Lobby.png)](https://gitter.im/hashicorp-terraform/Lobby)
- Mailing list: [Google Groups](http://groups.google.com/group/terraform-tool)

<img alt="Terraform" src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" width="600px">

Terraform is a tool for building, changing, and versioning infrastructure safely and efficiently. Terraform can manage existing and popular service providers as well as custom in-house solutions.

The key features of Terraform are:

- **Infrastructure as Code**: Infrastructure is described using a high-level configuration syntax. This allows a blueprint of your datacenter to be versioned and treated as you would any other code. Additionally, infrastructure can be shared and re-used.

- **Execution Plans**: Terraform has a "planning" step where it generates an *execution plan*. The execution plan shows what Terraform will do when you call apply. This lets you avoid any surprises when Terraform manipulates infrastructure.

- **Resource Graph**: Terraform builds a graph of all your resources, and parallelizes the creation and modification of any non-dependent resources. Because of this, Terraform builds infrastructure as efficiently as possible, and operators get insight into dependencies in their infrastructure.

- **Change Automation**: Complex changesets can be applied to your infrastructure with minimal human interaction. With the previously mentioned execution plan and resource graph, you know exactly what Terraform will change and in what order, avoiding many possible human errors.

For more information, see the [introduction section](http://www.terraform.io/intro) of the Terraform website.

Getting Started & Documentation
-------------------------------

If you're new to Terraform and want to get started creating infrastructure, please checkout our [Getting Started](https://www.terraform.io/intro/getting-started/install.html) guide, available on the [Terraform website](http://www.terraform.io).

All documentation is available on the [Terraform website](http://www.terraform.io):

  - [Intro](https://www.terraform.io/intro/index.html)
  - [Docs](https://www.terraform.io/docs/index.html)

Developing Terraform
--------------------

If you wish to work on Terraform itself or any of its built-in providers, you'll first need [Go](http://www.golang.org) installed on your machine (version 1.11+ is *required*).

This repository contains only Terraform core, which includes the command line interface and the main graph engine. Providers are implemented as plugins that each have their own repository in [the `terraform-providers` organization](https://github.com/terraform-providers) on GitHub. Instructions for developing each provider are in the associated README file. For more information, see [the provider development overview](https://www.terraform.io/docs/plugins/provider.html).

For local development of Terraform core, first make sure Go is properly installed and that a
[GOPATH](http://golang.org/doc/code.html#GOPATH) has been set. You will also need to add `$GOPATH/bin` to your `$PATH`.

Next, using [Git](https://git-scm.com/), clone this repository into `$GOPATH/src/github.com/hashicorp/terraform`.

You'll need to run `make tools` to install some required tools, then `make`.  This will compile the code and then run the tests. If this exits with exit status 0, then everything is working!
You only need to run `make tools` once (or when the tools change).

```sh
$ cd "$GOPATH/src/github.com/hashicorp/terraform"
$ make tools
$ make
```

To compile a development version of Terraform and the built-in plugins, run `make dev`. This will build everything using [gox](https://github.com/mitchellh/gox) and put Terraform binaries in the `bin` and `$GOPATH/bin` folders:

```sh
$ make dev
...
$ bin/terraform
...
```

If you're developing a specific package, you can run tests for just that package by specifying the `TEST` variable. For example below, only`terraform` package tests will be run.

```sh
$ make test TEST=./terraform
...
```

If you're working on a specific provider which has not been separated into an individual repository and only wish to rebuild that provider, you can use the `plugin-dev` target. For example, to build only the Test provider:

```sh
$ make plugin-dev PLUGIN=provider-test
```

### Dependencies

Terraform uses Go Modules for dependency management, but for the moment is
continuing to use Go 1.6-style vendoring for compatibility with tools that
have not yet been updated for full Go Modules support.

If you're developing Terraform, there are a few tasks you might need to perform.

#### Adding a dependency

If you're adding a dependency, you'll need to vendor it in the same Pull Request as the code that depends on it. You should do this in a separate commit from your code, as makes PR review easier and Git history simpler to read in the future.

To add a dependency:

Assuming your work is on a branch called `my-feature-branch`, the steps look like this:

1. Add an `import` statement to a suitable package in the Terraform code.

2. Run `go mod vendor` to download the latest version of the module containing
   the imported package into the `vendor/` directory, and update the `go.mod`
   and `go.sum` files.

3. Review the changes in git and commit them.

#### Updating a dependency

To update a dependency:

1. Run `go get -u module-path@version-number`, such as `go get -u github.com/hashicorp/hcl@2.0.0`

2. Run `go mod vendor` to update the vendored copy in the `vendor/` directory.

3. Review the changes in git and commit them.

### Acceptance Tests

Terraform has a comprehensive [acceptance
test](http://en.wikipedia.org/wiki/Acceptance_testing) suite covering the
built-in providers. Our [Contributing Guide](https://github.com/hashicorp/terraform/blob/master/.github/CONTRIBUTING.md) includes details about how and when to write and run acceptance tests in order to help contributions get accepted quickly.


### Cross Compilation and Building for Distribution

If you wish to cross-compile Terraform for another architecture, you can set the `XC_OS` and `XC_ARCH` environment variables to values representing the target operating system and architecture before calling `make`. The output is placed in the `pkg` subdirectory tree both expanded in a directory representing the OS/architecture combination and as a ZIP archive.

For example, to compile 64-bit Linux binaries on Mac OS X, you can run:

```sh
$ XC_OS=linux XC_ARCH=amd64 make bin
...
$ file pkg/linux_amd64/terraform
terraform: ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked, not stripped
```

`XC_OS` and `XC_ARCH` can be space separated lists representing different combinations of operating system and architecture. For example, to compile for both Linux and Mac OS X, targeting both 32- and 64-bit architectures, you can run:

```sh
$ XC_OS="linux darwin" XC_ARCH="386 amd64" make bin
...
$ tree ./pkg/ -P "terraform|*.zip"
./pkg/
├── darwin_386
│   └── terraform
├── darwin_386.zip
├── darwin_amd64
│   └── terraform
├── darwin_amd64.zip
├── linux_386
│   └── terraform
├── linux_386.zip
├── linux_amd64
│   └── terraform
└── linux_amd64.zip

4 directories, 8 files
```

_Note: Cross-compilation uses [gox](https://github.com/mitchellh/gox), which requires toolchains to be built with versions of Go prior to 1.5. In order to successfully cross-compile with older versions of Go, you will need to run `gox -build-toolchain` before running the commands detailed above._

#### Docker

When using docker you don't need to have any of the Go development tools installed and you can clone terraform to any location on disk (doesn't have to be in your $GOPATH).  This is useful for users who want to build `master` or a specific branch for testing without setting up a proper Go environment.

For example, run the following command to build terraform in a linux-based container for macOS.

```sh
docker run --rm -v $(pwd):/go/src/github.com/hashicorp/terraform -w /go/src/github.com/hashicorp/terraform -e XC_OS=darwin -e XC_ARCH=amd64 golang:latest bash -c "apt-get update && apt-get install -y zip && make bin"
```


## License
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bhttps%3A%2F%2Fgithub.com%2Fhashicorp%2Fterraform.svg?type=large)](https://app.fossa.io/projects/git%2Bhttps%3A%2F%2Fgithub.com%2Fhashicorp%2Fterraform?ref=badge_large)
