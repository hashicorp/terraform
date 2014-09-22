---
layout: "google"
page_title: "Provider: Google Cloud"
sidebar_current: "docs-google-index"
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
    client_secrets_file = "client_secrets.json"
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

* `account_file` - (Required) Path to the JSON file used to describe
  your account credentials, downloaded from Google Cloud Console. More
  details on retrieving this file are below.

* `client_secrets_file` - (Required) Path to the JSON file containing
  the secrets for your account, downloaded from Google Cloud Console.
  More details on retrieving this file are below.

* `project` - (Required) The name of the project to apply any resources to.

* `region` - (Required) The region to operate under.

## Authentication JSON Files

Authenticating with Google Cloud services requires two separate JSON
files: one which we call the _account file_ and the _client secrets file_.

Both of these files are downloaded directly from the
[Google Developers Console](https://console.developers.google.com). To make
the process more straightforwarded, it is documented here.

1. Log into the [Google Developers Console](https://console.developers.google.com)
   and select a project.

2. Under the "APIs & Auth" section, click "Credentials."

3. Create a new OAuth client ID and select "Installed application" as the
   type of account. Once created, click the "Download JSON" button underneath
   the account. The file should start with "client\_secret". This is your _client
   secrets file_.

4. Create a new OAuth client ID and select "Service account" as the type
   of account. Once created, a JSON file should be downloaded. This is your
   _account file_.
