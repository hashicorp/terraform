---
layout: "ibmcloud"
page_title: "Provider: IBM Cloud"
sidebar_current: "docs-ibmcloud-index"
description: |-
  The IBM Cloud provider is used to interact with IBM Cloud resources.
---

# IBM Cloud Provider

The IBM Cloud provider is used to manage IBM Cloud resources. 
The provider needs to be configured with the proper credentials 
before it can be used.

Use the navigation to the left to read about the available resources.


## Example Usage

Here is an example that will setup the following:

+ An SSH key resource.
+ A virtual server resource that uses an existing SSH key.
+ A virtual server resource using an existing SSH key and a Terraform managed SSH key (created as `test_key_1` in the example below).

Add the below to a file called `sl.tf` and run the `terraform` command from the same directory:

```hcl
# Configure the IBM Cloud Provider
provider "ibmcloud" {
    ibmid = "${var.ibmcloud_bmx_user}"
    password = "${var.ibmcloud_bmx_pass}"
    softlayer_username = "${var.ibmcloud_sl_user}"
    softlayer_api_key = "${var.ibmcloud_sl_api_key}"
}

# This will create a new SSH key that will show up under the \
# Devices>Manage>SSH Keys in the SoftLayer console.
resource "ibmcloud_infra_ssh_key" "test_key_1" {
    name = "test_key_1"
    public_key = "${file(\"~/.ssh/id_rsa_test_key_1.pub\")}"
    # Windows Example:
    # public_key = "${file(\"C:\ssh\keys\path\id_rsa_test_key_1.pub\")}"
}

# Virtual Server created with existing SSH Key already in SoftLayer \
# inventory and not created using this Terraform template.
resource "ibmcloud_infra_virtual_guest" "my_server_1" {
    hostname = "host-a.example.com"
    domain = "example.com"
    ssh_key_ids = [123456]
    os_reference_code = "DEBIAN_7_64"
    datacenter = "ams01"
    network_speed = 10
    cores = 1
    memory = 1024
}

# Virtual Server created with a mix of previously existing and \
# Terraform created/managed resources.
resource "ibmcloud_infra_virtual_guest" "my_server_2" {
    hostname = "host-b.example.com"
    domain = "example.com"
    ssh_keys = [123456, "${softlayer_ssh_key.test_key_1.id}"]
    os_reference_code = "CENTOS_6_64"
    datacenter = "ams01"
    network_speed = 10
    cores = 1
    memory = 1024
}
```

## Authentication

The IBM Cloud provider offers a flexible means of providing credentials for
authentication. The following methods are supported, in this order, and
explained below:

- Static credentials
- Environment variables

### Static credentials ###

Static credentials can be provided by adding an `ibmid`, `password`, `softlayer_username` and `softlayer_api_key` in-line in the IBM Cloud provider block:

Usage:

```
provider "ibmcloud" {
    ibmid = ""
    password = ""
    softlayer_username = ""
    softlayer_api_key = ""
}
```


### Environment variables

You can provide your credentials via the `IBMID`,`IBMID_PASSWORD`, `SL_USERNAME` 
and`SL_API_KEY`environment variables, representing your IBM ID, IBM ID password,
SoftLayer username and SoftLayer API Key respectively.  

```
provider "ibmcloud" {}
```

Usage:

```
$ export IBMID="ibmid"
$ export IBMID_PASSWORD="password"
$ export SL_USERNAME="sl_user"
$ export SL_PASSWORD="sl_password"
$ terraform plan
```

## Argument Reference

The following arguments are supported in the `provider` block:

* `ibmid` - (Optional) This is the IBM ID. It must be provided, but
  it can also be sourced from the `IBMID`  environment variable.

* `password` - (Optional) This is the password for the IBM ID. It must be provided, but
  it can also be sourced from the `IBMID_PASSWORD` environment variable.

* `timeout` - (Optional) This is the timeout in seconds for the Bluemix API Key.
  It can also be sourced from the `BM_TIMEOUT` or `BLUEMIX_TIMEOUT` environment variable.
  The former variable has higher precedence. Default value is `60 seconds`.

* `region` - (Required) This is the Bluemix region. It must be provided, but
  it can also be sourced from the `BM_REGION` or `BLUEMIX_REGION` environment variable.
 The former variable has higher precedence. 

* `softlayer_username` - (Optional) This is the SoftLayer user name. It must be provided, but
  it can also be sourced from the `SL_USERNAME`  or `SOFTLAYER_USERNAME` environment variable.
  The former variable has higher precedence.

* `softlayer_api_key` - (Optional) This is the SoftLayer user API Key. It must be provided, but
  it can also be sourced from the `SL_API_KEY`  or `SOFTLAYER_API_KEY` environment variable.
  The former variable has higher precedence.
  
* `softlayer_endpoint_url` - (Optional) This is the SoftLayer user API Key. It must be provided, but
  it can also be sourced from the `SL_ENDPOINT_URL`  or `SOFTLAYER_ENDPOINT_URL` environment variable.
  The former variable has higher precedence. 

* `softlayer_timeout` - (Optional) This is the timeout in seconds for the SoftLayer API Key.
  It can also be sourced from the `SL_TIMEOUT`  or `SOFTLAYER_TIMEOUT` environment variable.
  The former variable has higher precedence. Default value is `60 seconds`.
