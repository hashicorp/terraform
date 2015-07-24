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
    account_file = "account.json"
    project = "my-gce-project"
    region = "us-central1"
}

# Create a new instance
resource "google_compute_instance" "default" {
    ...
}
```

## Configuration Reference

The following keys can be used to configure the provider.

* `account_file` - (Required, unless `account_file_contents` is present) Path
  to the JSON file used to describe your account credentials, downloaded from
  Google Cloud Console. More details on retrieving this file are below. The
  _account file_ can be "" if you are running terraform from a GCE instance with
  a properly-configured [Compute Engine Service
  Account](https://cloud.google.com/compute/docs/authentication). This can also
  be specified with the `GOOGLE_ACCOUNT_FILE` shell environment variable.

* `account_file_contents` - (Required, unless `account_file` is present) The
  contents of `account_file`. This can be used to pass the account credentials
  with a Terraform var or environment variable if the account file is not
  accessible. This can also be specified with the `GOOGLE_ACCOUNT_FILE_CONTENTS`
  shell environment variable.

* `project` - (Required) The ID of the project to apply any resources to.  This
  can also be specified with the `GOOGLE_PROJECT` shell environment variable.

* `region` - (Required) The region to operate under. This can also be specified
  with the `GOOGLE_REGION` shell environment variable.

## Authentication JSON File

Authenticating with Google Cloud services requires a JSON
file which we call the _account file_.

This file is downloaded directly from the
[Google Developers Console](https://console.developers.google.com). To make
the process more straightforwarded, it is documented here:

1. Log into the [Google Developers Console](https://console.developers.google.com)
   and select a project.

2. Under the "APIs & Auth" section, click "Credentials."

3. Create a new OAuth client ID and select "Service account" as the type
   of account. Once created, and after a P12 key is downloaded, a JSON file should be downloaded. This is your _account file_.
