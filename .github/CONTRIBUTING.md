# Contributing to Terraform

---

This repository contains only Terraform core, which includes the command line
interface and the main graph engine. Providers are implemented as plugins that
each have their own repository in
[the `terraform-providers` organization](https://github.com/terraform-providers)
on GitHub. Instructions for developing each provider are in the associated
README file. For more information, see
[the provider development overview](https://www.terraform.io/docs/plugins/provider.html).

---

Terraform is an open source project and we appreciate contributions of various
kinds, including bug reports and fixes, enhancement proposals, documentation
updates, and user experience feedback.

To record a bug report, enhancement proposal, or give any other product
feedback, please [open a GitHub issue](https://github.com/hashicorp/terraform/issues/new/choose)
using the most appropriate issue template. Please do fill in all of the
information the issue templates request, because we've seen from experience that
this will maximize the chance that we'll be able to act on your feedback.

Please not that we _don't_ use GitHub issues for usage questions. If you have
a question about how to use Terraform in general or how to solve a specific
problem with Terraform, please start a topic in
[the Terraform community forum](https://discuss.hashicorp.com/c/terraform-core),
where both Terraform team members and community members participate in
discussions.

**All communication on GitHub, the community forum, and other HashiCorp-provided
communication channels is subject to
[the HashiCorp community guidelines](https://www.hashicorp.com/community-guidelines).**

## Terraform CLI/Core Development Environment

This repository contains the source code for Terraform CLI, which is the main
component of Terraform that contains the core Terraform engine.

The HashiCorp-maintained Terraform providers are also open source but are not
in this repository; instead, they are each in their own repository in
[the `terraform-providers` organization](https://github.com/terraform-providers)
on GitHub.

This repository also does not include the source code for some other parts of
the Terraform product including Terraform Cloud, Terraform Enterprise, and the
Terraform Registry. Those components are not open source, though if you have
feedback about them (including bug reports) please do feel free to
[open a GitHub issue on this repository](https://github.com/hashicorp/terraform/issues/new/choose).

---

If you wish to work on the Terraform CLI source code, you'll first need to
install the [Go](https://golang.org/) compiler and the version control system
[Git](https://git-scm.com/).

At this time the Terraform development environment is targeting only Linux and
Mac OS X systems. While Terraform itself is compatible with Windows,
unfortunately the unit test suite currently contains Unix-specific assumptions
around maximum path lengths, path separators, etc.

Refer to the file [`.go-version`](.go-version) to see which version of Go
Terraform is currently built with. Other versions will often work, but if you
run into any build or testing problems please try with the specific Go version
indicated. You can optionally simplify the installation of multiple specific
versions of Go on your system by installing
[`goenv`](https://github.com/syndbg/goenv), which reads `.go-version` and
automatically selects the correct Go version.

Use Git to clone this repository into a location of your choice. Terraform is
using [Go Modules](https://blog.golang.org/using-go-modules), and so you
should _not_ clone it inside your `GOPATH`.

Switch into the root directory of the cloned repository and build Terraform
using the Go toolchain in the standard way:

```
cd terraform
go install .
```

The first time you run the `go install` command, the Go toolchain will download
any library dependencies that you don't already have in your Go modules cache.
Subsequent builds will be faster because these dependencies will already be
available on your local disk.

Once the compilation process succeeds, you can find a `terraform` executable in
the Go executable directory. If you haven't overridden it with the `GOBIN`
environment variable, the executable directory is the `bin` directory inside
the directory returned by the following command:

```
go env GOPATH
```

If you are planning to make changes to the Terraform source code, you should
run the unit test suite before you start to make sure everything is initially
passing:

```
go test ./...
```

As you make your changes, you can re-run the above command to ensure that the
tests are _still_ passing. If you are working only on a specific Go package,
you can speed up your testing cycle by testing only that single package, or
packages under a particular package prefix:

```
go test ./command/...
go test ./addrs
```

## Acceptance Tests: Testing interactions with external services

Terraform's unit test suite is self-contained, using mocks and local files
to help ensure that it can run offline and is unlikely to be broken by changes
to outside systems.

However, several Terraform components interact with external services, such
as the automatic provider installation mechanism, the Terraform Registry,
Terraform Cloud, etc.

There are some optional tests in the Terraform CLI codebase that _do_ interact
with external services, which we collectively refer to as "acceptance tests".
You can enable these by setting the environment variable `TF_ACC=1` when
running the tests. We recommend focusing only on the specific package you
are working on when enabling acceptance tests, both because it can help the
test run to complete faster and because you are less likely to encounter
failures due to drift in systems unrelated to your current goal:

```
TF_ACC=1 go test ./internal/initwd
```

Because the acceptance tests depend on services outside of the Terraform
codebase, and because the acceptance tests are usually used only when making
changes to the systems they cover, it is common and expected that drift in
those external systems will cause test failures. Because of this, prior to
working on a system covered by acceptance tests it's important to run the
existing tests for that system in an _unchanged_ work tree first and respond
to any test failures that preexist, to avoid misinterpreting such failures as
bugs in your new changes.

## Generated Code

Some files in the Terraform CLI codebase are generated. In most cases, we
update these using `go generate`, which is the standard way to encapsulate
code generation steps in a Go codebase.

```
go generate ./...
```

Use `git diff` afterwards to inspect the changes and ensure that they are what
you expected.

Terraform includes generated Go stub code for the Terraform provider plugin
protocol, which is defined using Protocol Buffers. Because the Protocol Buffers
tools are not written in Go and thus cannot be automatically installed using
`go get`, we follow a different process for generating these, which requires
that you've already installed a suitable version of `protoc`:

```
make protobuf
```

## External Dependencies

Terraform uses Go Modules for dependency management, but currently uses
"vendoring" to include copies of all of the external library dependencies
in the Terraform repository to allow builds to complete even if third-party
dependency sources are unavailable.

Our dependency licensing policy for Terraform excludes proprietary licenses
and "copyleft"-style licenses. We accept the common Mozilla Public License v2,
MIT License, and BSD licenses. We will consider other open source licenses
in similar spirit to those three, but if you plan to include such a dependency
in a contribution we'd recommend opening a GitHub issue first to discuss what
you intend to implement and what dependencies it will require so that the
Terraform team can review the relevant licenses to for whether they meet our
licensing needs.

If you need to add a new dependency to Terraform or update the selected version
for an existing one, use `go get` from the root of the Terraform repository
as follows:

```
go get github.com/hashicorp/hcl/v2@2.0.0
```

This command will download the requested version (2.0.0 in the above example)
and record that version selection in the `go.mod` file. It will also record
checksums for the module in the `go.sum`.

To complete the dependency change, clean up any redundancy in the module
metadata files and resynchronize the `vendor` directory with the new package
selections by running the following commands:

```
go mod tidy
go mod vendor
```

To ensure that the vendoring has worked correctly, be sure to run the unit
test suite at least once in _vendoring_ mode, where Go will use the vendored
dependencies to build the test programs:

```
go test -mod=vendor ./...
```

Because dependency changes affect a shared, top-level file, they are more likely
than some other change types to become conflicted with other proposed changes
during the code review process. For that reason, and to make dependency changes
more visible in the change history, we prefer to record dependency changes as
separate commits that include only the results of the above commands and the
minimal set of changes to Terraform's own code for compatibility with the
new version:

```
git add go.mod go.sum vendor
git commit -m "vendor: go get github.com/hashicorp/hcl/v2@2.0.0"
```

You can then make use of the new or updated dependency in new code added in
subsequent commits.

## Proposing a Change

If you'd like to contribute a code change to Terraform, we'd love to review
a GitHub pull request.

In order to be respectful of the time of community contributors, we prefer to
discuss potential changes in GitHub issues prior to implementation. That will
allow us to give design feedback up front and set expectations about the scope
of the change, and, for larger changes, how best to approach the work such that
the Terraform team can review it and merge it along with other concurrent work.

If the bug you wish to fix or enhancement you wish to implement isn't already
covered by a GitHub issue that contains feedback from the Terraform team,
please do start a discussion (either in
[a new GitHub issue](https://github.com/hashicorp/terraform/issues/new/choose)
or an existing one, as appropriate) before you invest significant development
time. If you mention your intent to implement the change described in your
issue, the Terraform team can prioritize including implementation-related
feedback in the subsequent discussion.

At this time, we do not have a formal process for reviewing outside proposals
that significantly change Terraform's workflow, its primary usage patterns,
and its language. While we do hope to put such a thing in place in the future,
we wish to be up front with potential contributors that unfortunately we are
unlikely to be able to give prompt feedback for large proposals that could
entail a significant design phase, though we are still interested to hear about
your use-cases so that we can consider ways to meet them as part of other
larger projects.

Most changes will involve updates to the test suite, and changes to Terraform's
documentation. The Terraform team can advise on different testing strategies
for specific scenarios, and may ask you to revise the specific phrasing of
your proposed documentation prose to match better with the standard "voice" of
Terraform's documentation.

This repository is primarily maintained by a small team at HashiCorp along with
their other responsibilities, so unfortunately we cannot always respond
promptly to pull requests, particularly if they do not relate to an existing
GitHub issue where the Terraform team has already participated. We _are_
grateful for all contributions however, and will give feedback on pull requests
as soon as we're able.
