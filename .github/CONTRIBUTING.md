# Contributing to Terraform

**First:** if you're unsure or afraid of _anything_, just ask
or submit the issue or pull request anyways. You won't be yelled at for
giving your best effort. The worst that can happen is that you'll be
politely asked to change something. We appreciate any sort of contributions,
and don't want a wall of rules to get in the way of that.

However, for those individuals who want a bit more guidance on the
best way to contribute to the project, read on. This document will cover
what we're looking for. By addressing all the points we're looking for,
it raises the chances we can quickly merge or address your contributions.

Specifically, we have provided checklists below for each type of issue and pull
request that can happen on the project. These checklists represent everything
we need to be able to review and respond quickly.

## HashiCorp, Official, and Community Providers

We separate providers out into what we call "HashiCorp Providers", "Partner Providers" and "Community Providers".

HashiCorp providers are providers that we dedicate full time engineers to
improving, supporting the latest features, and fixing bugs. These are providers
we understand deeply and are confident we have the resources to manage
ourselves.

Partner providers are providers where we depend on our partners to
contribute fixes and enhancements to improve. HashiCorp will run automated
tests and ensure these providers continue to work, but will not dedicate full
time engineers to add new features to these providers. These providers are
available in official Terraform releases, but the functionality is primarily
contributed.

All HashiCorp and Partner providers can be found in the (terraform-providers github organization)[https://github.com/terraform-providers].
Any provider issues should be opened in the provider's repository.

Our testing standards are the same for both HashiCorp and Official providers,
and HashiCorp runs full acceptance test suites for every provider nightly to
ensure Terraform remains stable.

Community Providers are providers that are neither maintained nor tested by
HashiCorp. We can make no promises that these providers will work with any given
version of Terraform. These providers are not automatically installed by
`terraform init` and instead require manual installation.

We make the distinction between these types of providers to help
highlight the vast amounts of community effort that goes in to making Terraform
great, and to help contributors better understand the role HashiCorp employees
play in the various areas of the code base.

## Issues

### Issue Reporting Checklists

We welcome feature requests and bug reports. Below you'll find checklists with
guidelines for well-formed issues of each type.

#### Bug Reports

 - [ ] __Test against latest release__: Make sure you test against the latest
   released version. It is possible we already fixed the bug you're experiencing.

 - [ ] __Search for possible duplicate reports__: It's helpful to keep bug
   reports consolidated to one thread, so do a quick search on existing bug
   reports to check if anybody else has reported the same thing. You can scope
   searches by the label "bug" to help narrow things down.

 - [ ] __Include steps to reproduce__: Provide steps to reproduce the issue,
   along with your `.tf` files, with secrets removed, so we can try to
   reproduce it. Without this, it makes it much harder to fix the issue.

 - [ ] __For panics, include `crash.log`__: If you experienced a panic, please
   create a [gist](https://gist.github.com) of the *entire* generated crash log
   for us to look at. Double check no sensitive items were in the log.

#### Feature Requests

 - [ ] __Search for possible duplicate requests__: It's helpful to keep requests
   consolidated to one thread, so do a quick search on existing requests to
   check if anybody else has reported the same thing. You can scope searches by
   the label "enhancement" to help narrow things down.

 - [ ] __Include a use case description__: In addition to describing the
   behavior of the feature you'd like to see added, it's helpful to also lay
   out the reason why the feature would be important and how it would benefit
   Terraform users.

#### Questions

Please do not use GitHub to ask questions! Instead:

 * __Search for answers in Terraform documentation__

 * __Ask in the Community Forum__: Use [the community forum](https://discuss.hashicorp.com/c/terraform-core) for questions not answered by the documentation.

 * __Request an update to the documentation__: If you find that the
 documentation is confusing or incorrect, open an issue (or a pull request) and
 let us know.

### Issue Lifecycle

1. The issue is reported.

2. The issue is verified and categorized by a Terraform collaborator.
   Categorization is done via GitHub labels. We generally use a two-label
   system of (1) issue/PR type, and (2) section of the codebase. Type is
   usually "bug", "enhancement", "documentation", or "question", and section
   can be any of the providers or provisioners or "core".

3. Unless it is critical, the issue is left for a period of time (sometimes
   many weeks), giving outside contributors a chance to address the issue.

4. The issue is addressed in a pull request or commit. The issue will be
   referenced in the commit message so that the code that fixes it is clearly
   linked.

5. The issue is closed. Sometimes, valid issues will be closed to keep
   the issue tracker clean. The issue is still indexed and available for
   future viewers, or can be re-opened if necessary.

## Pull Requests

Thank you for contributing! Here you'll find information on what to include in
your Pull Request to ensure it is accepted quickly.

 * Pull requests that don't follow the guidelines will be annotated with what
   they're missing. A community or core team member may be able to swing around
   and help finish up the work, but these PRs will generally hang out much
   longer until they can be completed and merged.

### Pull Request Lifecycle

1. You are welcome to submit your pull request for commentary or review before
   it is fully completed. Please prefix the title of your pull request with
   "[WIP]" to indicate this. It's also a good idea to include specific
   questions or items you'd like feedback on.

2. Once you believe your pull request is ready to be merged, you can remove any
   "[WIP]" prefix from the title and a core team member will review. Follow
   [the checklists below](#checklists-for-contribution) to help ensure that
   your contribution will be merged quickly.

3. One of Terraform's core team members will look over your contribution and
   either provide comments letting you know if there is anything left to do. We
   do our best to provide feedback in a timely manner, but it may take some
   time for us to respond.

4. Once all outstanding comments and checklist items have been addressed, your
   contribution will be merged! Merged PRs will be included in the next
   Terraform release. The core team takes care of updating the CHANGELOG as
   they merge.

5. In rare cases, we might decide that a PR should be closed. We'll make sure
   to provide clear reasoning when this happens.

### Checklists for Contribution

There are several different kinds of contribution, each of which has its own
standards for a speedy review. The following sections describe guidelines for
each type of contribution.

#### Documentation Update

Because [Terraform's website][website] is in the same repo as the code, it's
easy for anybody to help us improve our docs.

 - [ ] __Reasoning for docs update__: Including a quick explanation for why the
   update needed is helpful for reviewers.
 - [ ] __Relevant Terraform version__: Is this update worth deploying to the
   site immediately, or is it referencing an upcoming version of Terraform and
   should get pushed out with the next release?

#### New Provider

Implementing a new provider gives Terraform the ability to manage resources in
a whole new API. It's a larger undertaking, but brings major new functionality
into Terraform.

Terraform Providers are external plugins, not in the Terraform codebase. Please
see the [Provider Development Program](https://www.terraform.io/guides/terraform-provider-development-program.html) documentation if you are interested in
submitting a new provider.

#### Core Bugfix/Enhancement

We are always happy when any developer is interested in diving into Terraform's
core to help out! Here's what we look for in smaller Core PRs.

 - [ ] __Unit tests__: Terraform's core is covered by hundreds of unit tests at
   several different layers of abstraction. Generally the best place to start
   is with a "Context Test". These are higher level test that interact
   end-to-end with most of Terraform's core. They are divided into test files
   for each major action (plan, apply, etc.). Getting a failing test is a great
   way to prove out a bug report or a new enhancement. With a context test in
   place, you can work on implementation and lower level unit tests. Lower
   level tests are largely context dependent, but the Context Tests are almost
   always part of core work.
 - [ ] __Documentation updates__: If the core change involves anything that
   needs to be reflected in our documentation, you can make those changes in
   the same PR. The [Terraform website][website] source is in this repo and
   includes instructions for getting a local copy of the site up and running if
   you'd like to preview your changes.
 - [ ] __Well-formed Code__: Do your best to follow existing conventions you
   see in the codebase, and ensure your code is formatted with `go fmt`. (The
   Travis CI build will fail if `go fmt` has not been run on incoming code.)
   The PR reviewers can help out on this front, and may provide comments with
   suggestions on how to improve the code.

#### Core Feature

If you're interested in taking on a larger core feature, it's a good idea to
get feedback early and often on the effort.

 - [ ] __Early validation of idea and implementation plan__: Terraform's core
   is complicated enough that there are often several ways to implement
   something, each of which has different implications and tradeoffs. Working
   through a plan of attack with the team before you dive into implementation
   will help ensure that you're working in the right direction. Opening a GitHub
   issue, or commenting on an existing issue, is a great way to get these
   conversations started.
 - [ ] __Unit tests__: Terraform's core is covered by hundreds of unit tests at
   several different layers of abstraction. Generally the best place to start
   is with a "Context Test". These are higher level test that interact
   end-to-end with most of Terraform's core. They are divided into test files
   for each major action (plan, apply, etc.). Getting a failing test is a great
   way to prove out a bug report or a new enhancement. With a context test in
   place, you can work on implementation and lower level unit tests. Lower
   level tests are largely context dependent, but the Context Tests are almost
   always part of core work.
 - [ ] __Documentation updates__: If the core change involves anything that
   needs to be reflected in our documentation, you can make those changes in
   the same PR. The [Terraform website][website] source is in this repo and
   includes instructions for getting a local copy of the site up and running if
   you'd like to preview your changes.
 - [ ] __Well-formed Code__: Do your best to follow existing conventions you
   see in the codebase, and ensure your code is formatted with `go fmt`. (The
   Travis CI build will fail if `go fmt` has not been run on incoming code.)
   The PR reviewers can help out on this front, and may provide comments with
   suggestions on how to improve the code.

### Writing Acceptance Tests

#### Acceptance Tests Often Cost Money to Run

Because acceptance tests create real resources, they often cost money to run.
Because the resources only exist for a short period of time, the total amount
of money required is usually a relatively small. Nevertheless, we don't want
financial limitations to be a barrier to contribution, so if you are unable to
pay to run acceptance tests for your contribution, simply mention this in your
pull request. We will happily accept "best effort" implementations of
acceptance tests and run them for you on our side. This might mean that your PR
takes a bit longer to merge, but it most definitely is not a blocker for
contributions.

#### Running an Acceptance Test

Acceptance tests can be run using the `testacc` target in the Terraform
`Makefile`. The individual tests to run can be controlled using a regular
expression. Prior to running the tests provider configuration details such as
access keys must be made available as environment variables.


[website]: https://github.com/hashicorp/terraform/tree/master/website
[acctests]: https://github.com/hashicorp/terraform#acceptance-tests
[community forum]: https://discuss.hashicorp.com/c/terraform-core
[ml]: https://groups.google.com/group/terraform-tool
