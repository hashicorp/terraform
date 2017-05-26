---
layout: "alicloud"
page_title: "Provider: alicloud"
sidebar_current: "docs-alicloud-index"
description: |-
  The Alicloud provider is used to interact with many resources supported by Alicloud. The provider needs to be configured with the proper credentials before it can be used.
---

# Alicloud Provider

The Alicloud provider is used to interact with the
many resources supported by [Alicloud](https://www.aliyun.com). The provider needs to be configured
with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the Alicloud Provider
provider "alicloud" {
  access_key = "${var.access_key}"
  secret_key = "${var.secret_key}"
  region     = "${var.region}"
}

# Create a web server
resource "alicloud_instance" "web" {
  # cn-beijing
  provider          = "alicloud"
  availability_zone = "cn-beijing-b"
  image_id          = "ubuntu_140405_32_40G_cloudinit_20161115.vhd"

  instance_network_type = "Classic"
  internet_charge_type  = "PayByBandwidth"

  instance_type        = "ecs.n1.medium"
  io_optimized         = "optimized"
  system_disk_category = "cloud_efficiency"
  security_groups      = ["${alicloud_security_group.default.id}"]
  instance_name        = "web"
}

# Create security group
resource "alicloud_security_group" "default" {
  name        = "default"
  provider    = "alicloud"
  description = "default"
}
```

## Authentication

The Alicloud provider offers a flexible means of providing credentials for authentication.
The following methods are supported, in this order, and explained below:

- Static credentials
- Environment variables

### Static credentials ###

Static credentials can be provided by adding an `access_key` `secret_key` and `region` in-line in the
alicloud provider block:

Usage:

```hcl
provider "alicloud" {
  access_key = "${var.access_key}"
  secret_key = "${var.secret_key}"
  region     = "${var.region}"
}
```


###Environment variables

You can provide your credentials via `ALICLOUD_ACCESS_KEY` and `ALICLOUD_SECRET_KEY`,
environment variables, representing your Alicloud Access Key and Secret Key, respectively.
`ALICLOUD_REGION` is also used, if applicable:

```hcl
provider "alicloud" {}
```

Usage:

```shell
$ export ALICLOUD_ACCESS_KEY="anaccesskey"
$ export ALICLOUD_SECRET_KEY="asecretkey"
$ export ALICLOUD_REGION="cn-beijing"
$ terraform plan
```


## Argument Reference

The following arguments are supported:

* `access_key` - (Optional) This is the Alicloud access key. It must be provided, but
  it can also be sourced from the `ALICLOUD_ACCESS_KEY` environment variable.

* `secret_key` - (Optional) This is the Alicloud secret key. It must be provided, but
  it can also be sourced from the `ALICLOUD_SECRET_KEY` environment variable.

* `region` - (Required) This is the Alicloud region. It must be provided, but
  it can also be sourced from the `ALICLOUD_REGION` environment variables.


## Testing

Credentials must be provided via the `ALICLOUD_ACCESS_KEY`, and `ALICLOUD_SECRET_KEY` environment variables in order to run acceptance tests.
