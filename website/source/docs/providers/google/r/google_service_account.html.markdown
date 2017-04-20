---
layout: "google"
page_title: "Google: google_service_account"
sidebar_current: "docs-google-service-account"
description: |-
 Allows management of a Google Cloud Platform service account.
---

# google\_service\_account

Allows management of a [Google Cloud Platform service account](https://cloud.google.com/compute/docs/access/service-accounts)

## Example Usage

This snippet creates a service account, then gives it objectViewer
permission in a project.

```hcl
resource "google_service_account" "object_viewer" {
  account_id   = "object-viewer"
  display_name = "Object viewer"
}

resource "google_project" "my_project" {
  id          = "your-project-id"
  policy_data = "${data.google_iam_policy.admin.policy_data}"
}

data "google_iam_policy" "admin" {
  binding {
    role = "roles/storage.objectViewer"

    members = [
      "serviceAccount:${google_service_account.object_viewer.email}",
    ]
  }
}
```

## Argument Reference

The following arguments are supported:

* `account_id` - (Required) The service account ID.
    Changing this forces a new service account to be created.

* `display_name` - (Optional) The display name for the service account.
    Can be updated without creating a new resource.

* `project` - (Optional) The project that the service account will be created in.
    Defaults to the provider project configuration.

* `policy_data` - (Optional) The `google_iam_policy` data source that represents
    the IAM policy that will be applied to the service account. The policy will be
    merged with any existing policy.

    Changing this updates the policy.

    Deleting this removes the policy declared in Terraform. Any policy bindings
    associated with the project before Terraform was used are not deleted.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `email` - The e-mail address of the service account. This value
    should be referenced from any `google_iam_policy` data sources
    that would grant the service account privileges.

* `name` - The fully-qualified name of the service account.

* `unique_id` - The unique id of the service account.
