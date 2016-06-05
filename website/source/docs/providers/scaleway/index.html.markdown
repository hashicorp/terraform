---
layout: "scaleway"
page_title: "Provider: Scaleway"
sidebar_current: "docs-scaleway-index"
description: |-
  The Docker provider is used to interact with Docker containers and images.
---

# Scaleway Provider

The Scaleway provider is used to manage Scaleway resources.

Use the navigation to the left to read about the available resources.

<div class="alert alert-block alert-info">
<strong>Note:</strong> The Scaleway provider is new as of Terraform 0.X.
It is ready to be used but many features are still being added. If there
is a SoftLayer feature missing, please report it in the GitHub repo.
</div>

## Example Usage

Here is an example that will setup the following:
+ An ARM Server.
+ An IP Address.
+ A security group.

(create this as sl.tf and run terraform commands from this directory):

```hcl
provider "scaleway" {
  api_key = ""
  organization = ""
}

resource "scaleway_ip" "ip" {
  server = "${scaleway_server.test.id}"
}

resource "scaleway_server" "test" {
  name = "test"
  image = "aecaed73-51a5-4439-a127-6d8229847145"
  type = "C2S"
}

resource "scaleway_volume" "test" {
  name = "test"
  size_in_gb = 20
  type = "l_ssd"
}

resource "scaleway_volume_attachment" "test" {
  server = "${scaleway_server.test.id}"
  volume = "${scaleway_volume.test.id}"
}

resource "scaleway_security_group" "http" {
  name = "http"
  description = "allow HTTP and HTTPS traffic"
}

resource "scaleway_security_group_rule" "http_accept" {
  security_group = "${scaleway_security_group.http.id}"

  action = "accept"
  direction = "inbound"
  ip_range = "0.0.0.0/0"
  protocol = "TCP"
  dest_port_from = 80
}

resource "scaleway_security_group_rule" "https_accept" {
  security_group = "${scaleway_security_group.http.id}"

  action = "accept"
  direction = "inbound"
  ip_range = "0.0.0.0/0"
  protocol = "TCP"
  dest_port_from = 443
}

```

You'll need to provide your Scaleway organization and Auth token,
so that Terraform can connect. If you don't want to put
credentials in your configuration file, you can leave them
out:

```
provider "scaleway" {}
```

...and instead set these environment variables:

- **SCALEWAY_ORGANIZATION**: Your Scaleway organization
- **SCALEWAY_API_KEY**: Your API key
