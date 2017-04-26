---
layout: "backend-types"
page_title: "Backend Type: swift"
sidebar_current: "docs-backends-types-standard-swift"
description: |-
  Terraform can store state remotely in Swift.
---

# swift

**Kind: Standard (with no locking)**

Stores the state as an artifact in [Swift](http://docs.openstack.org/developer/swift/).

## Example Configuration

```hcl
terraform {
  backend "swift" {
    path = "terraform-state"
  }
}
```

Note that for the access credentials we recommend using a
[partial configuration](/docs/backends/config.html).

## Example Referencing

```hcl
data "terraform_remote_state" "foo" {
  backend = "swift"
  config {
    path = "terraform_state"
  }
}
```

## Configuration variables

The following configuration options are supported:

 * `auth_url` - (Required) The Identity authentication URL. If omitted, the
   `OS_AUTH_URL` environment variable is used.

 * `path` - (Required) The path where to store `terraform.tfstate`.
 * `user_name` - (Optional) The Username to login with. If omitted, the
   `OS_USERNAME` environment variable is used.

 * `user_id` - (Optional) The User ID to login with. If omitted, the
   `OS_USER_ID` environment variable is used.

 * `password` - (Optional) The Password to login with. If omitted, the
   `OS_PASSWORD` environment variable is used.

 * `token` - (Optional) Access token to login with instead of user and password.
    If omitted, the `OS_AUTH_TOKEN` variable is used.

 * `region_name` (Required) - The region in which to store `terraform.tfstate`. If
   omitted, the `OS_REGION_NAME` environment variable is used.

 * `tenant_id` (Optional) The ID of the Tenant (Identity v2) or Project
   (Identity v3) to login with. If omitted, the `OS_TENANT_ID` or
   `OS_PROJECT_ID` environment variables are used.

 * `tenant_name` - (Optional) The Name of the Tenant (Identity v2) or Project
   (Identity v3) to login with. If omitted, the `OS_TENANT_NAME` or
   `OS_PROJECT_NAME` environment variable are used.

 * `domain_id` - (Optional) The ID of the Domain to scope to (Identity v3). If
   If omitted, the following environment variables are checked (in this order):
   `OS_USER_DOMAIN_ID`, `OS_PROJECT_DOMAIN_ID`, `OS_DOMAIN_ID`.

 * `domain_name` - (Optional) The Name of the Domain to scope to (Identity v3).
   If omitted, the following environment variables are checked (in this order):
   `OS_USER_DOMAIN_NAME`, `OS_PROJECT_DOMAIN_NAME`, `OS_DOMAIN_NAME`,
   `DEFAULT_DOMAIN`.

 * `insecure` - (Optional) Trust self-signed SSL certificates. If omitted, the
   `OS_INSECURE` environment variable is used.

 * `cacert_file` - (Optional) Specify a custom CA certificate when communicating
   over SSL. If omitted, the `OS_CACERT` environment variable is used.

 * `cert` - (Optional) Specify client certificate file for SSL client
   authentication. If omitted the `OS_CERT` environment variable is used.

 * `key` - (Optional) Specify client private key file for SSL client
   authentication. If omitted the `OS_KEY` environment variable is used.

 * `archive_path` - (Optional) The path to store archived copied of `terraform.tfstate`.
   If specified, Swift object versioning is enabled on the container created at `path`.

 * `expire_after` - (Optional) How long should the `terraform.tfstate` created at `path`
   be retained for? Supported durations: `m` - Minutes, `h` - Hours, `d` - Days.
