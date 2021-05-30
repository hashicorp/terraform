---
layout: "language"
page_title: "Backend Type: gcs"
sidebar_current: "docs-backends-types-standard-gcs"
description: |-
  Terraform can store the state remotely, making it easier to version and work with in a team.
---

# gcs

**Kind: Standard (with locking)**

Stores the state as an object in a configurable prefix in a pre-existing bucket on [Google Cloud Storage](https://cloud.google.com/storage/) (GCS).
This backend also supports [state locking](/docs/language/state/locking.html). The bucket must exist prior to configuring the backend.

~> **Warning!** It is highly recommended that you enable
[Object Versioning](https://cloud.google.com/storage/docs/object-versioning)
on the GCS bucket to allow for state recovery in the case of accidental deletions and human error.

## Example Configuration

```hcl
terraform {
  backend "gcs" {
    bucket  = "tf-state-prod"
    prefix  = "terraform/state"
  }
}
```

## Data Source Configuration

```hcl
data "terraform_remote_state" "foo" {
  backend = "gcs"
  config = {
    bucket  = "terraform-state"
    prefix  = "prod"
  }
}

resource "template_file" "bar" {
  template = "${greeting}"

  vars {
    greeting = "${data.terraform_remote_state.foo.greeting}"
  }
}
```

## Authentication

IAM Changes to buckets are [eventually consistent](https://cloud.google.com/storage/docs/consistency#eventually_consistent_operations) and may take upto a few minutes to take effect. Terraform will return 403 errors till it is eventually consistent.

### Running Terraform on your workstation.

If you are using terraform on your workstation, you will need to install the Google Cloud SDK and authenticate using [User Application Default
Credentials](https://cloud.google.com/sdk/gcloud/reference/auth/application-default).

User ADCs do [expire](https://developers.google.com/identity/protocols/oauth2#expiration) and you can refresh them by running `gcloud auth application-default login`.

### Running Terraform on Google Cloud

If you are running terraform on Google Cloud, you can configure that instance or cluster to use a [Google Service
Account](https://cloud.google.com/compute/docs/authentication). This will allow Terraform to authenticate to Google Cloud without having to bake in a separate
credential/authentication file. Make sure that the scope of the VM/Cluster is set to cloud-platform.

### Running Terraform outside of Google Cloud

If you are running terraform outside of Google Cloud, generate a service account key and set the `GOOGLE_APPLICATION_CREDENTIALS` environment variable to
the path of the service account key. Terraform will use that key for authentication.

### Impersonating Service Accounts

Terraform can impersonate a Google Service Account as described [here](https://cloud.google.com/iam/docs/creating-short-lived-service-account-credentials). A valid credential must be provided as mentioned in the earlier section and that identity must have the `roles/iam.serviceAccountTokenCreator` role on the service account you are impersonating.

## Configuration variables

The following configuration options are supported:

 *  `bucket` - (Required) The name of the GCS bucket.  This name must be
    globally unique.  For more information, see [Bucket Naming
    Guidelines](https://cloud.google.com/storage/docs/bucketnaming.html#requirements).
 *  `credentials` / `GOOGLE_BACKEND_CREDENTIALS` / `GOOGLE_CREDENTIALS` -
    (Optional) Local path to Google Cloud Platform account credentials in JSON
    format.  If unset, [Google Application Default
    Credentials](https://developers.google.com/identity/protocols/application-default-credentials)
    are used.  The provided credentials must have Storage Object Admin role on the bucket.
    **Warning**: if using the Google Cloud Platform provider as well, it will
    also pick up the `GOOGLE_CREDENTIALS` environment variable.
 * `impersonate_service_account` - (Optional) The service account to impersonate for accessing the State Bucket.
    You must have `roles/iam.serviceAccountTokenCreator` role on that account for the impersonation to succeed. 
    If you are using a delegation chain, you can specify that using the `impersonate_service_account_delegates` field.
    Alternatively, this can be specified using the `GOOGLE_IMPERSONATE_SERVICE_ACCOUNT` environment
    variable.
 * `impersonate_service_account_delegates` - (Optional) The delegation chain for an impersonating a service account as described [here](https://cloud.google.com/iam/docs/creating-short-lived-service-account-credentials#sa-credentials-delegated).
 * `access_token` - (Optional) A temporary [OAuth 2.0 access token] obtained
   from the Google Authorization server, i.e. the `Authorization: Bearer` token
   used to authenticate HTTP requests to GCP APIs. This is an alternative to
   `credentials`. If both are specified, `access_token` will be used over the
   `credentials` field.
 *  `prefix` - (Optional) GCS prefix inside the bucket. Named states for
    workspaces are stored in an object called `<prefix>/<name>.tfstate`.
 *  `encryption_key` / `GOOGLE_ENCRYPTION_KEY` - (Optional) A 32 byte base64
    encoded 'customer supplied encryption key' used to encrypt all state. For
    more information see [Customer Supplied Encryption
    Keys](https://cloud.google.com/storage/docs/encryption#customer-supplied).
