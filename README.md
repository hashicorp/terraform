# Terraform

* Website: http://www.terraform.io
* IRC: `#terraform` on Freenode
* Mailing list: [Google Groups](http://groups.google.com/group/terraform)

Terraform is a tool for building and changing infrastructure
safetly and efficiently.

## Developing Terraform

If you wish to work on Terraform itself or any of its built-in providers,
you'll first need [Go](http://www.golang.org) installed (version 1.2+ is
_required_). Make sure Go is properly installed, including setting up
a [GOPATH](http://golang.org/doc/code.html#GOPATH). Make sure Go is compiled
with cgo support. You can verify this by running `go env` and checking that
`CGOENABLED` is set to "1".

Next, install [Git](http://git-scm.com/), which is needed for some dependencies.

Finally, install [Gox](https://github.com/mitchellh/gox), which is used
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
