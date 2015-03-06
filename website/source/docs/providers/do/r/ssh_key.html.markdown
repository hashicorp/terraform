---
layout: "digitalocean"
page_title: "DigitalOcean: digitalocean_ssh_key"
sidebar_current: "docs-do-resource-ssh-key"
description: |-
  Provides a DigitalOcean SSH key resource.
---

# digitalocean\_ssh_key

Provides a DigitalOcean SSH key resource to allow you manage SSH
keys for Droplet access. Keys created with this resource
can be referenced in your droplet configuration via their ID or
fingerprint.

## Example Usage

```
# Create a new SSH key
resource "digitalocean_ssh_key" "default" {
    name = "Terraform Example"
    public_key = "${file("/Users/terraform/.ssh/id_rsa.pub")}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the SSH key for identification
* `public_key` - (Required) The public key. If this is a file, it
can be read using the file interpolation function

## Attributes Reference

The following attributes are exported:

* `id` - The unique ID of the key
* `name` - The name of the SSH key
* `public_key` - The text of the public key
* `fingerprint` - The fingerprint of the SSH key
