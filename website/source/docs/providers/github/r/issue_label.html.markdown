---
layout: "github"
page_title: "GitHub: github_issue_label"
sidebar_current: "docs-github-resource-issue-label"
description: |-
  Provides a GitHub issue label resource.
---

# github\_issue_label

Provides a GitHub issue label resource.

This resource allows you to create and manage issue labels within your
Github organization.

## Example Usage

```
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
* `color` - (Required) A 6 character hex code, without the leading #, identifying the color of the label.
