---
layout: "google"
page_title: "Google: google_iam_policy"
sidebar_current: "docs-google-datasource-iam-policy"
description: |-
  Generates an IAM policy that can be referenced by other resources, applying
  the policy to them.
---

# google\_iam\_policy

Generates an IAM policy document that may be referenced by and applied to
other Google Cloud Platform resources, such as the `google_project` resource.

```
data "google_iam_policy" "admin" {
  binding {
    role = "roles/compute.instanceAdmin"

    members = [
      "serviceAccount:your-custom-sa@your-project.iam.gserviceaccount.com",
    ]
  }

  binding {
    role = "roles/storage.objectViewer"

    members = [
      "user:evanbrown@google.com",
    ]
  }
}
```

This data source is used to define IAM policies to apply to other resources.
Currently, defining a policy through a datasource and referencing that policy
from another resource is the only way to apply an IAM policy to a resource.

**Note:** Several restrictions apply when setting IAM policies through this API.
See the [setIamPolicy docs](https://cloud.google.com/resource-manager/reference/rest/v1/projects/setIamPolicy)
for a list of these restrictions.

## Argument Reference

The following arguments are supported:

* `binding` (Required) - A nested configuration block (described below)
  defining a binding to be included in the policy document. Multiple
  `binding` arguments are supported.

Each document configuration must have one or more `binding` blocks, which
each accept the following arguments:

* `role` (Required) - The role/permission that will be granted to the members.
  See the [IAM Roles](https://cloud.google.com/compute/docs/access/iam) documentation for a complete list of roles.
* `members` (Required) - An array of users/principals that will be granted
  the privilege in the `role`. For a human user, prefix the user's e-mail
  address with `user:` (e.g., `user:evandbrown@gmail.com`). For a service
  account, prefix the service account e-mail address with `serviceAccount:`
  (e.g., `serviceAccount:your-service-account@your-project.iam.gserviceaccount.com`).

## Attributes Reference

The following attribute is exported:

* `policy_data` - The above bindings serialized in a format suitable for
  referencing from a resource that supports IAM.
