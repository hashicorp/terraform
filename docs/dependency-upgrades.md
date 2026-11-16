# Upgrading Terraform's Library Dependencies

This codebase depends on a variety of external Go modules. At the time of
writing, a Terraform CLI build includes the union of all dependencies required
by Terraform CLI and Core itself and each of the remote state backends.

Because the remote state backends have quite a different set of needs than
Terraform itself -- special client libraries, in particular -- we've declared
them as being separate Go Modules with their own `go.mod` and `go.sum` files,
even though we don't intend to ever publish them separately. The goal is just
to help track which dependencies are used by each of these components, and to
more easily determine which of the components are affected by a particular
dependency upgrade so that we can make sure to do the appropriate testing.

## Dependency Synchronization

Because all of these components ultimately link into the same executable, there
can be only one version of each distinct module and thus all of the modules
must agree on which version to use.

The makefile target `syncdeps` runs a script which synchronizes all of the
modules to declare compatible dependencies, selecting the newest version of
each external module selected across all of the internal modules:

```shell
make syncdeps
```

After running this, use `git status` to see what's been changed. If you've
changed the dependencies of any of the modules then that should typically
cause an update to the root module, because that one imports all of the others.

## Upgrading a Dependency

To select a newer version of one of the dependencies, use `go get` in the
root to specify which version to use:

```shell
go get example.com/foo/bar@v1.0.0
```

Or, if you just want to move to the latest stable release, you can use the
`latest` pseudo-version:

```shell
go get example.com/foo/bar@latest
```

Then run `make syncdeps` to update any of the child modules that also use
this dependency. The remote state backends use only a subset of the packages
in Terraform CLI/Core, so not all dependency updates will affect the remote
state backends, and an update might affect only a subset of the backends.

When you open the pull request for your change, our code owners rules will
automatically request review from the team that maintains any affected remote
state backend. The affected teams can judge whether the update seems likely
to affect their backend and run their acceptance tests if so, before approving
the pull request. As usual, these PRs should also be reviewed by at least
one member of the Terraform Core team since they are ultimately responsible
for the complete set of dependencies used in Terraform CLI releases.

**Note:** Currently our code owners rules are simplistic and will request
review for _any_ change under a remote state backend module directory, but
in practice an update that only changes a backend's `go.sum` cannot affect
the runtime behavior of the backend, and so those review requests are not
strictly required. You should therefore remove the review requests for
any backend whose only diff is the `go.sum` file once you've opened the
pull request.

## Dependabot Updates

When Dependabot automatically opens a pull request to upgrade a dependency,
unfortunately it isn't smart enough to automatically synchronize the change
across the modules and so the code consistency checks for the change will
typically fail.

To apply the proposed change, you'll need to check out the branch that
Dependabot created on your development system, run `make syncdeps`, add
all of the files that get modified, and then amend Dependabot's commit using
`git commit --amend`.

After you've done this, use `git push --force` to replace Dependabot's original
commit with your new commit, and then wait for GitHub to re-run the PR
checks. The code consistency checks should now pass.

We've configured Dependabot to monitor only the root `go.mod` file for potential
upgrades, because that one aggregates the dependencies for all other child
modules. Therefore there should never be a Dependabot upgrade targeting a
module in a subdirectory. If one _does_ get created somehow, you should close
it and perform the same upgrade at the root of the repository instead, using
the instructions in [Upgrading a Dependency](#upgrading-a-dependency) above.

## Dependencies with Special Requirements

Most of our dependencies can be treated generically, but a few have some
special constraints due to how Terraform uses them:

* HCL, cty, and their indirect dependencies `golang.org/x/text` and
  `github.com/apparentlymart/go-textseg` all include logic based on Unicode
  specifications, and so should be updated with care to make sure that
  Terraform's support for Unicode follows a consistent Unicode version
  throughout.

    Additionally, each time we adopt a new minor release of Go, we may need to
    upgrade some or all of these dependencies to match the Unicode version used
    by the Go standard library.

    For more information, refer to [How Terraform Uses Unicode](unicode.md).

    (This concern does not apply if the new version we're upgrading to is built
    for the same version of Unicode that Terraform was already using.)

* `github.com/hashicorp/go-getter` represents a significant part of Terraform
  CLI's remote module installer, and is the final interpreter of Terraform's
  module source address syntax. Because the module source address syntax is
  protected by the Terraform v1.x Compatibility Promises, for each upgrade
  we must make sure that:

    - The upgrade doesn't expand the source address syntax in a way that is
      undesirable from a Terraform product standpoint or in a way that we would
      not feel comfortable supporting indefinitely under the compatibility
      promises.
    - The upgrade doesn't break any already-supported source address forms
      that would therefore cause the next Terraform version to break the
      v1.x compatibility promises.

    Terraform's use of `go-getter` is all encapsulated in `internal/getmodules`
    and is set up to try to minimize the possibility that a go-getter upgrade
    would immediately introduce new functionality, but that encapsulation cannot
    prevent adoption of changes made to pre-existing functionality that
    Terraform already exposes.

* `github.com/hashicorp/go-tfe` -- the client library for the HCP Terraform
  API -- includes various types corresponding to HCP Terraform API
  requests and responses. The internal package `internal/cloud` contains mock
  implementations of some of those types, which may need to be updated when
  the client library is upgraded.

    These upgrades should typically be done only in conjunction with a project
    that will somehow use the new features through the Cloud integration, so
    that the team working on that project can perform any needed updates to
    the mocks as part of their work.

* `go.opentelemetry.io/otel` and the other supporting OpenTelemetry modules
  should typically be upgraded together in lockstep, because some of the
  modules define interfaces that other modules implement, and strange behavior
  can emerge if one is upgraded without the other.

    The main modules affected by this rule are the ones under the
    `go.opentelemetry.io/otel` prefix. The "contrib" packages can be trickier
    to upgrade because they tend to have dependencies that overlap with ours
    and so might affect non-telemetry-related behavior, and so it's acceptable
    for those to lag slightly behind to reduce risk in routine upgrades.
