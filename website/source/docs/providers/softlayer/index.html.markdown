---
layout: "softlayer"
page_title: "Provider: SoftLayer"
sidebar_current: "docs-softlayer-index"
description: |-
  The Docker provider is used to interact with Docker containers and images.
---

# SoftLayer Provider

The SoftLayer provider is used to manage SoftLayer resources.

Use the navigation to the left to read about the available resources.

<div class="alert alert-block alert-info">
<strong>Note:</strong> The SoftLayer provider is new as of Terraform 0.X.
It is ready to be used but many features are still being added. If there
is a SoftLayer feature missing, please report it in the GitHub repo.
</div>

## Example Usage

Here is an example that will setup the following:
+ An SSH key resource.
+ A virtual server resource that uses an existing SSH key.
+ A virtual server resource using an existing SSH key and a Terraform managed SSH key (created as "test_key_1" in the example below).

(create this as sl.tf and run terraform commands from this directory):

```hcl
provider "softlayer" {
    username = ""
    api_key = ""
}

# This will create a new SSH key that will show up under the \
# Devices>Manage>SSH Keys in the SoftLayer console.
resource "softlayer_ssh_key" "test_key_1" {
    name = "test_key_1"
    public_key = "${file(\"~/.ssh/id_rsa_test_key_1.pub\")}"
    # Windows Example:
    # public_key = "${file(\"C:\ssh\keys\path\id_rsa_test_key_1.pub\")}"
}

# Virtual Server created with existing SSH Key already in SoftLayer \
# inventory and not created using this Terraform template.
resource "softlayer_virtual_guest" "my_server_1" {
    name = "my_server_1"
    domain = "example.com"
    ssh_keys = ["123456"]
    image = "DEBIAN_7_64"
    region = "ams01"
    public_network_speed = 10
    cpu = 1
    ram = 1024
}

# Virtual Server created with a mix of previously existing and \
# Terraform created/managed resources.
resource "softlayer_virtual_guest" "my_server_2" {
    name = "my_server_2"
    domain = "example.com"
    ssh_keys = ["123456", "${softlayer_ssh_key.test_key_1.id}"]
    image = "CENTOS_6_64"
    region = "ams01"
    public_network_speed = 10
    cpu = 1
    ram = 1024
}
```

You'll need to provide your SoftLayer username and API key,
so that Terraform can connect. If you don't want to put
credentials in your configuration file, you can leave them
out:

```
provider "softlayer" {}
```

...and instead set these environment variables:

- **SOFTLAYER_USERNAME**: Your SoftLayer username
- **SOFTLAYER_API_KEY**: Your API key
