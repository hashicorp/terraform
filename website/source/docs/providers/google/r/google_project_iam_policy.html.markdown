---
layout: "google"
page_title: "Google: google_project_iam_policy"
sidebar_current: "docs-google-project-iam-policy"
description: |-
 Allows management of an IAM policy for a Google Cloud Platform project.
---

# google\_project\_iam\_policy

Allows creation and management of an IAM policy for an existing Google Cloud
Platform project.

~> **Be careful!** You can accidentally lock yourself out of your project
   using this resource. Proceed with caution.

## Example Usage

```hcl
resource "google_project_iam_policy" "project" {
  project     = "your-project-id"
  policy_data = "${data.google_iam_policy.admin.policy_data}"
}

data "google_iam_policy" "admin" {
  binding {
    role = "roles/editor"

    members = [
      "user:jane@example.com",
    ]
  }
}
```

## Argument Reference

The following arguments are supported:

* `project` - (Required) The project ID.
    Changing this forces a new project to be created.

* `policy_data` - (Required) The `google_iam_policy` data source that represents
    the IAM policy that will be applied to the project. The policy will be
    merged with any existing policy applied to the project.

    Changing this updates the policy.

    Deleting this removes the policy, but leaves the original project policy
    intact. If there are overlapping `binding` entries between the original
    project policy and the data source policy, they will be removed.

* `authoritative` - (Optional) A boolean value indicating if this policy
    should overwrite any existing IAM policy on the project. When set to true,
    **any policies not in your config file will be removed**. This can **lock
    you out** of your project until an Organization Administrator grants you
    access again, so please exercise caution. If this argument is `true` and you
    want to delete the resource, you must set the `disable_project` argument to
    `true`, acknowledging that the project will be inaccessible to anyone but the
    Organization Admins, as it will no longer have an IAM policy.

* `disable_project` - (Optional) A boolean value that must be set to `true`
    if you want to delete a `google_project_iam_policy` that is authoritative.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `etag` - (Computed) The etag of the project's IAM policy.

* `restore_policy` - (Computed) The IAM policy that will be restored when a
    non-authoritative policy resource is deleted.
