# Maintainer's Etiquette

Are you a core maintainer of Terraform? Great! Here's a few notes
to help you get comfortable when working on the project.

## Expectations

We value the time you spend on the project and as such your maintainer status
doesn't imply any obligations to do any specific work.

### Your PRs

These apply to all contributors, but maintainers should lead by examples! :wink:

 - for `provider/*` PRs it's useful to attach test results & advise on how to run the relevant tests
 - for `bug`fixes it's useful to attach repro case, ideally in a form of a test

### PRs/issues from others

 - you're welcomed to triage (attach labels to) other PRs and issues
   - we generally use 2-label system (= at least 2 labels per issue/PR) where one label is generic and other one API-specific, e.g. `enhancement` & `provider/aws`

## Merging

 - you're free to review PRs from the community or other HC employees and give :+1: / :-1:
 - if the PR submitter has push privileges (recognizable via `Collaborator`, `Member` or `Owner` badge) - we expect **the submitter** to merge their own PR after receiving a positive review from either HC employee or another maintainer. _Exceptions apply - see below._
 - we prefer to use the Github's interface or API to do this, just click the green button
 - squash?
   - squash when you think the commit history is irrelevant (will not be helpful for any readers in T+6mons)
 - Add the new PR to the **Changelog** if it may affect the user (almost any PR except test changes and docs updates)
   - we prefer to use the Github's web interface to modify the Changelog and use `[GH-12345]` to format the PR number. These will be turned into links as part of the release process. Breaking changes should be always documented separately.

## Release process

 - HC employees are responsible for cutting new releases
 - The employee cutting the release will always notify all maintainers via Slack channel before & after each release
	so you can avoid merging PRs during the release process.

## Exceptions

Any PR that is significantly changing or even breaking user experience cross-providers should always get at least one :+1: from a HC employee prior to merge.

It is generally advisable to leave PRs labelled as `core` for HC employees to review and merge.

Examples include:
 - adding/changing/removing a CLI (sub)command or a [flag](https://github.com/hashicorp/terraform/pull/12939)
 - introduce a new feature like [Environments](https://github.com/hashicorp/terraform/pull/12182) or [Shadow Graph](https://github.com/hashicorp/terraform/pull/9334)
 - changing config (HCL) like [adding support for lists](https://github.com/hashicorp/terraform/pull/6322)
 - change of the [build process or test environment](https://github.com/hashicorp/terraform/pull/9355)

## Breaking Changes

 - we always try to avoid breaking changes where possible and/or defer them to the nearest major release
   - [state migration](https://github.com/hashicorp/terraform/blob/2fe5976aec290f4b53f07534f4cde13f6d877a3f/helper/schema/resource.go#L33-L56) may help you avoid breaking changes, see [example](https://github.com/hashicorp/terraform/blob/351c6bed79abbb40e461d3f7d49fe4cf20bced41/builtin/providers/aws/resource_aws_route53_record_migrate.go)
   - either way BCs should be clearly documented in special section of the Changelog
 - Any BC must always receive at least one :+1: from HC employee prior to merge, two :+1:s are advisable

 ### Examples of Breaking Changes

  - https://github.com/hashicorp/terraform/pull/12396
  - https://github.com/hashicorp/terraform/pull/13872
  - https://github.com/hashicorp/terraform/pull/13752

## Unsure?

If you're unsure about anything, ask in the committer's Slack channel.

## New Providers

These will require :+1: and some extra effort from HC employee.

We expect all acceptance tests to be as self-sustainable as possible
to keep the bar for running any acceptance test low for anyone
outside of HashiCorp or core maintainers team.

We expect any test to run **in parallel** alongside any other test (even the same test).
To ensure this is possible, we need all tests to avoid sharing namespaces or using static unique names.
In rare occasions this may require the use of mutexes in the resource code.

### New Remote-API-based provider (e.g. AWS, Google Cloud, PagerDuty, Atlas)

We will need some details about who to contact or where to register for a new account
and generally we can't merge providers before ensuring we have a way to test them nightly,
which usually involves setting up a new account and obtaining API credentials.

### Local provider (e.g. MySQL, PostgreSQL, Kubernetes, Vault)

We will need either Terraform configs that will set up the underlying test infrastructure
(e.g. GKE cluster for Kubernetes) or Dockerfile(s) that will prepare test environment (e.g. MySQL)
and expose the endpoint for testing.

