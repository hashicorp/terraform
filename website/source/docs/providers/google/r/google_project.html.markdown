---
layout: "google"
page_title: "Google: google_project"
sidebar_current: "docs-google-project"
description: |-
 Allows management of a Google Cloud Platform project. 
---

# google\_project

Allows creation and management of a Google Cloud Platform project and its
associated enabled services/APIs.

Projects created with this resource must be associated with an Organization.
See the [Organization documentation](https://cloud.google.com/resource-manager/docs/quickstart) for more details.

The service account used to run Terraform when creating a `google_project`
resource must have `roles/resourcemanager.projectCreator`. See the
[Access Control for Organizations Using IAM](https://cloud.google.com/resource-manager/docs/access-control-org)
doc for more information.

## Example Usage

```js
resource "google_project" "my_project" {
    project_id = "your-project-id"
    org_id = "1234567"
    services = ["compute_component", "storage-component-json.googleapis.com", "iam.googleapis.com"]
}
```

## Argument Reference

The following arguments are supported:

* `project_id` - (Optional) The project ID.
    Changing this forces a new project to be created. If this attribute is not
    set, `id` must be set. As `id` is deprecated, consider this attribute
    required. If you are using `project_id` and creating a new project, the
    `org_id` and `name` attributes are also required.

* `id` - (Deprecated) The project ID.
    This attribute has unexpected behaviour and probably does not work
    as users would expect; it has been deprecated, and will be removed in future
    versions of Terraform. The `project_id` attribute should be used instead. See
    [below](#id-field) for more information about its behaviour.

* `project_id` - (Required) The project ID.
    Changing this forces a new project to be created.

* `org_id` - (Optional) The numeric ID of the organization this project belongs to.
    This is required if you are creating a new project.
    Changing this forces a new project to be created.

* `name` - (Optional) The display name of the project.
    This is required if you are creating a new project.

* `services` - (Optional) The services/APIs that are enabled for this project.
    For a list of available services, run `gcloud beta service-management list`

* `skip_delete` - (Optional) If true, the Terraform resource can be deleted
    without deleting the Project via the Google API.

* `policy_data` - (Deprecated) The IAM policy associated with the project.
    This argument is no longer supported, and will be removed in a future version
    of Terraform. It should be replaced with a `google_project_iam_policy` resource.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `number` - The numeric identifier of the project.
* `policy_etag` - (Deprecated) The etag of the project's IAM policy, used to
    determine if the IAM policy has changed. Please use `google_project_iam_policy`'s
    `etag` property instead; future versions of Terraform will remove the `policy_etag`
    attribute

## ID Field

In previous versions of Terraform, `google_project` resources used an `id` field in
config files to specify the project ID. Unfortunately, due to limitations in Terraform,
this field always looked empty to Terraform. Terraform fell back on using the project
the Google Cloud provider is configured with. If you're using the `id` field in your
configurations, know that it is being ignored, and its value will always be seen as the
ID of the project being used to authenticate Terraform's requests. You should move to the
`project_id` field as soon as possible.
