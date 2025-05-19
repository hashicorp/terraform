# Equivalence testing

This directory contains the test cases for the equivalence testing. The
Terraform equivalence tests are E2E tests that are used to verify that the
output of Terraform commands doesn't change in unexpected ways. The tests are
run by comparing the output of the Terraform commands before and after a change
to the codebase.

## Running the tests

The equivalence tests are executed by the Terraform equivalence testing 
framework. This is built in [github.com/hashicorp/terraform-equivalence-testing](https://github.com/hashicorp/terraform-equivalence-testing).

To execute the tests you must download the `terraform-equivalence-testing` 
binary and execute either the `diff` or `update` command. The `diff` command
will run the tests and output the differences between the current and previous
run. The `update` command will run the tests and update the reference output
files.

You can also execute the tests directly using the `equivalence-tests-manual`
GitHub action. This action will run the tests against a given branch and
open a PR with the results.

## Automated testing

The equivalence tests are run automatically by the Terraform CI system. The
tests are run when every pull request is opened and when every pull request
is closed.

When pull requests are opened, the tests run the diff command and will comment
on the PR with the results. PR authors should validate any changes to the output
files and make sure that the changes are expected.

When pull requests are closed, the tests run the update command and open a new
PR with the updated reference output files. PR authors should review the changes
and make sure that the changes are expected before merging the automated PR.

If the framework detects no changes, the process should be invisible to the PR
author. No comments will be made on the PR and no new PRs will be opened.

## Writing new tests

New tests should be written into the `tests` directory. Each test should be
written in a separate directory and should follow the guidelines in the
equivalence testing framework documentation. Any tests added to this directory
will be picked up the CI system and run automatically.
