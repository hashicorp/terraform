---
layout: "docs"
page_title: "Provisioner: local-exec"
sidebar_current: "docs-provisioners-local"
description: |-
  The `local-exec` provisioner invokes a local executable after a resource is created. This invokes a process on the machine running Terraform, not on the resource. See the `remote-exec` provisioner to run commands on the resource.
---

# local-exec Provisioner

The `local-exec` provisioner invokes a local executable after a resource
is created. This invokes a process on the machine running Terraform, not on
the resource. See the `remote-exec` [provisioner](/docs/provisioners/remote-exec.html)
to run commands on the resource.

## Example usage

```
# Join the newly created machine to our Consul cluster
resource "aws_instance" "web" {
    ...
    provisioner "local-exec" {
        command = "echo ${aws_instance.web.private_ip} >> private_ips.txt"
    }
}
```

## Argument Reference

The following arguments are supported:

* `command` - (Required) This is the command to execute. It can be provided
  as a relative path to the current working directory or as an absolute path.
  It is evaluated in a shell, and can use environment variables or Terraform
  variables.

