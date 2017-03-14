---
layout: "ibmcloud"
page_title: "Provider: IBM Cloud"
sidebar_current: "docs-ibmcloud-index"
description: |-
  The IBM Cloud provider is used to interact with IBM Cloud resources.
---

# IBM Cloud Provider

The IBM Cloud provider is used to manage IBM Cloud resources. The provider needs to be configured with the proper credentials before it can be used.

Use the navigation menu on the left to read about the available resources.


## Example Usage

```
# Define the Provider Settings
provider "ibmcloud" {
    ibmid          = "your-ibm-id"
    ibmid_password = "your-ibm-id-password"
}

# Create an SSH key.
resource "ibmcloud_infra_ssh_key" "test_key_1" {
  ...
}
```

## Authentication

The IBM Cloud provider offers a flexible means of providing credentials for authentication. The following methods are supported, in this order, and explained below:

- Static credentials
- Environment variables

### Static credentials ###

Static credentials can be provided by adding an `ibmid` and `ibmid_password` in-line in the IBM Cloud provider block:

Usage:

```
provider "ibmcloud" {
    ibmid = ""
    ibmid_password = ""
}
```


### Environment variables

You can provide your credentials via the `IBMID` and `IBMID_PASSWORD` environment variables, representing your IBM ID, IBM ID password respectively.  

Suppose you have below contents in your configuration file (.tf file)

```
# Create a virtual guest
resource "ibmcloud_infra_virtual_guest" "my_virtual_guest" {
  ...
}
```

Notice we don't specify the provider details in the above configuration. You can simply export the
required environment variables as shown below and provider will be configured without explicity
specifying them in a provider block.

```
$ export IBMID="ibmid"
$ export IBMID_PASSWORD="ibmid_password"
$ terraform plan
```

## Argument Reference

The following arguments are supported in the `provider` block:

* `ibmid` - (Optional) The IBM ID used to log into IBM services and applications. The IBM ID must be provided, but it can also be sourced from the `IBMID` environment variable.

* `ibmid_password` - (Optional) The password for the IBM ID. The password must be provided, but it can also be sourced from the `IBMID_PASSWORD` environment variable.

* `region` - (Optional) This is the Bluemix region. It can also be sourced from the `BM_REGION` or `BLUEMIX_REGION` environment variable. The former variable has higher precedence. The default value is `ng`.

* `softlayer_timeout` - (Optional) This is the timeout, expressed in seconds, for the SoftLayer API key. It can also be sourced from the `SL_TIMEOUT`  or `SOFTLAYER_TIMEOUT` environment variable. The former variable has higher precedence. The default value is `60 seconds`.

* `softlayer_account_number` - (Optional) This is the SoftLayer account number. It can also be sourced from the `SL_ACCOUNT_NUMBER`  or `SOFTLAYER_ACCOUNT_NUMBER` environment variable. The former variable has higher precedence.
Currently the provider accepts only those account numbers for which 2FA is not enabled. If the account number is not provided then the provider works with default SoftLayer Account Number and resources are created in the same default account.
