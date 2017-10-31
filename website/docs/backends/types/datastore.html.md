---
layout: "backend-types"
page_title: "Backend Type: datastore"
sidebar_current: "docs-backends-types-standard-datastore"
description: |-
  Terraform can store the state remotely, making it easier to version and work with in a team.
---

# datastore

**Kind: Standard (with locking)**

Stores the state as an entity in
[Google Cloud Datastore](https://cloud.google.com/datastore/).

~> This backend uses global Datastore queries to determine which workspaces
exist. Global Datastore queries are
**[eventually consistent](https://cloud.google.com/datastore/docs/articles/balancing-strong-and-eventual-consistency-with-google-cloud-datastore/#what-is-eventual-consistency)**.
This means it *may* take a short time for `terraform workspace list` to notice
that a workspace has been added or deleted. Strongly consistent Datastore
operations are used for all other functionality. Selecting, locking, reading,
and updating a workspace will always benefit from strong consistency.

~> This backend stores a workspace's state in a Datastore blob property. Blob
properties are **limited to just under 1MB**. The backend attempts to avoid
this limitation by zlib compressing the state, but may not be suitable for very
large or complex workspaces.

## Example Configuration

```hcl
terraform {
  backend "datastore" {
    project          = "myproject"
    namespace        = "tf-state-prod"
    credentials_file = "path/to/googlecreds.json"
  }
}
```

## Example Referencing

```hcl
data "terraform_remote_state" "foo" {
  backend = "datastore"
  config {
    project          = "myproject"
    namespace        = "tf-state-prod"
    credentials_file = "path/to/googlecreds.json"
  }
}

resource "template_file" "bar" {
  template = "${greeting}"

  vars {
    greeting = "${data.terraform_remote_state.foo.greeting}"
  }
}
```

## Configuration variables

The following configuration options are supported:

 * `project` - (Required) The ID of the project to apply any resources to.  This
  can also be specified using any of the following environment variables (listed
  in order of precedence):

    * `GOOGLE_PROJECT`
    * `GCLOUD_PROJECT`
    * `CLOUDSDK_CORE_PROJECT`
* `namespace` / `GOOGLE_DATASTORE_NAMESPACE` - (Optional) The Datastore
  [namespace](https://cloud.google.com/datastore/docs/concepts/multitenancy)
  in which to store state. Leave unset to use the default namespace in the
  supplied project.
 * `credentials_file` - (Optional) Path to the JSON file used to describe your
  account credentials, downloaded from Google Cloud Console.

  The [`GOOGLE_APPLICATION_CREDENTIALS`](https://developers.google.com/identity/protocols/application-default-credentials#howtheywork)
  environment variable can also contain the path of a file to obtain credentials
  from.

  If no credentials are specified, the backend will fall back to using the
  [Google Application Default Credentials](https://developers.google.com/identity/protocols/application-default-credentials).
  If you are running Terraform from a GCE instance, see
  [Creating and Enabling Service Accounts for Instances](https://cloud.google.com/compute/docs/authentication)
  for details. On your computer, if you have made your identity available as the
  Application Default Credentials by running
  [`gcloud auth application-default login`](https://cloud.google.com/sdk/gcloud/reference/auth/application-default/login),
  the backend will use your identity.
