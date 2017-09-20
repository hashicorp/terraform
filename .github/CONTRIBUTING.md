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

## HashiCorp vs. Community Providers

We separate providers out into what we call "HashiCorp Providers" and
"Community Providers".

HashiCorp providers are providers that we'll dedicate full time resources to
improving, supporting the latest features, and fixing bugs. These are providers
we understand deeply and are confident we have the resources to manage
ourselves.

Community providers are providers where we depend on the community to
contribute fixes and enhancements to improve. HashiCorp will run automated
tests and ensure these providers continue to work, but will not dedicate full
time resources to add new features to these providers. These providers are
available in official Terraform releases, but the functionality is primarily
contributed.

The current list of HashiCorp Providers is as follows:

 * `aws`
 * `azurerm`
 * `google`
 * `opc`

Our testing standards are the same for both HashiCorp and Community providers,
and HashiCorp runs full acceptance test suites for every provider nightly to
ensure Terraform remains stable.

We make the distinction between these two types of providers to help
highlight the vast amounts of community effort that goes in to making Terraform
great, and to help contributors better understand the role HashiCorp employees
play in the various areas of the code base.

## Issues

### Issue Reporting Checklists

We welcome issues of all kinds including feature requests, bug reports, and
general questions. Below you'll find checklists with guidelines for well-formed
issues of each type.

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

 - [ ] __Search for answers in Terraform documentation__: We're happy to answer
   questions in GitHub Issues, but it helps reduce issue churn and maintainer
   workload if you work to find answers to common questions in the
   documentation. Often times Question issues result in documentation updates
   to help future users, so if you don't find an answer, you can give us
   pointers for where you'd expect to see it in the docs.

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

 * For pull requests that follow the guidelines, we expect to be able to review
   and merge very quickly.
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

 - [ ] __Minimal LOC__: It can be inefficient for both the reviewer
   and author to go through long feedback cycles on a big PR with many
   resources. We therefore encourage you to only submit **1 resource at a time**.
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
a whole new API. It's a larger undertaking, but brings major new functionality
into Terraform.

 - [ ] __Minimal initial LOC__: Some providers may be big and it can be
   inefficient for both reviewer & author to go through long feedback cycles
   on a big PR with many resources. We encourage you to only submit
   the necessary minimum in a single PR, ideally **just the first resource**
   of the provider.
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

For example, to run an acceptance test against the Azure Resource Manager
provider, the following environment variables must be set:

```sh
export ARM_SUBSCRIPTION_ID=...
export ARM_CLIENT_ID=...
export ARM_CLIENT_SECRET=...
export ARM_TENANT_ID=...
```

Tests can then be run by specifying the target provider and a regular
expression defining the tests to run:

```sh
$ make testacc TEST=./builtin/providers/azurerm TESTARGS='-run=TestAccAzureRMPublicIpStatic_update'
==> Checking that code complies with gofmt requirements...
go generate ./...
TF_ACC=1 go test ./builtin/providers/azurerm -v -run=TestAccAzureRMPublicIpStatic_update -timeout 120m
=== RUN   TestAccAzureRMPublicIpStatic_update
--- PASS: TestAccAzureRMPublicIpStatic_update (177.48s)
PASS
ok      github.com/hashicorp/terraform/builtin/providers/azurerm    177.504s
```

Entire resource test suites can be targeted by using the naming convention to
write the regular expression. For example, to run all tests of the
`azurerm_public_ip` resource rather than just the update test, you can start
testing like this:

```sh
$ make testacc TEST=./builtin/providers/azurerm TESTARGS='-run=TestAccAzureRMPublicIpStatic'
==> Checking that code complies with gofmt requirements...
go generate ./...
TF_ACC=1 go test ./builtin/providers/azurerm -v -run=TestAccAzureRMPublicIpStatic -timeout 120m
=== RUN   TestAccAzureRMPublicIpStatic_basic
--- PASS: TestAccAzureRMPublicIpStatic_basic (137.74s)
=== RUN   TestAccAzureRMPublicIpStatic_update
--- PASS: TestAccAzureRMPublicIpStatic_update (180.63s)
PASS
ok      github.com/hashicorp/terraform/builtin/providers/azurerm    318.392s
```

#### Writing an Acceptance Test

Terraform has a framework for writing acceptance tests which minimises the
amount of boilerplate code necessary to use common testing patterns. The entry
point to the framework is the `resource.Test()` function.

Tests are divided into `TestStep`s. Each `TestStep` proceeds by applying some
Terraform configuration using the provider under test, and then verifying that
results are as expected by making assertions using the provider API. It is
common for a single test function to exercise both the creation of and updates
to a single resource. Most tests follow a similar structure.

1. Pre-flight checks are made to ensure that sufficient provider configuration
   is available to be able to proceed - for example in an acceptance test
   targeting AWS, `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` must be set prior
   to running acceptance tests. This is common to all tests exercising a single
   provider.

Each `TestStep` is defined in the call to `resource.Test()`. Most assertion
functions are defined out of band with the tests. This keeps the tests
readable, and allows reuse of assertion functions across different tests of the
same type of resource. The definition of a complete test looks like this:

```go
func TestAccAzureRMPublicIpStatic_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMPublicIpDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureRMVPublicIpStatic_basic,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMPublicIpExists("azurerm_public_ip.test"),
				),
			},
        },
    })
}
```

When executing the test, the following steps are taken for each `TestStep`:

1. The Terraform configuration required for the test is applied. This is
   responsible for configuring the resource under test, and any dependencies it
   may have. For example, to test the `azurerm_public_ip` resource, an
   `azurerm_resource_group` is required. This results in configuration which
   looks like this:

    ```hcl
    resource "azurerm_resource_group" "test" {
        name = "acceptanceTestResourceGroup1"
        location = "West US"
    }

    resource "azurerm_public_ip" "test" {
        name = "acceptanceTestPublicIp1"
        location = "West US"
        resource_group_name = "${azurerm_resource_group.test.name}"
        public_ip_address_allocation = "static"
    }
    ```

1. Assertions are run using the provider API. These use the provider API
   directly rather than asserting against the resource state. For example, to
   verify that the `azurerm_public_ip` described above was created
   successfully, a test function like this is used:

    ```go
    func testCheckAzureRMPublicIpExists(name string) resource.TestCheckFunc {
        return func(s *terraform.State) error {
            // Ensure we have enough information in state to look up in API
            rs, ok := s.RootModule().Resources[name]
            if !ok {
                return fmt.Errorf("Not found: %s", name)
            }

            publicIPName := rs.Primary.Attributes["name"]
            resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
            if !hasResourceGroup {
                return fmt.Errorf("Bad: no resource group found in state for public ip: %s", availSetName)
            }

            conn := testAccProvider.Meta().(*ArmClient).publicIPClient

            resp, err := conn.Get(resourceGroup, publicIPName, "")
            if err != nil {
                return fmt.Errorf("Bad: Get on publicIPClient: %s", err)
            }

            if resp.StatusCode == http.StatusNotFound {
                return fmt.Errorf("Bad: Public IP %q (resource group: %q) does not exist", name, resourceGroup)
            }

            return nil
        }
    }
    ```

   Notice that the only information used from the Terraform state is the ID of
   the resource - though in this case it is necessary to split the ID into
   constituent parts in order to use the provider API. For computed properties,
   we instead assert that the value saved in the Terraform state was the
   expected value if possible. The testing framework provides helper functions
   for several common types of check - for example:

    ```go
    resource.TestCheckResourceAttr("azurerm_public_ip.test", "domain_name_label", "mylabel01"),
    ```

1. The resources created by the test are destroyed. This step happens
   automatically, and is the equivalent of calling `terraform destroy`.

1. Assertions are made against the provider API to verify that the resources
   have indeed been removed. If these checks fail, the test fails and reports
   "dangling resources". The code to ensure that the `azurerm_public_ip` shown
   above looks like this:

    ```go
    func testCheckAzureRMPublicIpDestroy(s *terraform.State) error {
        conn := testAccProvider.Meta().(*ArmClient).publicIPClient

        for _, rs := range s.RootModule().Resources {
            if rs.Type != "azurerm_public_ip" {
                continue
            }

            name := rs.Primary.Attributes["name"]
            resourceGroup := rs.Primary.Attributes["resource_group_name"]

            resp, err := conn.Get(resourceGroup, name, "")

            if err != nil {
                return nil
            }

            if resp.StatusCode != http.StatusNotFound {
                return fmt.Errorf("Public IP still exists:\n%#v", resp.Properties)
            }
        }

        return nil
    }
    ```

   These functions usually test only for the resource directly under test: we
   skip the check that the `azurerm_resource_group` has been destroyed when
   testing `azurerm_resource_group`, under the assumption that
   `azurerm_resource_group` is tested independently in its own acceptance
   tests.

[website]: https://github.com/hashicorp/terraform/tree/master/website
[acctests]: https://github.com/hashicorp/terraform#acceptance-tests
[ml]: https://groups.google.com/group/terraform-tool
