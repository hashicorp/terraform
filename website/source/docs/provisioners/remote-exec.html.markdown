---
layout: "docs"
page_title: "Provisioner: remote-exec"
sidebar_current: "docs-provisioners-remote"
description: |-
  The `remote-exec` provisioner invokes a script on a remote resource after it is created. This can be used to run a configuration management tool, bootstrap into a cluster, etc. To invoke a local process, see the `local-exec` provisioner instead. The `remote-exec` provisioner supports both `ssh` and `winrm` type connections.
---

# remote-exec Provisioner

The `remote-exec` provisioner invokes a script on a remote resource after it
is created. This can be used to run a configuration management tool, bootstrap
into a cluster, etc. To invoke a local process, see the `local-exec`
[provisioner](/docs/provisioners/local-exec.html) instead. The `remote-exec`
provisioner supports both `ssh` and `winrm` type [connections](/docs/provisioners/connection.html).


## Example usage

```hcl
resource "aws_instance" "web" {
  # ...

  provisioner "remote-exec" {
    inline = [
      "puppet apply",
      "consul join ${aws_instance.web.private_ip}",
    ]
  }
}
```

## Argument Reference

The following arguments are supported:

* `inline` - This is a list of command strings. They are executed in the order
  they are provided. This cannot be provided with `script` or `scripts`.

* `script` - This is a path (relative or absolute) to a local script that will
  be copied to the remote resource and then executed. This cannot be provided
  with `inline` or `scripts`.

* `scripts` - This is a list of paths (relative or absolute) to local scripts
  that will be copied to the remote resource and then executed. They are executed
  in the order they are provided. This cannot be provided with `inline` or `script`.

## Script Arguments

You cannot pass any arguments to scripts using the `script` or
`scripts` arguments to this provisioner. If you want to specify arguments,
upload the script with the
[file provisioner](/docs/provisioners/file.html)
and then use `inline` to call it. Example:

```hcl
resource "aws_instance" "web" {
  # ...

  provisioner "file" {
    source      = "script.sh"
    destination = "/tmp/script.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x /tmp/script.sh",
      "/tmp/script.sh args",
    ]
  }
}
```
