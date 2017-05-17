---
layout: "google"
page_title: "Google: google_project_services"
sidebar_current: "docs-google-project-services"
description: |-
 Allows management of API services for a Google Cloud Platform project.
---

# google\_project\_services

Allows management of enabled API services for an existing Google Cloud
Platform project. Services in an existing project that are not defined
in the config will be removed.

For a list of services available, visit the
[API library page](https://console.cloud.google.com/apis/library) or run `gcloud service-management list`.

## Example Usage

```hcl
resource "google_project_services" "project" {
  project = "your-project-id"
  services   = ["iam.googleapis.com", "cloudresourcemanager.googleapis.com"]
}
```

## Argument Reference

The following arguments are supported:

* `project` - (Required) The project ID.
    Changing this forces a new project to be created.

* `services` - (Required) The list of services that are enabled. Supports
    update.
