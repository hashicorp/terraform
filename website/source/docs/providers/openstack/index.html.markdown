---
layout: "openstack"
page_title: "Provider: OpenStack"
sidebar_current: "docs-openstack-index"
description: |-
  The OpenStack provider is used to interact with the many resources supported by OpenStack. The provider needs to be configured with the proper credentials before it can be used.
---

# OpenStack Provider

The OpenStack provider is used to interact with the
many resources supported by OpenStack. The provider needs to be configured
with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the OpenStack Provider
provider "openstack" {
  user_name   = "admin"
  tenant_name = "admin"
  password    = "pwd"
  auth_url    = "http://myauthurl:5000/v2.0"
}

# Create a web server
resource "openstack_compute_instance_v2" "test-server" {
  # ...
}
```

## Configuration Reference

The following arguments are supported:

* `auth_url` - (Required) The Identity authentication URL. If omitted, the
  `OS_AUTH_URL` environment variable is used.

* `user_name` - (Optional) The Username to login with. If omitted, the
  `OS_USERNAME` environment variable is used.

* `user_id` - (Optional) The User ID to login with. If omitted, the
  `OS_USER_ID` environment variable is used.

* `tenant_id` - (Optional) The ID of the Tenant (Identity v2) or Project
  (Identity v3) to login with. If omitted, the `OS_TENANT_ID` or
  `OS_PROJECT_ID` environment variables are used.

* `tenant_name` - (Optional) The Name of the Tenant (Identity v2) or Project
  (Identity v3) to login with. If omitted, the `OS_TENANT_NAME` or
  `OS_PROJECT_NAME` environment variable are used.

* `password` - (Optional) The Password to login with. If omitted, the
  `OS_PASSWORD` environment variable is used.

* `token` - (Optional; Required if not using `user_name` and `password`)
  A token is an expiring, temporary means of access issued via the Keystone
  service. By specifying a token, you do not have to specify a username/password
  combination, since the token was already created by a username/password out of
  band of Terraform. If omitted, the `OS_AUTH_TOKEN` environment variable is used.

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
  over SSL. You can specify either a path to the file or the contents of the
  certificate. If omitted, the `OS_CACERT` environment variable is used.

* `cert` - (Optional) Specify client certificate file for SSL client
  authentication. You can specify either a path to the file or the contents of
  the certificate. If omitted the `OS_CERT` environment variable is used.

* `key` - (Optional) Specify client private key file for SSL client
  authentication. You can specify either a path to the file or the contents of
  the key. If omitted the `OS_KEY` environment variable is used.

* `endpoint_type` - (Optional) Specify which type of endpoint to use from the
  service catalog. It can be set using the OS_ENDPOINT_TYPE environment
  variable. If not set, public endpoints is used.

* `swauth` - (Optional) Set to `true` to authenticate against Swauth, a
  Swift-native authentication system. If omitted, the `OS_SWAUTH` environment
  variable is used. You must also set `username` to the Swauth/Swift username
  such as `username:project`. Set the `password` to the Swauth/Swift key.
  Finally, set `auth_url` as the location of the Swift service. Note that this
  will only work when used with the OpenStack Object Storage resources.

## Additional Logging

This provider has the ability to log all HTTP requests and responses between
Terraform and the OpenStack cloud which is useful for troubleshooting and
debugging.

To enable these logs, set the `OS_DEBUG` environment variable to `1` along
with the usual `TF_LOG=DEBUG` environment variable:

```shell
$ OS_DEBUG=1 TF_LOG=DEBUG terraform apply
```

If you submit these logs with a bug report, please ensure any sensitive
information has been scrubbed first!

## Rackspace Compatibility

Using this OpenStack provider with Rackspace is not supported and not
guaranteed to work; however, users have reported success with the
following notes in mind:

* Interacting with instances has been seen to work. Interacting with
all other resources is either untested or known to not work.

* Use your _password_ instead of your Rackspace API KEY.

* Explicitly define the public and private networks in your
instances as shown below:

```
resource "openstack_compute_instance_v2" "my_instance" {
  name      = "my_instance"
  region    = "DFW"
  image_id  = "fabe045f-43f8-4991-9e6c-5cabd617538c"
  flavor_id = "general1-4"
  key_pair  = "provisioning_key"

  network {
    uuid = "00000000-0000-0000-0000-000000000000"
    name = "public"
  }

  network {
    uuid = "11111111-1111-1111-1111-111111111111"
    name = "private"
  }
}
```

If you try using this provider with Rackspace and run into bugs, you
are welcomed to open a bug report / issue on Github, but please keep
in mind that this is unsupported and the reported bug may not be
able to be fixed.

If you have successfully used this provider with Rackspace and can
add any additional comments, please let us know.

## Testing and Development

In order to run the Acceptance Tests for development, the following environment
variables must also be set:

* `OS_REGION_NAME` - The region in which to create the server instance.

* `OS_IMAGE_ID` or `OS_IMAGE_NAME` - a UUID or name of an existing image in
    Glance.

* `OS_FLAVOR_ID` or `OS_FLAVOR_NAME` - an ID or name of an existing flavor.

* `OS_POOL_NAME` - The name of a Floating IP pool.

* `OS_NETWORK_ID` - The UUID of a network in your test environment.

* `OS_EXTGW_ID` - The UUID of the external gateway.

You should be able to use any OpenStack environment to develop on as long as the
above environment variables are set.

Most of Terraform's OpenStack support is done in a standardized Packstack
all-in-one environment. You can find the scripts to build this environment
[here](https://github.com/jtopjian/terraform-devstack/tree/master/packstack-standard).
The included `main.tf` file will need to be modified for your specific
environment. Once it's up and running, you will have access to a standard,
up-to-date OpenStack environment with the latest OpenStack services.

If you require access to deprecated services, such as Keystone v2 and
LBaaS v1, you can use the "legacy" environment
[here](https://github.com/jtopjian/terraform-devstack/tree/master/packstack-legacy).
