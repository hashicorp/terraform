---
layout: "backend-types"
page_title: "Backend Type: minio"
sidebar_current: "docs-backends-types-standard-minio"
description: |-
  Terraform can store state remotely in Minio.
---

# Minio

**Kind: Standard (with no locking)**

Stores the state as a given object_name in a given bucket on
[Minio](https://minio.io/) or on any S3-compatible object
store that the 
[Minio client](http://docs.minio.io/docs/golang-client-quickstart-guide) 
supports (including those requiring both AWS Signature V4 and V2).

Supported object stores include:

 * [Amazon S3](https://aws.amazon.com/s3/)
 * [Minio](https://minio.io/)
 * [Google Cloud Storage (S3 Compatibility Mode)](https://cloud.google.com/storage/docs/migrating#migration-simple)
 * Openstack Swift with [Swift3 Middleware](https://github.com/openstack/swift3)
 * [Ceph Object Gateway (radosgw)](http://docs.ceph.com/docs/master/radosgw/)
 * [Riak CS](http://docs.basho.com/riak/cs/)

## Example Configuration

```
terraform {
  backend "minio" {
    bucket_name     = "mybucket"
    object_name     = "path/to/my/key"
    bucket_location = "us-east-1"
  }
}
```

This assumes we have a bucket created called `mybucket`. The
Terraform state is written to an object named `path/to/my/key`.

## Using the Minio remote state

To make use of the Minio remote state we can use the
[`terraform_remote_state` data
source](/docs/providers/terraform/d/remote_state.html).

```
data "terraform_remote_state" "foo" {
	backend = "minio"
	config {
		bucket_name = "terraform-state-prod"
		object_name = "network/terraform.tfstate"
		bucket_location = "us-east-1"
	}
}
```

The `terraform_remote_state` data source will return all of the root outputs
defined in the referenced remote state, an example output might look like:

```
data.terraform_remote_state.network:
  id = 2016-10-29 01:57:59.780010914 +0000 UTC
  addresses.# = 2
  addresses.0 = 52.207.220.222
  addresses.1 = 54.196.78.166
  backend = minio
  config.% = 3
  config.bucket = terraform-state-prod
  config.key = network/terraform.tfstate
  config.region = us-east-1
  elb_address = web-elb-790251200.us-east-1.elb.amazonaws.com
  public_subnet_id = subnet-1e05dd33
```

## Configuration variables

The following configuration options or environment variables are supported:

 * `endpoint` / `MINIO_ENDPOINT` - (Required)
A custom endpoint (host:port) for the Minio-compatible S3 API.
 * `access_key_id` / `MINIO_ACCESS_KEY_ID` - (Required)
Minio/S3 access key id.
 * `secret_access_key` / `MINIO_SECRET_ACCESS_KEY` - (Required)
Minio/S3 secret access key.
 * `bucket_name` / `MINIO_BUCKET_NAME` - (Required)
The name of the bucket in which to store the object (will be created if it does
not exist).
 * `object_name` / `MINIO_OBJECT_NAME` - (Required)
The path to the state file object inside the bucket.
 * `bucket_location` / `MINIO_BUCKET_LOCATION` - (Optional)
The location of the Minio bucket (equivalent to region in Amazon S3).
 * `use_ssl` / `MINIO_USE_SSL` - (Optional)
Whether to use SSL (https) for the connection to the API endpoint
(default: true).
