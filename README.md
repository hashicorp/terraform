# Terraform

* Website: http://www.terraform.io
* IRC: `#terraform-tool` on Freenode
* Mailing list: [Google Groups](http://groups.google.com/group/terraform-tool)

Terraform is a tool for building and changing infrastructure
safely and efficiently.

## Developing Terraform

If you wish to work on Terraform itself or any of its built-in providers,
you'll first need [Go](http://www.golang.org) installed (version 1.2+ is
_required_). Make sure Go is properly installed, including setting up
a [GOPATH](http://golang.org/doc/code.html#GOPATH). Make sure Go is compiled
with cgo support. You can verify this by running `go env` and checking that
`CGOENABLED` is set to "1".

Next, install [Git](http://git-scm.com/), which is needed for some dependencies.

Then, install [Gox](https://github.com/mitchellh/gox), which is used
as a compilation tool on top of Go:

    $ go get -u github.com/mitchellh/gox

Next, clone this repository into `$GOPATH/src/github.com/hashicorp/terraform`
and then just type `make`. This will compile some dependencies and then
run the tests. If this exits with exit status 0, then everything is working!

    $ make
    ...

To compile a development version of Terraform and the built-in plugins,
run `make dev`. This will put Terraform binaries in the `bin` folder:

    $ make dev
    ...
    $ bin/terraform
    ...


If you're developing a specific package, you can run tests for just that
package by specifying the `TEST` variable. For example below, only
`terraform` package tests will be run.

    $ make test TEST=./terraform
    ...

### Acceptance Tests

Terraform also has a comprehensive
[acceptance test](http://en.wikipedia.org/wiki/Acceptance_testing)
suite covering most of the major features of the built-in providers.

If you're working on a feature of a provider and want to verify it
is functioning (and hasn't broken anything else), we recommend running
the acceptance tests. Note that we _do not require_ that you run or
write acceptance tests to have a PR accepted. The acceptance tests
are just here for your convenience.

**Warning:** The acceptance tests create/destroy/modify _real resources_,
which may incur real costs. In the presence of a bug, it is technically
possible that broken providers could corrupt existing infrastructure
as well. Therefore, please run the acceptance providers at your own
risk. At the very least, we recommend running them in their own private
account for whatever provider you're testing.

To run the acceptance tests, invoke `make testacc`:

    $ make testacc TEST=./builtin/providers/aws TESTARGS='-run=VPC'
    ...

The `TEST` variable is required, and you should specify the folder where
the provider is. The `TESTARGS` variable is recommended to filter down
to a specific resource to test, since testing all of them at once can
take a very long time.

Acceptance tests typically require other environment variables to be
set for things such as access keys. The provider itself should error
early and tell you what to set, so it is not documented here.
