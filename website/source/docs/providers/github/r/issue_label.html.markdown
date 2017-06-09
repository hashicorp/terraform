---
layout: "github"
page_title: "GitHub: github_issue_label"
sidebar_current: "docs-github-resource-issue-label"
description: |-
  Provides a GitHub issue label resource.
---

# github_issue_label

Provides a GitHub issue label resource.

This resource allows you to create and manage issue labels within your
Github organization.

Issue labels are keyed off of their "name", so pre-existing issue labels result
in a 422 HTTP error if they exist outside of Terraform. Normally this would not
be an issue, except new repositories are created with a "default" set of labels,
and those labels easily conflict with custom ones.

This resource will first check if the label exists, and then issue an update,
otherwise it will create.

## Example Usage

```hcl
# Create a new, red colored label
resource "github_issue_label" "test_repo" {
  repository = "test-repo"
  name       = "Urgent"
  color      = "FF0000"
}
```

## Argument Reference

The following arguments are supported:

* `repository` - (Required) The GitHub repository

* `name` - (Required) The name of the label.

* `color` - (Required) A 6 character hex code, **without the leading #**, identifying the color of the label.

## Import

Github Issue Labels can be imported using an id made up of `repository:name`, e.g.

```
$ terraform import github_issue_label.panic_label terraform:panic
```