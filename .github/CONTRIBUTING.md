# Contributing to Terraform

**All communication on GitHub, the community forum, and other HashiCorp-provided communication channels is subject to [the HashiCorp community guidelines](https://www.hashicorp.com/community-guidelines).**

This repository contains Terraform core, which includes the command line interface and the main graph engine. 

Providers are implemented as plugins that each have their own repository linked from the [Terraform Registry index](https://registry.terraform.io/browse/providers). Instructions for developing each provider are usually in the associated README file. For more information, see [the provider development overview](https://www.terraform.io/docs/plugins/provider.html).

This document provides guidance on Terraform contribution recommended practices. It covers what we're looking for in order to help set expectations and help you get the most out of participation in this project. 

To report a bug, an enhancement proposal, or give any other product feedback, please [open a GitHub issue](https://github.com/hashicorp/terraform/issues/new/choose) using the most appropriate issue template. Please fill in all of the information the issue templates request. This will maximize our ability to act on your feedback.

---

<!-- MarkdownTOC autolink="true" -->

- [Introduction](#Introduction)
- [Contributing a Pull Request](#contributing-a-pull-request)
- [Proposing a Change](#proposing-a-change)
	- [Caveats & areas of special concern](#caveats--areas-of-special-concern)
		- [State Storage Backends](#state-storage-backends)
		- [Provisioners](#provisioners)
		- [Maintainers](#maintainers)
	- [Pull Request Lifecycle](#pull-request-lifecycle)
		- [Getting Your Pull Requests Merged Faster](#getting-your-pull-requests-merged-faster)
		- [Changelog entries](#changelog-entries)
		- [Create a change file using `changie`](#create-a-change-file-using-changie)
		- [Backport a PR to a past release](#backport-a-pr-to-a-past-release)
	- [PR Checks](#pr-checks)
- [Terraform CLI/Core Development Environment](#terraform-clicore-development-environment)
- [Acceptance Tests: Testing interactions with external services](#acceptance-tests-testing-interactions-with-external-services)
- [Generated Code](#generated-code)
- [External Dependencies](#external-dependencies)

<!-- /MarkdownTOC -->

## Introduction

One of the great things about publicly available source code is that you can dive into the project and help _build the thing_ you believe is missing. It's a wonderful and generous instinct. However, Terraform is a complex tool. Even simple changes can have a serious impact on other areas of the code and it can take some time to become familiar with the effects of even basic changes. The Terraform team is not immune to unintended and sometimes undesirable consequences. We take our work seriously, and appreciate the responsibility of maintaining software for a globally diverse community that relies on Terraform for workflows of all sizes and criticality. 

As a result of Terraform's complexity and high bar for stability, the most straightforward way to help with the Terraform project is to [file a feature request or bug report](https://github.com/hashicorp/terraform/issues/new/choose), following the template to fully express your desired use case. 

If you believe you can also implement the solution for your bug or feature, we request that you first discuss the proposed solution with the core maintainer team. This discussion happens in GitHub, on the issue you created to describe the bug or feature. This discussion gives the core team a chance to explore any missing best practices or unintended consequences of the proposed change. Participating in this discussion and getting the go-ahead from a core maintainer is the only way to ensure your code is reviewed for inclusion with the project. It is also possible that the proposed solution is not workable, and will save you time writing code that will not be used due to unforeseen unintended consequences. Please read the section [Proposing a Change](#proposing-a-change) for the full details on this process. 

(As a side note, this is how we work internally at HashiCorp as well. Changes are proposed internally via an RFC process, in which all impacted teams are able to review the proposed changes and give feedback before any code is written. Written communication of changes via the RFC process is a core pillar of our internal coordination.)


## Contributing a Pull Request

If you are a new contributor to Terraform, or looking to get started committing to the Terraform ecosystem, here are a couple of tips to get started. 

First, the easiest way to get started is to make fixes or improvements to the documentation. This can be done completely within GitHub, no need to even clone the project! 

Beyond documentation improvements, it is easiest to contribute to Terraform on the edges. If you are looking for a good starting place to contribute, finding and resolving issues in the providers is the best first step. These projects have huge breadth of coverage and are always looking for contributors to fix issues that might not otherwise get the attention of a maintainer. 

Closer to home, within the Terraform core repository, working in areas like functions or backends tend to have less harmful unintended interactions with the core of Terraform (but, also, are not currently a high priority to be reviewed, so please discuss any changes with the team before you start.) It gets more difficult to contribute as you get closer to the core functionality (e.g., manipulating the graph and core language features). For these types of changes, please start with the [Proposing a Change](#proposing-a-change) section to understand how we think about managing this process.

Once you are ready to write code, please see the section [Terraform CLI/Core Development Environment](#terraform-clicore-development-environment) to create your dev environment. Please read the documentation, and don't be afraid to ask questions in our [community forum](https://discuss.hashicorp.com/c/terraform-core/27). 

You may see the `Good First Issue` label on issues in the Terraform repository on GitHub. We use this label to maintain a list of issues for new internal core team members to ramp up the codebase. That said, if you are feeling particularly ambitious, you can follow our process to propose a solution. Other HashiCorp repositories (for example, https://github.com/hashicorp/terraform-provider-aws/) do use the `Good First Issue` to indicate good issues for external contributors to get started. 


## Proposing a Change

In order to be respectful of the time of community contributors, we aim to discuss potential changes in GitHub issues prior to implementation. That will allow us to give design feedback up front and set expectations about the scope of the change, and, for larger changes, how best to approach the work such that the Terraform team can review it and merge it along with other concurrent work.

If the bug you wish to fix or enhancement you wish to implement isn't already covered by a GitHub issue that contains feedback from the Terraform team, please do start a discussion (either in [a new GitHub issue](https://github.com/hashicorp/terraform/issues/new/choose) or an existing one, as appropriate) before you invest significant development time. If you mention your intent to implement the change described in your issue, the Terraform team can, as best as possible, prioritize including implementation-related feedback in the subsequent discussion.

At this time, we do not have a formal process for reviewing outside proposals that significantly change Terraform's workflow, its primary usage patterns, and its language. Additionally, some seemingly simple proposals can have deep effects across Terraform, which is why we strongly suggest starting with an issue-based proposal. Also, we do not normally accept minor changes in comments or help text.

For large proposals that could entail a significant design phase, we wish to be up front with potential contributors that, unfortunately, we are unlikely to be able to give prompt feedback. We are still interested to hear about your use-cases so that we can consider ways to meet them as part of other larger projects.

Most changes will involve updates to the test suite, and changes to Terraform's documentation. The Terraform team can advise on different testing strategies for specific scenarios, and may ask you to revise the specific phrasing of your proposed documentation prose to match better with the standard "voice" of Terraform's documentation.

We cannot always respond promptly to pull requests, particularly if they do not relate to an existing GitHub issue where the Terraform team has already participated and indicated willingness to work on the issue or accept PRs for the proposal. We *are* grateful for all contributions however, and will give feedback on pull requests as soon as we are able. 


### Caveats & areas of special concern

There are some areas of Terraform which are of special concern to the Terraform team. 

#### State Storage Backends

The Terraform team is not merging PRs for new state storage backends. Our priority regarding state storage backends is to find maintainers for existing backends and remove those backends without maintainers.

Please see the [CODEOWNERS](https://github.com/hashicorp/terraform/blob/main/CODEOWNERS) file for the status of a given backend. Community members with an interest in a particular backend are welcome to offer to maintain it.

In terms of setting expectations, there are three categories of backends in the Terraform repository: backends maintained by the core team (ex.: http); backends maintained by one of HashiCorp's provider teams (e.g. AWS S3, Azure, etc); and backends maintained by third party maintainers (ex.: Postgres, COS). 

* Backends maintained by the core team are unlikely to see accepted contributions. We are triaging incoming pull requests, but these are not highly prioritized against our other work. The smaller and more-contained the change, the more likely it will be reviewed (please see also [Proposing a Change](#proposing-a-change)).

* Backends maintained by one of HashiCorp's provider teams review contributions irregularly. There is no official commitment, typically once every few months one of the maintainers will review a number of backend PRs relating to their provider. The S3 and Azure backends tend to see the most on-going development. 

* Backends maintained by third-party maintainers are reviewed at the whim and availability of those maintainers. When the maintainer gives a positive code review to the pull request, the core team will do a review and merge the changes. 


#### Provisioners

Provisioners are an area of concern in Terraform for a number of reasons. Chiefly, they are often used in the place of configuration management tools or custom providers. 

There are two main types of provisioners in Terraform, the generic provisioners (`file`,`local-exec`, and `remote-exec`) and the tool-specific provisioners (`chef`, `habbitat`, `puppet` & `salt-masterless`). **The tool-specific provisioners [are deprecated](https://discuss.hashicorp.com/t/notice-terraform-to-begin-deprecation-of-vendor-tool-specific-provisioners-starting-in-terraform-0-13-4/13997).** In practice this means we will not be accepting PRs for these areas of the codebase. 

From our [documentation](https://www.terraform.io/docs/provisioners/index.html):

> ... they [...] add a considerable amount of complexity and uncertainty to Terraform usage.[...] we still recommend attempting to solve it [your problem] using other techniques first, and use provisioners only if there is no other option.

The Terraform team is in the process of building a way forward which continues to decrease reliance on provisioners. In the mean time however, as our documentation indicates, they are a tool of last resort. As such expect that PRs and issues for provisioners are not high in priority. 

Please see the [CODEOWNERS](https://github.com/hashicorp/terraform/blob/main/CODEOWNERS) file for the status of a given provisioner.


#### Maintainers

Maintainers are key contributors to our community project. They contribute their time and expertise and we ask that the community take extra special care to be mindful of this when interacting with them.

For code that has a listed maintainer or maintainers in our [CODEOWNERS](https://github.com/hashicorp/terraform/blob/main/CODEOWNERS) file, the Terraform team will highlight them for participation in PRs which relate to the area of code they maintain. The expectation is that a maintainer will review the code and work with the PR contributor before the code is merged by the Terraform team.

There is no expectation on response time for our maintainers; they may be indisposed for prolonged periods of time. Please be patient. Discussions on when code becomes "unmaintained" will be on a case-by-case basis. 

If an an unmaintained area of code interests you and you'd like to become a maintainer, you may simply make a PR against our [CODEOWNERS](https://github.com/hashicorp/terraform/blob/main/CODEOWNERS) file with your github handle attached to the approriate area. If there is a maintainer or team of maintainers for that area, please coordinate with them as necessary. 


### Pull Request Lifecycle

1. You are welcome to submit a [draft pull request](https://github.blog/2019-02-14-introducing-draft-pull-requests/) for commentary or review before it is fully completed. It's also a good idea to include specific questions or items you'd like feedback on.
2. Once you believe your pull request is ready to be merged you can create your pull request.
3. If your change is user-facing, add a short description in a [changelog entry](#changelog-entries).
4. When time permits Terraform's core team members will look over your contribution and either merge, or provide comments letting you know if there is anything left to do. It may take some time for us to respond. We may also have questions that we need answered about the code, either because something doesn't make sense to us or because we want to understand your thought process. We kindly ask that you do not target specific team members. 
5. If we have requested changes, you can either make those changes or, if you disagree with the suggested changes, we can have a conversation about our reasoning and agree on a path forward. This may be a multi-step process. Our view is that pull requests are a chance to collaborate, and we welcome conversations about how to do things better. It is the contributor's responsibility to address any changes requested. While reviewers are happy to give guidance, it is unsustainable for us to perform the coding work necessary to get a PR into a mergeable state.
6. Once all outstanding comments and checklist items have been addressed, your contribution will be merged! Merged PRs may or may not be included in the next release based on changes the Terraform teams deems as breaking or not. The core team takes care of updating the [CHANGELOG.md](https://github.com/hashicorp/terraform/blob/main/CHANGELOG.md) as they merge.
7. In some cases, we might decide that a PR should be closed without merging. We'll make sure to provide clear reasoning when this happens. Following the recommended process above is one of the ways to ensure you don't spend time on a PR we can't or won't merge.

#### Getting Your Pull Requests Merged Faster

It is much easier to review pull requests that are:

1. Well-documented: Try to explain in the pull request comments what your change does, why you have made the change, and provide instructions for how to produce the new behavior introduced in the pull request. If you can, provide screen captures or terminal output to show what the changes look like. This helps the reviewers understand and test the change.
2. Small: Try to only make one change per pull request. If you found two bugs and want to fix them both, that's *awesome*, but it's still best to submit the fixes as separate pull requests. This makes it much easier for reviewers to keep in their heads all of the implications of individual code changes, and that means the PR takes less effort and energy to merge. In general, the smaller the pull request, the sooner reviewers will be able to make time to review it.
3. Passing Tests: Based on how much time we have, we may not review pull requests which aren't passing our tests (look below for advice on how to run unit tests). If you need help figuring out why tests are failing, please feel free to ask, but while we're happy to give guidance it is generally your responsibility to make sure that tests are passing. If your pull request changes an interface or invalidates an assumption that causes a bunch of tests to fail, then you need to fix those tests before we can merge your PR.

If we request changes, try to make those changes in a timely manner. Otherwise, PRs can go stale and be a lot more work for all of us to merge in the future.

Even with everyone making their best effort to be responsive, it can be time-consuming to get a PR merged. It can be frustrating to deal with the back-and-forth as we make sure that we understand the changes fully. Please bear with us, and please know that we appreciate the time and energy you put into the project.

#### Changelog entries

If your PR's changes are not user-facing add the label `no-changelog-needed`. If this label isn't present and your PR doesn't include any change files a Github Action workflow will prompt you to add whichever is needed.

If your PR's changes are user-facing then you will need to add a change file in your PR. See the next section for how to create one. The change file will need to be created in the `.changes/v1.XX/` folder that matches the version number present in [version/VERSION on the main branch](https://github.com/hashicorp/terraform/blob/main/version/VERSION).

This is different if you are backporting your changes to an earlier release version. In that case, put the change file in the `.changes/v1.XX/` folder for the earliest version that the change is being backported into. For example if a PR was labelled 1.11-backport and 1.10-backport then the change file should be created in the `.changes/v1.10/` folder only.


#### Create a change file using `changie`

If your change is user-facing you can use `npx changie new` to create a new changelog entry via your terminal. The command is interactive and you will need to: select which kind of change you're introducing, provide a short description, and enter either the number of the GitHub issue your PR closes or your PR's number.

Make sure to select the correct kind of change:


| Change kind      | When to use |
|------------------|-------------|
| NEW FEATURES     | Use this if you've added new, separate functionality to Terraform. For example, introduction of ephemeral resources. |
| ENHANCEMENTS     | Use this if you've improved existing functionality in Terraform. Examples include: adding a new field to a remote-state backend, or adding a new environment variable to use when configuring Terraform. |
| BUG FIXES        | Use this if you've fixed a user-facing issue. Examples include: crash fixes, improvements to error feedback, regression fixes. |
| NOTES            | This is used for changes that are unlikely to cause user-facing issues but might have edge cases. For example, changes to how the Terraform binary is built. |
| UPGRADE NOTES    | Use this if you've introduced a change that forces users to take action when upgrading, or changes Terraform's behaviour notably. For example, deprecating a field on a remote-state backend or changing the output of Terraform operations. |
| BREAKING CHANGES | Use this if you've introduced a change that could make a valid Terraform configuration stop working after a user upgrades Terraform versions. This might be paired with an upgrade note change file. Examples include: removing a field on a remote-state backend, changing a builtin function's behavior, making validation stricter. |

#### Backport a PR to a past release

PRs can be backported to previous release version as part of preparing a patch release. For example, a fix for a bug could be merged into main but also backported to one or two previous minor versions.

If you want to backport your PR then the PR needs to have one or more [backport labels](https://github.com/hashicorp/terraform/labels?q=backport) added. The PR reviewer will then ensure that the PR is merged into those versions' release branches, as well as merged into `main`.

### PR Checks

The following checks run when a PR is opened:

- Contributor License Agreement (CLA): If this is your first contribution to Terraform you will be asked to sign the CLA.
- Tests: tests include unit tests and acceptance tests, and all tests must pass before a PR can be merged.
- Change files: PRs that include user-facing changes should include change files (see [Pull Request Lifecycle](#pull-request-lifecycle)). Automation will verify if PRs are labelled correctly and/or contain appropriate change files.
- Vercel: this is an internal tool that does not run correctly for external contributors. We are aware of this and work around it for external contributions. 

----

## Terraform CLI/Core Development Environment

This repository contains the source code for Terraform CLI, which is the main component of Terraform that contains the core Terraform engine.

Terraform providers are not maintained in this repository; you can find relevant
repository and relevant issue tracker for each provider within the
[Terraform Registry index](https://registry.terraform.io/browse/providers).

This repository also does not include the source code for some other parts of the Terraform product including HCP Terraform, Terraform Enterprise, and the Terraform Registry. The source for those components is not publicly available. If you have feedback about these products, including bug reports, please email [tf-cloud@hashicorp.support](mailto:tf-cloud@hashicorp.support) or [open a support request](https://support.hashicorp.com/hc/en-us/requests/new).

---

If you wish to work on the Terraform CLI source code, you'll first need to install the [Go](https://golang.org/) compiler and the version control system [Git](https://git-scm.com/).

At this time the Terraform development environment is targeting only Linux and Mac OS X systems. While Terraform itself is compatible with Windows, unfortunately the unit test suite currently contains Unix-specific assumptions around maximum path lengths, path separators, etc.

Refer to the file [`.go-version`](https://github.com/hashicorp/terraform/blob/main/.go-version) to see which version of Go Terraform is currently built with. As of Go 1.21, the `go` command (e.g. in `go build`) will automatically install the version of the Go toolchain corresponding to the version specified in `go.mod`, if it is newer than the version you have installed. The version in `go.mod` is considered the _minimum_ compatible Go version for Terraform, while the version in `.go-version` is what the production binary is actually built with.

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
go test ./internal/command/...
go test ./internal/addrs
```

## Acceptance Tests: Testing interactions with external services

Terraform's unit test suite is self-contained, using mocks and local files to help ensure that it can run offline and is unlikely to be broken by changes to outside systems.

However, several Terraform components interact with external services, such as the automatic provider installation mechanism, the Terraform Registry, HCP Terraform, Terraform Enterprise, etc.

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
