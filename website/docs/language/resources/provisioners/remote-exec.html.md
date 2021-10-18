---
layout: "language"
page_title: "Provisioner: remote-exec"
sidebar_current: "docs-provisioners-remote"
description: |-
  The `remote-exec` provisioner invokes a script on a remote resource after it is created. This can be used to run a configuration management tool, bootstrap into a cluster, etc. To invoke a local process, see the `local-exec` provisioner instead. The `remote-exec` provisioner supports both `ssh` and `winrm` type connections.
---

# remote-exec Provisioner

The `remote-exec` provisioner invokes a script on a remote resource after it
is created. This can be used to run a configuration management tool, bootstrap
into a cluster, etc. To invoke a local process, see the `local-exec`
[provisioner](/docs/language/resources/provisioners/local-exec.html) instead. The `remote-exec`
provisioner requires a [connection](/docs/language/resources/provisioners/connection.html)
and supports both `ssh` and `winrm`.

-> **Note:** Provisioners should only be used as a last resort. For most
common situations there are better alternatives. For more information, see
[the main Provisioners page](./).

## Example usage

```hcl
resource "aws_instance" "web" {
  # ...

  # Establishes connection to be used by all 
  # generic remote provisioners (i.e. file/remote-exec)
  connection {
    type     = "ssh"
    user     = "root"
    password = var.root_password
    host     = self.public_ip
  }

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

-> **Note:** Since `inline` is implemented by concatenating commands into a script, [`on_failure`](/docs/language/resources/provisioners/syntax.html#failure-behavior) applies only to the final command in the list. In particular, with `on_failure = fail` (the default behaviour) earlier commands will be allowed to fail, and later commands will also execute. If this behaviour is not desired, consider using `"set -o errexit"` as the first command.

## Script Arguments

You cannot pass any arguments to scripts using the `script` or
`scripts` arguments to this provisioner. If you want to specify arguments,
upload the script with the
[file provisioner](/docs/language/resources/provisioners/file.html)
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
