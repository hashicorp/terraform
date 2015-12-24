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

```
# Configure the Google Cloud provider
provider "google" {
  credentials = "${file("account.json")}"
  project     = "my-gce-project"
  region      = "us-central1"
}

# Create a new instance
resource "google_compute_instance" "default" {
  ...
}
```

## Configuration Reference

The following keys can be used to configure the provider.

* `credentials` - (Optional) Contents of the JSON file used to describe your
  account credentials, downloaded from Google Cloud Console. More details on
  retrieving this file are below. Credentials may be blank if you are running
  Terraform from a GCE instance with a properly-configured [Compute Engine
  Service Account](https://cloud.google.com/compute/docs/authentication). This
  can also be specified with the `GOOGLE_CREDENTIALS` shell environment
  variable.

* `project` - (Required) The ID of the project to apply any resources to.  This
  can also be specified with the `GOOGLE_PROJECT` shell environment variable.

* `region` - (Required) The region to operate under. This can also be specified
  with the `GOOGLE_REGION` shell environment variable.

The following keys are supported for backwards compatibility, and may be
removed in a future version:

* `account_file` - __Deprecated: please use `credentials` instead.__
  Path to or contents of the JSON file used to describe your
  account credentials, downloaded from Google Cloud Console. More details on
  retrieving this file are below. The `account file` can be "" if you are running
  terraform from a GCE instance with a properly-configured [Compute Engine
  Service Account](https://cloud.google.com/compute/docs/authentication). This
  can also be specified with the `GOOGLE_ACCOUNT_FILE` shell environment
  variable.


## Authentication JSON File

Authenticating with Google Cloud services requires a JSON
file which we call the _account file_.

This file is downloaded directly from the
[Google Developers Console](https://console.developers.google.com). To make
the process more straightforwarded, it is documented here:

1. Log into the [Google Developers Console](https://console.developers.google.com)
   and select a project.

2. Click the menu button in the top left corner, and navigate to "Permissions",
   then "Service accounts", and finally "Create service account".

3. Provide a name and ID in the corresponding fields, select
   "Furnish a new private key", and select "JSON" as the key type.

4. Clicking "Create" will download your `credentials`.
