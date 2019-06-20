---
layout: "docs"
page_title: "Provisioner: puppet"
sidebar_current: "docs-provisioners-puppet"
description: |-
  The `puppet` provisioner installs, configures and runs the Puppet agent on a resource.
---

# Puppet Provisioner

The `puppet` provisioner installs, configures and runs the Puppet agent on a
remote resource. The `puppet` provisioner supports both `ssh` and `winrm` type
[connections](/docs/provisioners/connection.html).

## Requirements

The `puppet` provisioner has some prerequisites for specific connection types:

* For `ssh` type connections, `cURL` must be available on the remote host.
* For `winrm` connections, `PowerShell 2.0` must be available on the remote host.

Without these prerequisites, your provisioning execution will fail.

Additionally, the `puppet` provisioner requires
[Bolt](https://puppet.com/products/bolt) to be installed on your workstation
with the following [modules
installed](https://puppet.com/docs/bolt/latest/bolt_installing_modules.html#install-modules)

* `danieldreier/autosign`
* `puppetlabs/puppet_agent`

## Example usage

```hcl
resource "aws_instance" "web" {
  # ...

  provisioner "puppet" {
    server             = aws_instance.puppetmaster.public_dns
    server_user        = "ubuntu"
    extension_requests = {
      pp_role = "webserver"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `server (string)` - (Required) The FQDN of the Puppet master that the agent
  is to connect to.

* `server_user (string)` - (Optional) The user that Bolt should connect to the
  server as (defaults to `root`).

* `os_type (string)` - (Optional) The OS type of the resource. Valid options
  are: `linux` and `windows`. If not supplied, the connection type will be used
  to determine the OS type (`ssh` will assume `linux` and `winrm` will assume
  `windows`).

* `use_sudo (boolean)` - (Optional) If `true`, commands run on the resource
  will have their privileges elevated with sudo (defaults to `true` when the OS
  type is `linux` and `false` when the OS type is `windows`).

* `autosign (boolean)` - (Optional) Set to `true` if the Puppet master is using an autosigner such as
  [Daniel Dreier's policy-based autosigning
  tool](https://danieldreier.github.io/autosign). If `false` new agent certificate requests will have to be signed manually (defaults to `true`).

* `open_source (boolean)` - (Optional) If `true` the provisioner uses an open source Puppet compatible agent install method (push via the Bolt agent install task). If `false` the simplified Puppet Enterprise installer will pull the agent from the Puppet master (defaults to `true`).

* `certname (string)` - (Optional) The Subject CN used when requesting
  a certificate from the Puppet master CA (defaults to the FQDN of the
  resource).

* `extension_request (map)` - (Optional) A map of [extension 
  requests](https://puppet.com/docs/puppet/latest/ssl_attributes_extensions.html#concept-932)
  to be embedded in the certificate signing request before it is sent to the
  Puppet master CA and then transferred to the final certificate when the CSR
  is signed. These become available during Puppet agent runs as [trusted facts](https://puppet.com/docs/puppet/latest/lang_facts_and_builtin_vars.html#trusted-facts). Friendly names for common extensions such as pp_role and pp_environment have [been predefined](https://puppet.com/docs/puppet/latest/lang_facts_and_builtin_vars.html#trusted-facts).

* `custom_attributes (map)` - (Optional) A map of [custom
  attributes](https://puppet.com/docs/puppet/latest/ssl_attributes_extensions.html#concept-5488)
  to be embedded in the certificate signing request before it is sent to the
  Puppet master CA.

* `environment (string)` - (Optional) The name of the Puppet environment that the
  Puppet agent will be running in (defaults to `production`).

* `bolt_timeout (string)` - (Optional) The timeout to wait for Bolt tasks to
  complete. This should be specified as a string like `30s` or `5m` (defaults
  to `5m` - 5 minutes).
