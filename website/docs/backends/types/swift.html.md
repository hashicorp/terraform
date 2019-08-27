---
layout: "backend-types"
page_title: "Backend Type: swift"
sidebar_current: "docs-backends-types-standard-swift"
description: |-
  Terraform can store state remotely in Swift.
---

# swift

**Kind: Standard (with no locking)**

Stores the state as an artifact in [Swift](http://docs.openstack.org/developer/swift/latest/).

~> Warning! It is highly recommended that you enable [Object Versioning](https://docs.openstack.org/developer/swift/latest/overview_object_versioning.html) by setting the [`archive_container`](https://www.terraform.io/docs/backends/types/swift.html#archive_container) configuration. This allows for state recovery in the case of accidental deletions and human error.

## Example Configuration

```hcl
terraform {
  backend "swift" {
    container         = "terraform-state"
    archive_container = "terraform-state-archive"
  }
}
```
This will create a container called `terraform-state` and an object within that container called `tfstate.tf`. It will enable versioning using the `terraform-state-archive` container to contain the older version.

For the access credentials we recommend using a
[partial configuration](/docs/backends/config.html).

## Example Referencing

```hcl
data "terraform_remote_state" "foo" {
  backend = "swift"
  config = {
    container         = "terraform_state"
    archive_container = "terraform_state-archive"
  }
}
```

## Configuration variables

The following configuration options are supported:

 * `auth_url` - (Required) The Identity authentication URL. If omitted, the
   `OS_AUTH_URL` environment variable is used.

* `cloud` - (Optional; required if `auth_url` is not specified) An entry in a
  `clouds.yaml` file. See the OpenStack `os-client-config`
  [documentation](https://docs.openstack.org/os-client-config/latest/user/configuration.html)
  for more information about `clouds.yaml` files. If omitted, the `OS_CLOUD`
  environment variable is used.

 * `container` - (Required) The name of the container to create for storing
   the Terraform state file.

 * `state_name` - (Optional) The name of the state file in the container.
   Defaults to `tfstate.tf`.

 * `path` - (Optional) DEPRECATED: Use `container` instead.
   The name of the container to create in order to store the state file.

 * `user_name` - (Optional) The Username to login with. If omitted, the
   `OS_USERNAME` environment variable is used.

 * `user_id` - (Optional) The User ID to login with. If omitted, the
   `OS_USER_ID` environment variable is used.

 * `password` - (Optional) The Password to login with. If omitted, the
   `OS_PASSWORD` environment variable is used.

 * `application_credential_id` - (Optional) (Identity v3 only) The ID of an
    application credential to authenticate with. An
    `application_credential_secret` has to bet set along with this parameter.

 * `application_credential_name` - (Optional) (Identity v3 only) The name of an
    application credential to authenticate with. Conflicts with the
    `application_credential_name`, requires `user_id`, or `user_name` and
    `domain_name` (or `domain_id`) to be set.

 * `application_credential_secret` - (Optional) (Identity v3 only) The secret of an
    application credential to authenticate with. Required by
    `application_credential_id` or `application_credential_name`.

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
   omitted, the following environment variables are checked (in this order):
   `OS_USER_DOMAIN_ID`, `OS_PROJECT_DOMAIN_ID`, `OS_DOMAIN_ID`.

 * `domain_name` - (Optional) The Name of the Domain to scope to (Identity v3).
   If omitted, the following environment variables are checked (in this order):
   `OS_USER_DOMAIN_NAME`, `OS_PROJECT_DOMAIN_NAME`, `OS_DOMAIN_NAME`,
   `DEFAULT_DOMAIN`.

* `user_domain_name` - (Optional) The domain name where the user is located. If
  omitted, the `OS_USER_DOMAIN_NAME` environment variable is checked.

* `user_domain_id` - (Optional) The domain ID where the user is located. If
  omitted, the `OS_USER_DOMAIN_ID` environment variable is checked.

* `project_domain_name` - (Optional) The domain name where the project is
  located. If omitted, the `OS_PROJECT_DOMAIN_NAME` environment variable is
  checked.

* `project_domain_id` - (Optional) The domain ID where the project is located
  If omitted, the `OS_PROJECT_DOMAIN_ID` environment variable is checked.

* `default_domain` - (Optional) The ID of the Domain to scope to if no other
  domain is specified (Identity v3). If omitted, the environment variable
  `OS_DEFAULT_DOMAIN` is checked or a default value of "default" will be
  used.

 * `insecure` - (Optional) Trust self-signed SSL certificates. If omitted, the
   `OS_INSECURE` environment variable is used.

 * `cacert_file` - (Optional) Specify a custom CA certificate when communicating
   over SSL. If omitted, the `OS_CACERT` environment variable is used.

 * `cert` - (Optional) Specify client certificate file for SSL client authentication. 
   If omitted the `OS_CERT` environment variable is used.

 * `key` - (Optional) Specify client private key file for SSL client authentication. 
   If omitted the `OS_KEY` environment variable is used.

 * `archive_container` - (Optional) The container to create to store archived copies
   of the Terraform state file. If specified, Swift [object versioning](https://docs.openstack.org/developer/swift/latest/overview_object_versioning.html) is enabled on the container created at `container`.

 * `archive_path` - (Optional) DEPRECATED: Use `archive_container` instead.
   The path to store archived copied of `terraform.tfstate`. If specified,
   Swift [object versioning](https://docs.openstack.org/developer/swift/latest/overview_object_versioning.html) is enabled on the container created at `path`.

 * `expire_after` - (Optional) How long should the `terraform.tfstate` created at `container`
   be retained for? If specified, Swift [expiring object support](https://docs.openstack.org/developer/swift/latest/overview_expiring_objects.html) is enabled on the state. Supported durations: `m` - Minutes, `h` - Hours, `d` - Days.
   ~> **NOTE:** Since Terraform is inherently stateful - we'd strongly recommend against auto-expiring Statefiles.
