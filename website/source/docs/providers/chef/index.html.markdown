---
layout: "chef"
page_title: "Provider: Chef"
sidebar_current: "docs-chef-index"
description: |-
  Chef is a systems and cloud infrastructure automation framework.
---

# Chef Provider

[Chef](https://www.chef.io/) is a systems and cloud infrastructure automation
framework. The Chef provider allows Terraform to manage various resources
that exist within [Chef Server](http://docs.chef.io/chef_server.html).

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the Chef provider
provider "chef" {
  server_url = "https://api.chef.io/organizations/example/"

  # You can set up a "Client" within the Chef Server management console.
  client_name  = "terraform"
  key_material = "${file("chef-terraform.pem")}"
}

# Create a Chef Environment
resource "chef_environment" "production" {
  name = "production"
}

# Create a Chef Role
resource "chef_role" "app_server" {
  name = "app_server"

  run_list = [
    "recipe[terraform]",
  ]
}
```

## Argument Reference

The following arguments are supported:

* `server_url` - (Required) The HTTP(S) API URL of the Chef server to use. If
  the target Chef server supports organizations, use the full URL of the
  organization you wish to configure. May be provided instead via the
  ``CHEF_SERVER_URL`` environment variable.
* `client_name` - (Required) The name of the client account to use when making
  requests. This must have been already configured on the Chef server.
  May be provided instead via the ``CHEF_CLIENT_NAME`` environment variable.
* `key_material` - (Required) The PEM-formatted private key contents belonging to
  the configured client. This is issued by the server when a new client object
  is created. May be provided via the
  ``CHEF_PRIVATE_KEY_FILE`` environment variable.
* `allow_unverified_ssl` - (Optional) Boolean indicating whether to make
  requests to a Chef server whose SSL certicate cannot be verified. Defaults
  to ``false``.
