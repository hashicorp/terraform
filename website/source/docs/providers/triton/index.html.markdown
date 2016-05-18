---
layout: "triton"
page_title: "Provider: Triton"
sidebar_current: "docs-triton-index"
description: |-
  Used to provision infrastructure in Joyent's Triton public or on-premise clouds.
---

# Triton Provider

The Triton provider is used to interact with resources in Joyent's Triton cloud. It is compatible with both public- and on-premise installations of Triton. The provider needs to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
provider "triton" {
    account      = "AccountName"
    key_material = "${file("~/.ssh/id_rsa")}"
    key_id       = "25:d4:a9:fe:ef:e6:c0:bf:b4:4b:4b:d4:a8:8f:01:0f"

    # If using a private installation of Triton, specify the URL
    url = "https://us-west-1.api.joyentcloud.com"
}
```

## Argument Reference

The following arguments are supported in the `provider` block:

* `account` - (Required) This is the name of the Triton account. It can also be provided via the `SDC_ACCOUNT` environment variable.
* `key_material` - (Required) This is the private key of an SSH key associated with the Triton account to be used.
* `key_id` - (Required) This is the fingerprint of the public key matching the key specified in `key_path`. It can be obtained via the command `ssh-keygen -l -E md5 -f /path/to/key`
* `url` - (Optional) This is the URL to the Triton API endpoint. It is required if using a private installation of Triton. The default is to use the Joyent public cloud.
