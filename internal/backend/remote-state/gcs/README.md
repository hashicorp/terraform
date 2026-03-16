# How to test the `gcs` backend

## Google Cloud resources needed for testing the backend

You will need access to a Google Cloud project that has Google Cloud Storage and Cloud Key Management Service (KMS) APIs enabled. You will need sufficient permissions to create and manage resources in those services:
*[IAM roles for Cloud Storage](https://cloud.google.com/storage/docs/access-control/iam-roles).
*[IAM roles for Cloud Key Management Service](https://cloud.google.com/kms/docs/iam).

For HashiCorp employees, [a temporary GCP project can be created for testing here](https://doormat.hashicorp.services/gcp/project/temp/create).

No infrastructure needs to be set up within that project before running the tests for the `gcs` backend; tests will provision and delete GCS buckets and KMS keys themselves. However if you want to use service accounts for accessing the project you will need to create those.

## Set up credentials and access

These instructions use [application default credentials](https://cloud.google.com/docs/authentication/application-default-credentials) from a Google Account for simplicity. If you want to use a service account instead, see this documentation on how to [create a key file](https://cloud.google.com/iam/docs/keys-create-delete) and reference that file when providing credentials through environment variables.

1. Run `gcloud auth application-default login` and log in with your Google Account when prompted in the browser. This will create a file at `~/.config/gcloud/application_default_credentials.json` on your machine.
1. Set these environment variables:
    * `GOOGLE_CREDENTIALS=~/.config/gcloud/application_default_credentials.json` (Required) - this file is created in the previous step.
    * `GOOGLE_PROJECT=<project-id>` (Required) - here, use the project id that you want to be linked to the GCS buckets created in the tests.
    * `TF_ACC=1` (Required) - this signals that you're happy for the test to provision real infrastructure.
    * `GOOGLE_REGION=<region>` (Required) - This region name is used to set the region of the GCS bucket and the region that customer-managed encryption keys are created in. If in doubt, use "us-central1".

## Run the tests!

Run tests in the `internal/backend/remote-state/gcs` package with the above environment variables set.
If any errors indicate an issue with credentials or permissions, please review how you're providing credentials to the code and whether sufficient permissions are present.