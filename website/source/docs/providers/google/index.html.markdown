---
layout: "google"
page_title: "Provider: Google Cloud"
sidebar_current: "docs-google-index"
description: |-
  The Google Cloud provider is used to interact with Google Cloud services. The provider needs to be configured with the proper credentials before it can be used.
---

# Google Cloud Provider

The Google Cloud provider is used to interact with
[Google Cloud services](https://cloud.google.com/). The provider needs
to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
// Configure the Google Cloud provider
provider "google" {
  credentials = "${file("account.json")}"
  project     = "my-gce-project"
  region      = "us-central1"
}

// Create a new instance
resource "google_compute_instance" "default" {
  # ...
}
```

## Configuration Reference

The following keys can be used to configure the provider.

* `credentials` - (Optional) Contents of the JSON file used to describe your
  account credentials, downloaded from Google Cloud Console. More details on
  retrieving this file are below. Credentials may be blank if you are running
  Terraform from a GCE instance with a properly-configured [Compute Engine
  Service Account](https://cloud.google.com/compute/docs/authentication). This
  can also be specified using any of the following environment variables
  (listed in order of precedence):

    * `GOOGLE_CREDENTIALS`
    * `GOOGLE_CLOUD_KEYFILE_JSON`
    * `GCLOUD_KEYFILE_JSON`

    The [`GOOGLE_APPLICATION_CREDENTIALS`](https://developers.google.com/identity/protocols/application-default-credentials#howtheywork)
    environment variable can also contain the path of a file to obtain credentials
    from.

* `project` - (Required) The ID of the project to apply any resources to.  This
  can be specified using any of the following environment variables (listed in
  order of precedence):

    * `GOOGLE_PROJECT`
    * `GCLOUD_PROJECT`
    * `CLOUDSDK_CORE_PROJECT`

* `region` - (Required) The region to operate under. This can also be specified
  using any of the following environment variables (listed in order of
  precedence):

    * `GOOGLE_REGION`
    * `GCLOUD_REGION`
    * `CLOUDSDK_COMPUTE_REGION`

## Authentication JSON File

Authenticating with Google Cloud services requires a JSON
file which we call the _account file_.

This file is downloaded directly from the
[Google Developers Console](https://console.developers.google.com). To make
the process more straightforwarded, it is documented here:

1. Log into the [Google Developers Console](https://console.developers.google.com)
   and select a project.

2. The API Manager view should be selected, click on "Credentials" on the left,
   then "Create credentials", and finally "Service account key".

3. Select "Compute Engine default service account" in the "Service account"
   dropdown, and select "JSON" as the key type.

4. Clicking "Create" will download your `credentials`.
