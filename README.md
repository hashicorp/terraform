Terraform
=========

- Website: http://www.terraform.io
- IRC: `#terraform-tool` on Freenode
- Mailing list: [Google Groups](http://groups.google.com/group/terraform-tool)

![Terraform](https://raw.githubusercontent.com/hashicorp/terraform/master/website/source/assets/images/readme.png)

Terraform is a tool for building, changing, and versioning infrastructure safely and efficiently. Terraform can manage existing and popular service providers as well as custom in-house solutions.

The key features of Terraform are:

- **Infrastructure as Code**: Infrastructure is described using a high-level configuration syntax. This allows a blueprint of your datacenter to be versioned and treated as you would any other code. Additionally, infrastructure can be shared and re-used.

- **Execution Plans**: Terraform has a "planning" step where it generates an *execution plan*. The execution plan shows what Terraform will do when you call apply. This lets you avoid any surprises when Terraform manipulates infrastructure.

- **Resource Graph**: Terraform builds a graph of all your resources, and parallelizes the creation and modification of any non-dependent resources. Because of this, Terraform builds infrastructure as efficiently as possible, and operators get insight into dependencies in their infrastructure.

- **Change Automation**: Complex changesets can be applied to your infrastructure with minimal human interaction. With the previously mentioned execution plan and resource graph, you know exactly what Terraform will change and in what order, avoiding many possible human errors.

For more information, see the [introduction section](http://www.terraform.io/intro) of the Terraform website.

Getting Started & Documentation
-------------------------------

All documentation is available on the [Terraform website](http://www.terraform.io).

Developing Terraform
--------------------

If you wish to work on Terraform itself or any of its built-in providers, you'll first need [Go](http://www.golang.org) installed on your machine (version 1.6+ is *required*). Alternatively, you can use the Vagrantfile in the root of this repo to stand up a virtual machine with the appropriate dev tooling already set up for you.

For local dev first make sure Go is properly installed, including setting up a [GOPATH](http://golang.org/doc/code.html#GOPATH). You will also need to add `$GOPATH/bin` to your `$PATH`.

Next, using [Git](https://git-scm.com/), clone this repository into `$GOPATH/src/github.com/hashicorp/terraform`. All the necessary dependencies are either vendored or automatically installed, so you just need to type `make`. This will compile the code and then run the tests. If this exits with exit status 0, then everything is working!

```sh
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

If you're working on a specific provider and only wish to rebuild that provider, you can use the `plugin-dev` target. For example, to build only the Azure provider:

```sh
$ make plugin-dev PLUGIN=provider-azure
```

If you're working on the core of Terraform, and only wish to rebuild that without rebuilding providers, you can use the `core-dev` target. It is important to note that some types of changes may require both core and providers to be rebuilt - for example work on the RPC interface. To build just the core of Terraform:

```sh
$ make core-dev
```

### Dependencies

Terraform stores its dependencies under `vendor/`, which [Go 1.6+ will automatically recognize and load](https://golang.org/cmd/go/#hdr-Vendor_Directories). We use [`godep`](https://github.com/tools/godep) to manage the vendored dependencies.

Generally speaking, `godep` operations follow this pattern:

 1. Get current state of dependencies into your `$GOPATH` with `godep restore`.
 2. Make changes to the packages in `$GOPATH`.
 3. Tell `godep` to capture those changes in the Terraform repo.

If you're developing Terraform, there are a few tasks you might need to perform.

#### Adding a dependency

If you're adding a dependency, you'll need to vendor it in the same Pull Request as the code that depends on it. You should do this in a separate commit from your code, as makes PR review easier and Git history simpler to read in the future.

Because godep captures new dependencies from the local `$GOPATH`, you first need to `godep restore` from the master branch to ensure that the only diff is your new dependency.

Assuming your work is on a branch called `my-feature-branch`, the steps look like this:

```bash
# Get latest master branch's dependencies staged in local $GOPATH
git checkout master
git pull
godep restore -v

# Capture the new dependency referenced from my-feature-branch
git checkout my-feature-branch
git rebase master
godep save ./...

# There should now be a diff in `vendor/` with added files for your dependency,
# and a diff in Godeps/Godeps.json with metadata for your dependency.

# Make a commit with your new dependencies added
git add -A
git commit -m "vendor: Capture new dependency upstream-pkg"

# Push to your branch (may need -f if you rebased)
git push origin my-feature-branch
```

#### Updating a dependency

If you're updating an existing dependency, godep provides a specific command to snag the newer version from your `$GOPATH`.

```bash
# Get latest master branch's dependencies staged in local $GOPATH
git checkout master
git pull
godep restore -v

# Make your way to the dependency in question and checkout the target ref
pushd $GOPATH/src/github.com/some/dependency
git checkout v-1.next

# Head back to Terraform on a feature branch and update the dependncy to the
# version currently in your $GOPATH
popd
git checkout my-feature-branch
godep update github.com/some/dependency/...

# There should now be a diff in `vendor/` with changed files for your dependency,
# and a diff in Godeps/Godeps.json with metadata for the updated dependency.

# Make a commit with the updated dependency
git add -A
git commit -m "vendor: Update dependency upstream-pkg to 1.4.6"

# Push to your branch
git push origin my-feature-branch
```

### Acceptance Tests

Terraform has a comprehensive [acceptance
test](http://en.wikipedia.org/wiki/Acceptance_testing) suite covering the
built-in providers. Our [Contributing Guide](https://github.com/hashicorp/terraform/blob/master/.github/CONTRIBUTING.md) includes details about how and when to write and run acceptance tests in order to help contributions get accepted quickly.


### Cross Compilation and Building for Distribution

If you wish to cross-compile Terraform for another architecture, you can set the `XC_OS` and `XC_ARCH` environment variables to values representing the target operating system and architecture before calling `make`. The output is placed in the `pkg` subdirectory tree both expanded in a directory representing the OS/architecture combination and as a ZIP archive.

For example, to compile 64-bit Linux binaries on Mac OS X Linux, you can run:

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
