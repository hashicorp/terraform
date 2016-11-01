---
layout: "google"
page_title: "Google: google_project"
sidebar_current: "docs-google-project"
description: |-
 Allows management of a Google Cloud Platform project. 
---

# google\_project

Allows management of an existing Google Cloud Platform project, and is
currently limited to adding or modifying the IAM Policy for the project.

When adding a policy to a project, the policy will be merged with the
project's existing policy. The policy is always specified in a
`google_iam_policy` data source and referencd from the project's
`policy_data` attribute.

## Example Usage

```js
resource "google_project" "my-project" {
    id = "your-project-id"
    policy_data = "${data.google_iam_policy.admin.policy}"
}

data "google_iam_policy" "admin" {
  binding {
    role = "roles/storage.objectViewer"
    members = [
      "user:evandbrown@gmail.com",
    ]
  }
}
```

## Argument Reference

The following arguments are supported:

* `id` - (Required) The project ID.
    Changing this forces a new project to be referenced.

* `policy` - (Optional) The `google_iam_policy` data source that represents
    the IAM policy that will be applied to the project. The policy will be
    merged with any existing policy applied to the project.

    Changing this updates the policy.

    Deleting this removes the policy, but leaves the original project policy
    intact. If there are overlapping `binding` entries between the original
    project policy and the data source policy, they will be removed.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `name` - The name of the project.

* `number` - The numeric identifier of the project.
