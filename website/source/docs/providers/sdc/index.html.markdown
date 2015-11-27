---
layout: "sdc"
page_title: "Provider: SDC/CloudAPI"
sidebar_current: "docs-sdc-index"
description: |-
  The Smart Data Center/CloudAPI (SDC) provider is used to interact with the resources supported by Smart Data Center or the Joyent Public Cloud. The provider needs to be configured with the proper credentials before it can be used.
---

# SDC/CloudAPI Provider

The Smart Data Center/CloudAPI (SDC) provider is used to interact with the
resources supported by Smart Data Center or the Joyent Public Cloud. The
provider needs to be configured with the proper credentials before it can be
used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the SDC Provider
provider "sdc" {
    sdc_key_name = "${var.sdc_key_name}"
}

# Create an instance
resource "sdc_instance" "foo" {
    ...
}
```

## Argument Reference

The following arguments are supported:

* `sdc_key_name` - (Optional) This is the openssh key name that should be used
  to sign the CloudAPI requests.

In addition to the above parameters the provider is using the same
environmental variables as all the `sdc-*` cli tools:

* `SDC_ACCOUNT` - (Required) The account name to use for authentication.
* `SDC_URL` - (Required) The CloudAPI endpoint to authenticate against.
* `SDC_KEY_ID` - (Required) The key to use for authentication.
* `MANTA_KEY_ID` - (Required) The key to use for authentication against the
  Manta block storage system.

## Testing

The acceptance tests are configured to use the networks present in the AMS-1
datacenter.

`SDC_URL` has to be set to `https://eu-ams-1.api.joyentcloud.com`.

