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

## Issues

### Issue Reporting Checklists

We welcome issues of all kinds including feature requests, bug reports, and
general questions. Below you'll find checklists with guidlines for well-formed
issues of each type.

#### Bug Reports

 - [ ] __Test against latest release__: Make sure you test against the latest
   released version. It is possible we already fixed the bug you're experiencing.

 - [ ] __Search for possible duplicate issues__: It's helpful to keep issues
   consolidated to one thread, so do a quick search on existing issues to check
   if anybody else has reported the same thing.

 - [ ] __Include steps to reproduce__: Provide steps to reproduce the issue,
   along with your `.tf` files, with secrets removed, so we can try to
   reproduce it. Without this, it makes it much harder to fix the issue.

 - [ ] __For panics, include `crash.log`__: If you experienced a panic, please
   create a [gist](https://gist.github.com) of the *entire* generated crash log
   for us to look at. Double check no sensitive items were in the log.

#### Feature Requests

 - [ ] __Search for possible duplicate issues__: It's helpful to keep issues
   consolidated to one thread, so do a quick search on existing issues to check
   if anybody else has reported the same thing.

 - [ ] __Include a use case description__: In addition to describing the
   behavior of the feature you'd like to see added, it's helpful to also lay
   out the reason why the feature would be important and how it would benefit
   Terraform users.

#### Questions

 - [ ] __Search for answers in Terraform documentation__: We're happy to answer
   questions in GitHub Issues, but it helps reduce issue churn and maintainer
   workload if you work to find answers to common questions in the
   documentation. Often times Question issues result in documentation updates
   to help future users, so if you don't find an answer, you can give us
   pointers for where you'd expect to see it in the docs.

### Issue Lifecycle

1. The issue is reported.

2. The issue is verified and categorized by a Terraform collaborator.
   Categorization is done via tags. For example, bugs are marked as "bugs".

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

#### Enhancement/Bugfix to a Resource

Working on existing resources is a great way to get started as a Terraform
contributor because you can work within existing code and tests to get a feel
for what to do.

 - [ ] __Acceptance test coverage of new behavior__: Existing resources each
   have a set of [acceptance tests][acctests] covering their functionality.
   These tests should exercise all the behavior of the resource. Whether you are
   adding something or fixing a bug, the idea is to have an acceptance test that
   fails if your code were to be removed. Sometimes it is sufficient to
   "enhance" an existing test by adding an assertion or tweaking the config
   that is used, but often a new test is better to add. You can copy/paste an
   existing test and follow the conventions you see there, modifying the test
   to exercise the behavior of your code.
 - [ ] __Documentation updates__: If your code makes any changes that need to
   be documented, you should include those doc updates in the same PR. The
   [Terraform website][website] source is in this repo and includes
   instructions for getting a local copy of the site up and running if you'd
   like to preview your changes.
 - [ ] __Well-formed Code__: Do your best to follow existing conventions you
   see in the codebase, and ensure your code is formatted with `go fmt`. (The
   Travis CI build will fail if `go fmt` has not been run on incoming code.)
   The PR reviewers can help out on this front, and may provide comments with
   suggestions on how to improve the code.

#### New Resource

Implementing a new resource is a good way to learn more about how Terraform
interacts with upstream APIs. There are plenty of examples to draw from in the
existing resources, but you still get to implement something completely new.

 - [ ] __Acceptance tests__: New resources should include acceptance tests
   covering their behavior. See [Writing Acceptance
   Tests](#writing-acceptance-tests) below for a detailed guide on how to
   approach these.
 - [ ] __Documentation__: Each resource gets a page in the Terraform
   documentation. The [Terraform website][website] source is in this
   repo and includes instructions for getting a local copy of the site up and
   running if you'd like to preview your changes. For a resource, you'll want
   to add a new file in the appropriate place and add a link to the sidebar for
   that page.
 - [ ] __Well-formed Code__: Do your best to follow existing conventions you
   see in the codebase, and ensure your code is formatted with `go fmt`. (The
   Travis CI build will fail if `go fmt` has not been run on incoming code.)
   The PR reviewers can help out on this front, and may provide comments with
   suggestions on how to improve the code.

#### New Provider

Implementing a new provider gives Terraform the ability to manage resources in
a whole new API. It's a larger undertaking, but brings major new functionaliy
into Terraform.

 - [ ] __Acceptance tests__: Each provider should include an acceptance test
   suite with tests for each resource should include acceptance tests covering
   its behavior. See [Writing Acceptance Tests](#writing-acceptance-tests) below
   for a detailed guide on how to approach these.
 - [ ] __Documentation__: Each provider has a section in the Terraform
   documentation. The [Terraform website][website] source is in this repo and
   includes instructions for getting a local copy of the site up and running if
   you'd like to preview your changes. For a provider, you'll want to add new
   index file and individual pages for each resource.
 - [ ] __Well-formed Code__: Do your best to follow existing conventions you
   see in the codebase, and ensure your code is formatted with `go fmt`. (The
   Travis CI build will fail if `go fmt` has not been run on incoming code.)
   The PR reviewers can help out on this front, and may provide comments with
   suggestions on how to improve the code.

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
   will help ensure that you're working in the right direction.
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

Terraform includes an acceptance test harness that does most of the repetitive
work involved in testing a resource.

#### Anatomy of an Acceptance Test

An acceptance test exercises a resource by applying Terraform configuration and
following up with direct API calls to assert that everything worked properly.

```go
func TestAccMyProviderMyResource_basic(t *testing.T) {
}
```

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

[website]: https://github.com/hashicorp/terraform/tree/master/website
[acctests]: https://github.com/hashicorp/terraform#acceptance-tests
[ml]: https://groups.google.com/group/terraform-tool
