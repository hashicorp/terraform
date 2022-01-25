# Contributing to Terraform

This repository contains only Terraform core, which includes the command line interface and the main graph engine. Providers are implemented as plugins that each have their own repository linked from the [Terraform Registry index](https://registry.terraform.io/browse/providers). Instructions for developing each provider are usually in the associated README file. For more information, see [the provider development overview](https://www.terraform.io/docs/plugins/provider.html).

---

**Note:** Due to current low staffing on the Terraform Core team at HashiCorp, **we are not routinely reviewing and merging community-submitted pull requests**. We do hope to begin processing them again soon once we're back up to full staffing again, but for the moment we need to ask for patience. Thanks!

**Additional note:**  The intent of the prior comment was to provide clarity for the community around what to expect for a small part of the work related to Terraform. This does not affect other PR reviews, such as those for Terraform providers. We expect that the relevant team will be appropriately staffed within the coming weeks, which should allow us to get back to normal community PR review practices. For the broader context and information on HashiCorp’s continued commitment to and investment in Terraform, see [this blog post](https://www.hashicorp.com/blog/terraform-community-contributions).

---

**All communication on GitHub, the community forum, and other HashiCorp-provided communication channels is subject to [the HashiCorp community guidelines](https://www.hashicorp.com/community-guidelines).**

This document provides guidance on Terraform contribution recommended practices. It covers what we're looking for in order to help set some expectations and help you get the most out of participation in this project. 

To record a bug report, enhancement proposal, or give any other product feedback, please [open a GitHub issue](https://github.com/hashicorp/terraform/issues/new/choose) using the most appropriate issue template. Please do fill in all of the information the issue templates request, because we've seen from experience that this will maximize the chance that we'll be able to act on your feedback.

---

<!-- MarkdownTOC autolink="true" -->

- [Contributing Fixes](#contributing-fixes)
- [Proposing a Change](#proposing-a-change)
	- [Caveats & areas of special concern](#caveats--areas-of-special-concern)
		- [State Storage Backends](#state-storage-backends)
		- [Provisioners](#provisioners)
		- [Maintainers](#maintainers)
	- [Pull Request Lifecycle](#pull-request-lifecycle)
		- [Getting Your Pull Requests Merged Faster](#getting-your-pull-requests-merged-faster)
	- [PR Checks](#pr-checks)
- [Terraform CLI/Core Development Environment](#terraform-clicore-development-environment)
- [Acceptance Tests: Testing interactions with external services](#acceptance-tests-testing-interactions-with-external-services)
- [Generated Code](#generated-code)
- [External Dependencies](#external-dependencies)

<!-- /MarkdownTOC -->

## Contributing Fixes

It can be tempting to want to dive into an open source project and help _build the thing_ you believe you're missing. It's a wonderful and helpful intention. However, Terraform is a complex tool. Many seemingly simple changes can have serious effects on other areas of the code and it can take some time to become familiar with the effects of even basic changes. The Terraform team is not immune to unintended and sometimes undesirable changes. We do take our work seriously, and appreciate the globally diverse community that relies on Terraform for workflows of all sizes and criticality. 

As a result of Terraform's complexity and high bar for stability, the most straightforward way to start helping with the Terraform project is to pick an existing bug and [get to work](#terraform-clicore-development-environment). 

For new contributors we've labeled a few issues with `Good First Issue` as a nod to issues which will help get you familiar with Terraform development, while also providing an onramp to the codebase itself.

Read the documentation, and don't be afraid to [ask questions](https://discuss.hashicorp.com/c/terraform-core/27). 

## Proposing a Change

In order to be respectful of the time of community contributors, we aim to discuss potential changes in GitHub issues prior to implementation. That will allow us to give design feedback up front and set expectations about the scope of the change, and, for larger changes, how best to approach the work such that the Terraform team can review it and merge it along with other concurrent work.

If the bug you wish to fix or enhancement you wish to implement isn't already covered by a GitHub issue that contains feedback from the Terraform team, please do start a discussion (either in [a new GitHub issue](https://github.com/hashicorp/terraform/issues/new/choose) or an existing one, as appropriate) before you invest significant development time. If you mention your intent to implement the change described in your issue, the Terraform team can, as best as possible, prioritize including implementation-related feedback in the subsequent discussion.

At this time, we do not have a formal process for reviewing outside proposals that significantly change Terraform's workflow, its primary usage patterns, and its language. Additionally, some seemingly simple proposals can have deep effects across Terraform, which is why we strongly suggest starting with an issue-based proposal. 

For large proposals that could entail a significant design phase, we wish to be up front with potential contributors that, unfortunately, we are unlikely to be able to give prompt feedback. We are still interested to hear about your use-cases so that we can consider ways to meet them as part of other larger projects.

Most changes will involve updates to the test suite, and changes to Terraform's documentation. The Terraform team can advise on different testing strategies for specific scenarios, and may ask you to revise the specific phrasing of your proposed documentation prose to match better with the standard "voice" of Terraform's documentation.

This repository is primarily maintained by a small team at HashiCorp along with their other responsibilities, so unfortunately we cannot always respond promptly to pull requests, particularly if they do not relate to an existing GitHub issue where the Terraform team has already participated and indicated willingness to work on the issue or accept PRs for the proposal. We *are* grateful for all contributions however, and will give feedback on pull requests as soon as we're able.

### Caveats & areas of special concern

There are some areas of Terraform which are of special concern to the Terraform team. 

#### State Storage Backends

The Terraform team is not merging PRs for new state storage backends at the current time. Our priority regarding state storage backends is to find maintainers for existing backends and remove those backends without maintainers.

Please see the [CODEOWNERS](https://github.com/hashicorp/terraform/blob/main/CODEOWNERS) file for the status of a given backend. Community members with an interest in a particular standard backend are welcome to help maintain it.

Currently, merging state storage backends places a significant burden on the Terraform team. The team must set up an environment and cloud service provider account, or a new database/storage/key-value service, in order to build and test remote state storage backends. The time and complexity of doing so prevents us from moving Terraform forward in other ways.

We are working to remove ourselves from the critical path of state storage backends by moving them towards a plugin model. In the meantime, we won't be accepting new remote state backends into Terraform.

#### Provisioners

Provisioners are an area of concern in Terraform for a number of reasons. Chiefly, they are often used in the place of configuration management tools or custom providers. 

There are two main types of provisioners in Terraform, the generic provisioners (`file`,`local-exec`, and `remote-exec`) and the tool-specific provisioners (`chef`, `habbitat`, `puppet` & `salt-masterless`). **The tool-specific provisioners [are deprecated](https://discuss.hashicorp.com/t/notice-terraform-to-begin-deprecation-of-vendor-tool-specific-provisioners-starting-in-terraform-0-13-4/13997).** In practice this means we will not be accepting PRs for these areas of the codebase. 

From our [documentation](https://www.terraform.io/docs/provisioners/index.html):

> ... they [...] add a considerable amount of complexity and uncertainty to Terraform usage.[...] we still recommend attempting to solve it [your problem] using other techniques first, and use provisioners only if there is no other option.

The Terraform team is in the process of building a way forward which continues to decrease reliance on provisioners. In the mean time however, as our documentation indicates, they are a tool of last resort. As such expect that PRs and issues for provisioners are not high in priority. 

Please see the [CODEOWNERS](https://github.com/hashicorp/terraform/blob/main/CODEOWNERS) file for the status of a given provisioner. Community members with an interest in a particular provisioner are welcome to help maintain it.

#### Maintainers

Maintainers are key contributors to our Open Source project. They contribute their time and expertise and we ask that the community take extra special care to be mindful of this when interacting with them.

For code that has a listed maintainer or maintainers in our [CODEOWNERS](https://github.com/hashicorp/terraform/blob/main/CODEOWNERS) file, the Terraform team will highlight them for participation in PRs which relate to the area of code they maintain. The expectation is that a maintainer will review the code and work with the PR contributor before the code is merged by the Terraform team.

There is no expectation on response time for our maintainers; they may be indisposed for prolonged periods of time. Please be patient. Discussions on when code becomes "unmaintained" will be on a case-by-case basis. 

If an an unmaintained area of code interests you and you'd like to become a maintainer, you may simply make a PR against our [CODEOWNERS](https://github.com/hashicorp/terraform/blob/main/CODEOWNERS) file with your github handle attached to the approriate area. If there is a maintainer or team of maintainers for that area, please coordinate with them as necessary. 

### Pull Request Lifecycle

1. You are welcome to submit a [draft pull request](https://github.blog/2019-02-14-introducing-draft-pull-requests/) for commentary or review before it is fully completed. It's also a good idea to include specific questions or items you'd like feedback on.
2. Once you believe your pull request is ready to be merged you can create your pull request.
3. When time permits Terraform's core team members will look over your contribution and either merge, or provide comments letting you know if there is anything left to do. It may take some time for us to respond. We may also have questions that we need answered about the code, either because something doesn't make sense to us or because we want to understand your thought process. We kindly ask that you do not target specific team members. 
4. If we have requested changes, you can either make those changes or, if you disagree with the suggested changes, we can have a conversation about our reasoning and agree on a path forward. This may be a multi-step process. Our view is that pull requests are a chance to collaborate, and we welcome conversations about how to do things better. It is the contributor's responsibility to address any changes requested. While reviewers are happy to give guidance, it is unsustainable for us to perform the coding work necessary to get a PR into a mergeable state.
5. Once all outstanding comments and checklist items have been addressed, your contribution will be merged! Merged PRs may or may not be included in the next release based on changes the Terraform teams deems as breaking or not. The core team takes care of updating the [CHANGELOG.md](https://github.com/hashicorp/terraform/blob/main/CHANGELOG.md) as they merge.
6. In some cases, we might decide that a PR should be closed without merging. We'll make sure to provide clear reasoning when this happens. Following the recommended process above is one of the ways to ensure you don't spend time on a PR we can't or won't merge.

#### Getting Your Pull Requests Merged Faster

It is much easier to review pull requests that are:

1. Well-documented: Try to explain in the pull request comments what your change does, why you have made the change, and provide instructions for how to produce the new behavior introduced in the pull request. If you can, provide screen captures or terminal output to show what the changes look like. This helps the reviewers understand and test the change.
2. Small: Try to only make one change per pull request. If you found two bugs and want to fix them both, that's *awesome*, but it's still best to submit the fixes as separate pull requests. This makes it much easier for reviewers to keep in their heads all of the implications of individual code changes, and that means the PR takes less effort and energy to merge. In general, the smaller the pull request, the sooner reviewers will be able to make time to review it.
3. Passing Tests: Based on how much time we have, we may not review pull requests which aren't passing our tests (look below for advice on how to run unit tests). If you need help figuring out why tests are failing, please feel free to ask, but while we're happy to give guidance it is generally your responsibility to make sure that tests are passing. If your pull request changes an interface or invalidates an assumption that causes a bunch of tests to fail, then you need to fix those tests before we can merge your PR.

If we request changes, try to make those changes in a timely manner. Otherwise, PRs can go stale and be a lot more work for all of us to merge in the future.

Even with everyone making their best effort to be responsive, it can be time-consuming to get a PR merged. It can be frustrating to deal with the back-and-forth as we make sure that we understand the changes fully. Please bear with us, and please know that we appreciate the time and energy you put into the project.

### PR Checks

The following checks run when a PR is opened:

- Contributor License Agreement (CLA): If this is your first contribution to Terraform you will be asked to sign the CLA.
- Tests: tests include unit tests and acceptance tests, and all tests must pass before a PR can be merged.

----

## Terraform CLI/Core Development Environment

This repository contains the source code for Terraform CLI, which is the main component of Terraform that contains the core Terraform engine.

Terraform providers are not maintained in this repository; you can find relevant
repository and relevant issue tracker for each provider within the
[Terraform Registry index](https://registry.terraform.io/browse/providers).

This repository also does not include the source code for some other parts of the Terraform product including Terraform Cloud, Terraform Enterprise, and the Terraform Registry. Those components are not open source, though if you have feedback about them (including bug reports) please do feel free to [open a GitHub issue on this repository](https://github.com/hashicorp/terraform/issues/new/choose).

---

If you wish to work on the Terraform CLI source code, you'll first need to install the [Go](https://golang.org/) compiler and the version control system [Git](https://git-scm.com/).

At this time the Terraform development environment is targeting only Linux and Mac OS X systems. While Terraform itself is compatible with Windows, unfortunately the unit test suite currently contains Unix-specific assumptions around maximum path lengths, path separators, etc.

Refer to the file [`.go-version`](https://github.com/hashicorp/terraform/blob/main/.go-version) to see which version of Go Terraform is currently built with. Other versions will often work, but if you run into any build or testing problems please try with the specific Go version indicated. You can optionally simplify the installation of multiple specific versions of Go on your system by installing [`goenv`](https://github.com/syndbg/goenv), which reads `.go-version` and automatically selects the correct Go version.

Use Git to clone this repository into a location of your choice. Terraform is using [Go Modules](https://blog.golang.org/using-go-modules), and so you should *not* clone it inside your `GOPATH`.

Switch into the root directory of the cloned repository and build Terraform using the Go toolchain in the standard way:

```
cd terraform
go install .
```

The first time you run the `go install` command, the Go toolchain will download any library dependencies that you don't already have in your Go modules cache. Subsequent builds will be faster because these dependencies will already be available on your local disk.

Once the compilation process succeeds, you can find a `terraform` executable in the Go executable directory. If you haven't overridden it with the `GOBIN` environment variable, the executable directory is the `bin` directory inside the directory returned by the following command:

```
go env GOPATH
```

If you are planning to make changes to the Terraform source code, you should run the unit test suite before you start to make sure everything is initially passing:

```
go test ./...
```

As you make your changes, you can re-run the above command to ensure that the tests are *still* passing. If you are working only on a specific Go package, you can speed up your testing cycle by testing only that single package, or packages under a particular package prefix:

```
go test ./command/...
go test ./addrs
```

## Acceptance Tests: Testing interactions with external services

Terraform's unit test suite is self-contained, using mocks and local files to help ensure that it can run offline and is unlikely to be broken by changes to outside systems.

However, several Terraform components interact with external services, such as the automatic provider installation mechanism, the Terraform Registry, Terraform Cloud, etc.

There are some optional tests in the Terraform CLI codebase that *do* interact with external services, which we collectively refer to as "acceptance tests". You can enable these by setting the environment variable `TF_ACC=1` when running the tests. We recommend focusing only on the specific package you are working on when enabling acceptance tests, both because it can help the test run to complete faster and because you are less likely to encounter failures due to drift in systems unrelated to your current goal:

```
TF_ACC=1 go test ./internal/initwd
```

Because the acceptance tests depend on services outside of the Terraform codebase, and because the acceptance tests are usually used only when making changes to the systems they cover, it is common and expected that drift in those external systems will cause test failures. Because of this, prior to working on a system covered by acceptance tests it's important to run the existing tests for that system in an *unchanged* work tree first and respond to any test failures that preexist, to avoid misinterpreting such failures as bugs in your new changes.

## Generated Code

Some files in the Terraform CLI codebase are generated. In most cases, we update these using `go generate`, which is the standard way to encapsulate code generation steps in a Go codebase.

```
go generate ./...
```

Use `git diff` afterwards to inspect the changes and ensure that they are what you expected.

Terraform includes generated Go stub code for the Terraform provider plugin protocol, which is defined using Protocol Buffers. Because the Protocol Buffers tools are not written in Go and thus cannot be automatically installed using `go get`, we follow a different process for generating these, which requires that you've already installed a suitable version of `protoc`:

```
make protobuf
```

## External Dependencies

Terraform uses Go Modules for dependency management.

Our dependency licensing policy for Terraform excludes proprietary licenses and "copyleft"-style licenses. We accept the common Mozilla Public License v2, MIT License, and BSD licenses. We will consider other open source licenses in similar spirit to those three, but if you plan to include such a dependency in a contribution we'd recommend opening a GitHub issue first to discuss what you intend to implement and what dependencies it will require so that the Terraform team can review the relevant licenses to for whether they meet our licensing needs.

If you need to add a new dependency to Terraform or update the selected version for an existing one, use `go get` from the root of the Terraform repository as follows:

```
go get github.com/hashicorp/hcl/v2@2.0.0
```

This command will download the requested version (2.0.0 in the above example) and record that version selection in the `go.mod` file. It will also record checksums for the module in the `go.sum`.

To complete the dependency change, clean up any redundancy in the module metadata files by running:

```
go mod tidy
```

To ensure that the upgrade has worked correctly, be sure to run the unit test suite at least once:

```
go test ./...
```

Because dependency changes affect a shared, top-level file, they are more likely than some other change types to become conflicted with other proposed changes during the code review process. For that reason, and to make dependency changes more visible in the change history, we prefer to record dependency changes as separate commits that include only the results of the above commands and the minimal set of changes to Terraform's own code for compatibility with the new version:

```
git add go.mod go.sum
git commit -m "go get github.com/hashicorp/hcl/v2@2.0.0"
```

You can then make use of the new or updated dependency in new code added in subsequent commits.
