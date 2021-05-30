---
layout: "language"
page_title: "Module Testing Experiment - Configuration Language"
---

# Module Testing Experiment

This page is about some experimental features available in recent versions of
Terraform CLI related to integration testing of shared modules.

The Terraform team is aiming to use these features to gather feedback as part
of ongoing research into different strategies for testing Terraform modules.
These features are likely to change significantly in future releases based on
feedback.

## Current Research Goals

Our initial area of research is into the question of whether it's helpful and
productive to write module integration tests in the Terraform language itself,
or whether it's better to handle that as a separate concern orchestrated by
code written in other languages.

Some existing efforts have piloted both approaches:

* [Terratest](https://terratest.gruntwork.io/) and
  [kitchen-terraform](https://github.com/newcontext-oss/kitchen-terraform)
  both pioneered the idea of writing tests for Terraform modules with explicit
  orchestration written in the Go and Ruby programming languages, respectively.

* The Terraform provider
  [`apparentlymart/testing`](https://registry.terraform.io/providers/apparentlymart/testing/latest)
  introduced the idea of writing Terraform module tests in the Terraform
  language itself, using a special provider that can evaluate assertions
  and fail `terraform apply` if they don't pass.

Both of these approaches have both advantages and disadvantages, and so it's
likely that both will coexist for different situations, but the community
efforts have already explored the external-language testing model quite deeply
while the Terraform-integrated testing model has not yet been widely trialled.
For that reason, the current iteration of the module testing experiment is
aimed at trying to make the Terraform-integrated approach more accessible so
that more module authors can hopefully try it and share their experiences.

## Current Experimental Features

-> This page describes the incarnation of the experimental features introduced
in **Terraform CLI v0.15.0**. If you are using an earlier version of Terraform
then you'll need to upgrade to v0.15.0 or later to use the experimental features
described here, though you only need to use v0.15.0 or later for running tests;
your module itself can remain compatible with earlier Terraform versions, if
needed.

Our current area of interest is in what sorts of tests can and cannot be
written using features integrated into the Terraform language itself. As a
means to investigate that without invasive, cross-cutting changes to Terraform
Core we're using a special built-in Terraform provider as a placeholder for
potential new features.

If this experiment is successful then we expect to run a second round of
research and design about exactly what syntax is most ergonomic for writing
tests, but for the moment we're interested less in the specific syntax and more
in the capabilities of this approach.

The temporary extensions to Terraform for this experiment consist of the
following parts:

* A temporary experimental provider `terraform.io/builtin/test`, which acts as
  a placeholder for potential new language features related to test assertions.

* A `terraform test` command for more conveniently running multiple tests in
  a single action.

* An experimental convention of placing test configurations in subdirectories
  of a `tests` directory within your module, which `terraform test` will then
  discover and run.

We would like to invite adventurous module authors to try writing integration
tests for their modules using these mechanisms, and ideally also share the
tests you write (in a temporary VCS branch, if necessary) so we can see what
you were able to test, along with anything you felt unable to test in this way.

If you're interested in giving this a try, see the following sections for
usage details. Because these features are temporary experimental extensions,
there's some boilerplate required to activate and make use of it which would
likely not be required in a final design.

### Writing Tests for a Module

For the purposes of the current experiment, module tests are arranged into
_test suites_, each of which is a root Terraform module which includes a
`module` block calling the module under test, and ideally also a number of
test assertions to verify that the module outputs match expectations.

In the same directory where you keep your module's `.tf` and/or `.tf.json`
source files, create a subdirectory called `tests`. Under that directory,
make another directory which will serve as your first test suite, with a
directory name that concisely describes what the suite is aiming to test.

Here's an example directory structure of a typical module directory layout
with the addition of a test suite called `defaults`:

```
main.tf
outputs.tf
providers.tf
variables.tf
versions.tf
tests/
  defaults/
    test_defaults.tf
```

The `tests/defaults/test_defaults.tf` file will contain a call to the
main module with a suitable set of arguments and hopefully also one or more
resources that will, for the sake of the experiment, serve as the temporary
syntax for defining test assertions. For example:

```hcl
terraform {
  required_providers {
    # Because we're currently using a built-in provider as
    # a substitute for dedicated Terraform language syntax
    # for now, test suite modules must always declare a
    # dependency on this provider. This provider is only
    # available when running tests, so you shouldn't use it
    # in non-test modules.
    test = {
      source = "terraform.io/builtin/test"
    }

    # This example also uses the "http" data source to
    # verify the behavior of the hypothetical running
    # service, so we should declare that too.
    http = {
      source = "hashicorp/http"
    }
  }
}

module "main" {
  # source is always ../.. for test suite configurations,
  # because they are placed two subdirectories deep under
  # the main module directory.
  source = "../.."

  # This test suite is aiming to test the "defaults" for
  # this module, so it doesn't set any input variables
  # and just lets their default values be selected instead.
}

# As with all Terraform modules, we can use local values
# to do any necessary post-processing of the results from
# the module in preparation for writing test assertions.
locals {
  # This expression also serves as an implicit assertion
  # that the base URL uses URL syntax; the test suite
  # will fail if this function fails.
  api_url_parts = regex(
    "^(?:(?P<scheme>[^:/?#]+):)?(?://(?P<authority>[^/?#]*))?",
    module.main.api_url,
  )
}

# The special test_assertions resource type, which belongs
# to the test provider we required above, is a temporary
# syntax for writing out explicit test assertions.
resource "test_assertions" "api_url" {
  # "component" serves as a unique identifier for this
  # particular set of assertions in the test results.
  component = "api_url"

  # equal and check blocks serve as the test assertions.
  # the labels on these blocks are unique identifiers for
  # the assertions, to allow more easily tracking changes
  # in success between runs.

  equal "scheme" {
    description = "default scheme is https"
    got         = local.api_url_parts.scheme
    want        = "https"
  }

  check "port_number" {
    description = "default port number is 8080"
    condition   = can(regex(":8080$", local.api_url_parts.authority))
  }
}

# We can also use data resources to respond to the
# behavior of the real remote system, rather than
# just to values within the Terraform configuration.
data "http" "api_response" {
  depends_on = [
    # make sure the syntax assertions run first, so
    # we'll be sure to see if it was URL syntax errors
    # that let to this data resource also failing.
    test_assertions.api_url,
  ]

  url = module.main.api_url
}

resource "test_assertions" "api_response" {
  component = "api_response"

  check "valid_json" {
    description = "base URL responds with valid JSON"
    condition   = can(jsondecode(data.http.api_response.body))
  }
}
```

If you like, you can create additional directories alongside
the `default` directory to define additional test suites that
pass different variable values into the main module, and
then include assertions that verify that the result has changed
in the expected way.

### Running Your Tests

The `terraform test` command aims to make it easier to exercise all of your
defined test suites at once, and see only the output related to any test
failures or errors.

The current experimental incarnation of this command expects to be run from
your main module directory. In our example directory structure above,
that was the directory containing `main.tf` etc, and _not_ the specific test
suite directory containing `test_defaults.tf`.

Because these test suites are integration tests rather than unit tests, you'll
need to set up any credentials files or environment variables needed by the
providers your module uses before running `terraform test`. The test command
will, for each suite:

* Install the providers and any external modules the test configuration depends
  on.
* Create an execution plan to create the objects declared in the module.
* Apply that execution plan to create the objects in the real remote system.
* Collect all of the test results from the apply step, which would also have
  "created" the `test_assertions` resources.
* Destroy all of the objects recorded in the temporary test state, as if running
  `terraform destroy` against the test configuration.

```shellsession
$ terraform test
─── Failed: defaults.api_url.scheme (default scheme is https) ───────────────
wrong value
    got:  "http"
    want: "https"
─────────────────────────────────────────────────────────────────────────────
```

In this case, it seems like the module returned an `http` rather than an
`https` URL in the default case, and so the `defaults.api_url.scheme`
assertion failed, and the `terraform test` command detected and reported it.

The `test_assertions` resource captures any assertion failures but does not
return an error, because that can then potentially allow downstream
assertions to also run and thus capture as much context as possible.
However, if Terraform encounters any _errors_ while processing the test
configuration it will halt processing, which may cause some of the test
assertions to be skipped.

## Known Limitations

The design above is very much a prototype aimed at gathering more experience
with the possibilities of testing inside the Terraform language. We know it's
currently somewhat non-ergonomic, and hope to improve on that in later phases
of research and design, but the main focus of this iteration is on available
functionality and so with that in mind there are some specific possibilities
that we know the current prototype doesn't support well:

* Testing of subsequent updates to an existing deployment of a module.
  Currently tests written in this way can only exercise the create and destroy
  behaviors.

* Assertions about expected errors. For a module that includes variable
  validation rules and data resources that function as assertion checks,
  the current prototype doesn't have any way to express that a particular
  set of inputs is _expected_ to produce an error, and thus report a test
  failure if it doesn't. We'll hopefully be able to improve on this in a future
  iteration with the test assertions better integrated into the language.

* Capturing context about failures. Due to this prototype using a provider as
  an approximation for new assertion syntax, the `terraform test` command is
  limited in how much context it's able to gather about failures. A design
  more integrated into the language could potentially capture the source
  expressions and input values to give better feedback about what went wrong,
  similar to what Terraform typically returns from expression evaluation errors
  in the main language.

* Unit testing without creating real objects. Although we do hope to spend more
  time researching possibilities for unit testing against fake test doubles in
  the future, we've decided to focus on integration testing to start because
  it feels like the better-defined problem.

## Sending Feedback

The sort of feedback we'd most like to see at this stage of the experiment is
to see the source code of any tests you've written against real modules using
the features described above, along with notes about anything that you
attempted to test but were blocked from doing so by limitations of the above
features. The most ideal way to share that would be to share a link to a
version control branch where you've added such tests, if your module is open
source.

If you've previously written or attempted to write tests in an external
language, using a system like Terratest or kitchen-terraform, we'd also be
interested to hear about comparative differences between the two: what worked
well in each and what didn't work so well.

Our ultimate goal is to work towards an integration testing methodology which
strikes the best compromise between the capabilities of these different
approaches, ideally avoiding a hard requirement on any particular external
language and fitting well into the Terraform workflow.

Since this is still early work and likely to lead to unstructured discussion,
we'd like to gather feedback primarily via new topics in
[the community forum](https://discuss.hashicorp.com/c/terraform-core/27). That
way we can have some more freedom to explore different ideas and approaches
without the structural requirements we typically impose on GitHub issues.

Any feedback you'd like to share would be very welcome!
