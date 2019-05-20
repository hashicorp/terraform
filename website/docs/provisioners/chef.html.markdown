---
layout: "docs"
page_title: "Provisioner: chef"
sidebar_current: "docs-provisioners-chef"
description: |-
  The `chef` provisioner installs, configures and runs the Chef client on a resource.
---

# Chef Provisioner

The `chef` provisioner installs, configures and runs the Chef Client on a remote
resource. The `chef` provisioner supports both `ssh` and `winrm` type
[connections](/docs/provisioners/connection.html).

## Requirements

The `chef` provisioner has some prerequisites for specific connection types:

* For `ssh` type connections, `cURL` must be available on the remote host.
* For `winrm` connections, `PowerShell 2.0` must be available on the remote host.

[Chef end user license agreement](https://www.chef.io/end-user-license-agreement/) must be accepted by setting `chef_license` to `accept` in `client_options` argument unless you are installing an old version of Chef client.

Without these prerequisites, your provisioning execution will fail.

## Example usage

```hcl
resource "aws_instance" "web" {
  # ...

  provisioner "chef" {
    attributes_json = <<EOF
      {
        "key": "value",
        "app": {
          "cluster1": {
            "nodes": [
              "webserver1",
              "webserver2"
            ]
          }
        }
      }
    EOF

    environment     = "_default"
    client_options  = ["chef_license 'accept'"]
    run_list        = ["cookbook::recipe"]
    node_name       = "webserver1"
    secret_key      = "${file("../encrypted_data_bag_secret")}"
    server_url      = "https://chef.company.com/organizations/org1"
    recreate_client = true
    user_name       = "bork"
    user_key        = "${file("../bork.pem")}"
    version         = "12.4.1"
    # If you have a self signed cert on your chef server change this to :verify_none
    ssl_verify_mode = ":verify_peer"
  }
}
```

## Argument Reference

The following arguments are supported:

* `attributes_json (string)` - (Optional) A raw JSON string with initial node attributes
  for the new node. These can also be loaded from a file on disk using
  [the `file` function](/docs/configuration/functions/file.html).

* `channel (string)` - (Optional) The Chef Client release channel to install from. If not
  set, the `stable` channel will be used.

* `client_options (array)` - (Optional) A list of optional Chef Client configuration
  options. See the [Chef Client ](https://docs.chef.io/config_rb_client.html) documentation
  for all available options.

* `disable_reporting (boolean)` - (Optional) If `true` the Chef Client will not try to send
  reporting data (used by Chef Reporting) to the Chef Server (defaults to `false`).

* `environment (string)` - (Optional) The Chef environment the new node will be joining
  (defaults to `_default`).

* `fetch_chef_certificates (boolean)` (Optional) If `true` the SSL certificates configured
  on your Chef Server will be fetched and trusted. See the knife [ssl_fetch](https://docs.chef.io/knife_ssl_fetch.html)
  documentation for more details.

* `log_to_file (boolean)` - (Optional) If `true`, the output of the initial Chef Client run
  will be logged to a local file instead of the console. The file will be created in a
  subdirectory called `logfiles` created in your current directory. The filename will be
  the `node_name` of the new node.

* `use_policyfile (boolean)` - (Optional) If `true`, use the policy files to bootstrap the
  node. Setting `policy_group` and `policy_name` are required if this is `true`. (defaults to
  `false`).

* `policy_group (string)` - (Optional) The name of a policy group that exists on the Chef
  server. Required if `use_policyfile` is set; `policy_name` must also be specified.

* `policy_name (string)` - (Optional) The name of a policy, as identified by the `name`
  setting in a Policyfile.rb file. Required if `use_policyfile` is set; `policy_group`
  must also be specified.

* `http_proxy (string)` - (Optional) The proxy server for Chef Client HTTP connections.

* `https_proxy (string)` - (Optional) The proxy server for Chef Client HTTPS connections.

* `named_run_list (string)` - (Optional) The name of an alternate run-list to invoke during the
  initial Chef Client run. The run-list must already exist in the Policyfile that defines
  `policy_name`. Only applies when `use_policyfile` is `true`.

* `no_proxy (array)` - (Optional) A list of URLs that should bypass the proxy.

* `node_name (string)` - (Required) The name of the node to register with the Chef Server.

* `ohai_hints (array)` - (Optional) A list with
  [Ohai hints](https://docs.chef.io/ohai.html#hints) to upload to the node.

* `os_type (string)` - (Optional) The OS type of the node. Valid options are: `linux` and
  `windows`. If not supplied, the connection type will be used to determine the OS type (`ssh`
  will assume `linux` and `winrm` will assume `windows`).

* `prevent_sudo (boolean)` - (Optional) Prevent the use of the `sudo` command while installing, configuring
  and running the initial Chef Client run. This option is only used with `ssh` type
  [connections](/docs/provisioners/connection.html).

* `recreate_client (boolean)` - (Optional) If `true`, first delete any existing Chef Node and
  Client before registering the new Chef Client.

* `run_list (array)` - (Optional) A list with recipes that will be invoked during the initial
  Chef Client run. The run-list will also be saved to the Chef Server after a successful
  initial run. Required if `use_policyfile` is `false`; ignored when `use_policyfile` is `true`
  (see `named_run_list` to specify a run-list defined in a Policyfile).

* `secret_key (string)` - (Optional) The contents of the secret key that is used
  by the Chef Client to decrypt data bags on the Chef Server. The key will be uploaded to the remote
  machine. This can also be loaded from a file on disk using
  [the `file` function](/docs/configuration/functions/file.html).

* `server_url (string)` - (Required) The URL to the Chef server. This includes the path to
  the organization. See the example.

* `skip_install (boolean)` - (Optional) Skip the installation of Chef Client on the remote
  machine. This assumes Chef Client is already installed when you run the `chef`
  provisioner.

* `skip_register (boolean)` - (Optional) Skip the registration of Chef Client on the remote
  machine. This assumes Chef Client is already registered and the private key (`client.pem`)
  is available in the default Chef configuration directory when you run the `chef`
  provisioner.

* `ssl_verify_mode (string)` - (Optional) Used to set the verify mode for Chef Client HTTPS
  requests. The options are `:verify_none`, or `:verify_peer` which is default.

* `user_name (string)` - (Required) The name of an existing Chef user to register
  the new Chef Client and optionally configure Chef Vaults.

* `user_key (string)` - (Required) The contents of the user key that will be used to
  authenticate with the Chef Server. This can also be loaded from a file on disk using
  [the `file` function](/docs/configuration/functions/file.html).

* `vault_json (string)` - (Optional) A raw JSON string with Chef Vaults and Items to which the new node
  should have access. These can also be loaded from a file on disk using
  [the `file` function](/docs/configuration/functions/file.html).

* `version (string)` - (Optional) The Chef Client version to install on the remote machine.
  If not set, the latest available version will be installed.
