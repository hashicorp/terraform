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

```
# Configure the OpenStack Provider
provider "openstack" {
    user_name  = "admin"
    tenant_name = "admin"
    password  = "pwd"
    auth_url  = "http://myauthurl:5000/v2.0"
}

# Create a web server
resource "openstack_compute_instance_v2" "test-server" {
    ...
}
```

## Configuration Reference

The following arguments are supported:

* `auth_url` - (Required) If omitted, the `OS_AUTH_URL` environment
    variable is used.

* `user_name` - (Optional; Required for Identity V2) If omitted, the
    `OS_USERNAME` environment variable is used.

* `user_id` - (Optional)

* `password` - (Optional; Required if not using `api_key`) If omitted, the
    `OS_PASSWORD` environment variable is used.

* `token` - (Optional; Required if not using `user_name` and `password`)
    A token is an expiring, temporary means of access issued via the
    Keystone service. By specifying a token, you do not have to
    specify a username/password combination, since the token was
    already created by a username/password out of band of Terraform.
    If ommitted, the `OS_AUTH_TOKEN` environment variable is used.

* `api_key` - (Optional; Required if not using `password`) An API Key
    is issued by a cloud provider as alternative password. Unless
    your cloud provider has documentation referencing an API Key,
    you can safely ignore this argument. If omitted, the `OS_API_KEY`
    environment variable is used.

* `project_id` - (Optional) Project-level authentication scope. If omitted,
    the `OS_PROJECT_ID` environment variable is used.

* `project_name` - (Optional) `project_id` alternative. If omitted,
    the `OS_PROJECT_NAME` environment variable is used.

* `tenant_id` - (Optional)

* `tenant_name` - (Optional) If omitted, the `OS_TENANT_NAME` environment
    variable is used.

* `insecure` - (Optional) Explicitly allow the provider to perform
    "insecure" SSL requests. If omitted, default value is `false`

* `cacert_file` - (Optional) Specify a custom CA certificate when communicating
    over SSL. If omitted, the `OS_CACERT` environment variable is used.

* `endpoint_type` - (Optional) Specify which type of endpoint to use from the
    service catalog. It can be set using the OS_ENDPOINT_TYPE environment
    variable. If not set, public endpoints is used.

Authenticating using Identity Server API v3:

* `user_domain_id` - (Optional) If the user is specified by name, then the
    domain id of the user must also be specified in order to uniquely identify
    the user. If omitted, the `OS_USER_DOMAIN_ID` environment variable is used.

* `user_domain_name` - (Optional) Alternatively to `user_domain_id`, the
    domain name of the user may be used to uniquely identify the user. If
    omitted, the `OS_USER_DOMAIN_NAME` environment variable is used.

* `project_domain_id` - (Optional) Project scoping using the project domain
    id. If omitted, the `OS_PROJECT_DOMAIN_ID` environment variable is used.

* `project_domain_name` - (Optional) Project scoping using the project domain
    name. If omitted, the `OS_PROJECT_DOMAIN_NAME` environment variable is used.

* `domain_id` - (Optional) Domain scoping using the domain id. If omitted,
    the `OS_DOMAIN_ID` environment variable is used.

* `domain_name` - (Optional) Domain scoping using the domain name. If omitted,
    the `OS_DOMAIN_NAME` environment variable is used.

* `default_domain` - (Optional) Default domain id if the user and project
    share the same domain. If omitted, the `OS_DEFAULT_DOMAIN` environment
    variable is used. 

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
  name = "my_instance"
  region = "DFW"
  image_id = "fabe045f-43f8-4991-9e6c-5cabd617538c"
  flavor_id = "general1-4"
  key_pair = "provisioning_key"

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

To make development easier, the `builtin/providers/openstack/devstack/deploy.sh`
script will assist in installing and configuring a standardized
[DevStack](http://docs.openstack.org/developer/devstack/) environment along with
Golang, Terraform, and all development dependencies. It will also set the required
environment variables in the `devstack/openrc` file.

Do not run the `deploy.sh` script on your workstation or any type of production
server. Instead, run the script within a disposable virtual machine.
[Here's](https://github.com/berendt/terraform-configurations) an example of a
Terraform configuration that will create an OpenStack instance and then install and
configure DevStack inside.
