---
layout: "docs"
page_title: "Provisioner: chef"
sidebar_current: "docs-provisioners-chef"
description: |-
  The `chef` provisioner invokes a Chef Client run on a remote resource after first installing and configuring Chef Client on the remote resource. The `chef` provisioner supports both `ssh` and `winrm` type connections.
---

# Chef Provisioner

The `chef` provisioner invokes a Chef Client run on a remote resource after first installing
and configuring Chef Client on the remote resource. The `chef` provisioner supports both `ssh`
and `winrm` type [connections](/docs/provisioners/connection.html).

## Requirements

In order for the `chef` provisioner to work properly, you need either `cURL` (when using
a `ssh` type connection) or `PowerShell 2.0` (when using a `winrm` type connection) to be
available on the target machine.

## Example usage

```
# Start a initial chef run on a resource
resource "aws_instance" "web" {
    ...
    provisioner "chef"  {
        attributes {
            "key" = "value"
            "app" {
                "cluster1" {
                    "nodes" = ["webserver1", "webserver2"]
                }
            }
        }
        environment = "_default"
        run_list = ["cookbook::recipe"]
        node_name = "webserver1"
        secret_key = "${file("../encrypted_data_bag_secret")}"
        server_url = "https://chef.company.com/organizations/org1"
        validation_client_name = "chef-validator"
        validation_key = "${file("../chef-validator.pem")}"
        version = "12.4.1"
    }
}
```

## Argument Reference

The following arguments are supported:

* `attributes (map)` - (Optional) A map with initial node attributes for the new node.
  See example.

* `environment (string)` - (Optional) The Chef environment the new node will be joining
  (defaults `_default`).

* `log_to_file (boolean)` - (Optional) If true, the output of the initial Chef Client run
  will be logged to a local file instead of the console. The file will be created in a
  subdirectory called `logfiles` created in your current directory. The filename will be
  the `node_name` of the new node.

* `http_proxy (string)` - (Optional) The proxy server for Chef Client HTTP connections.

* `https_proxy (string)` - (Optional) The proxy server for Chef Client HTTPS connections.

* `no_proxy (array)` - (Optional) A list of URLs that should bypass the proxy.

* `node_name (string)` - (Required) The name of the node to register with the Chef Server.

* `ohai_hints (array)` - (Optional) A list with
  [Ohai hints](https://docs.chef.io/ohai.html#hints) to upload to the node.

* `os_type (string)` - (Optional) The OS type of the node. Valid options are: `linux` and
  `windows`. If not supplied the connection type will be used to determine the OS type (`ssh`
  will assume `linux` and `winrm` will assume `windows`).

* `prevent_sudo (boolean)` - (Optional) Prevent the use of sudo while installing, configuring
  and running the initial Chef Client run. This option is only used with `ssh` type
  [connections](/docs/provisioners/connection.html).

* `run_list (array)` - (Required) A list with recipes that will be invoked during the initial
  Chef Client run. The run-list will also be saved to the Chef Server after a successful
  initial run.

* `secret_key (string)` - (Optional) The contents of the secret key that is used
  by the client to decrypt data bags on the Chef Server. The key will be uploaded to the remote
  machine.  These can be loaded from a file on disk using the [`file()` interpolation
  function](/docs/configuration/interpolation.html#file_path_).

* `server_url (string)` - (Required) The URL to the Chef server. This includes the path to
  the organization. See the example.

* `skip_install (boolean)` - (Optional) Skip the installation of Chef Client on the remote
  machine. This assumes Chef Client is already installed when you run the `chef`
  provisioner.

* `ssl_verify_mode (string)` - (Optional) Use to set the verify mode for Chef Client HTTPS
  requests.

* `validation_client_name (string)` - (Required) The name of the validation client to use
  for the initial communication with the Chef Server.

* `validation_key (string)` - (Required) The contents of the validation key that is needed
  by the node to register itself with the Chef Server. The key will be uploaded to the remote
  machine. These can be loaded from a file on disk using the [`file()`
  interpolation function](/docs/configuration/interpolation.html#file_path_).

* `version (string)` - (Optional) The Chef Client version to install on the remote machine.
  If not set the latest available version will be installed.

These are supported for backwards compatibility and may be removed in a
future version:

* `validation_key_path (string)` - __Deprecated: please use `validation_key` instead__.
* `secret_key_path (string)` - __Deprecated: please use `secret_key` instead__.
