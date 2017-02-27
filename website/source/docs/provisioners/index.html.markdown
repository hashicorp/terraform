---
layout: "docs"
page_title: "Provisioners"
sidebar_current: "docs-provisioners"
description: |-
  Provisioners are used to execute scripts on a local or remote machine as part of resource creation or destruction.
---

# Provisioners

Provisioners are used to execute scripts on a local or remote machine
as part of resource creation or destruction. Provisioners can be used to
bootstrap a resource, cleanup before destroy, run configuration management, etc.

Provisioners are added directly to any resource:

```
resource "aws_instance" "web" {
  # ...

  provisioner "local-exec" {
    command = "echo ${self.private_ip_address} > file.txt"
  }
}
```

For provisioners other than local execution, you must specify
[connection settings](/docs/provisioners/connection.html) so Terraform knows
how to communicate with the resource.

## Creation-Time Provisioners

Provisioners by default run when the resource they are defined within is
created. Creation-time provisioners are only run during _creation_, not
during updating or any other lifecycle. They are meant as a means to perform
bootstrapping of a system.

If a creation-time provisioner fails, the resource is marked as **tainted**.
A tainted resource will be planned for destruction and recreation upon the
next `terraform apply`. Terraform does this because a failed provisioner
can leave a resource in a semi-configured state. Because Terraform cannot
reason about what the provisioner does, the only way to ensure proper creation
of a resource is to recreate it. This is tainting.

You can change this behavior by setting the `on_failure` attribute,
which is covered in detail below.

## Destroy-Time Provisioners

If `when = "destroy"` is specified, the provisioner will run when the
resource it is defined within is _destroyed_.

Destroy provisioners are run before the resource is destroyed. If they
fail, Terraform will error and rerun the provisioners again on the next
`terraform apply`. Due to this behavior, care should be taken for destroy
provisioners to be safe to run multiple times.

## Multiple Provisioners

Multiple provisioners can be specified within a resource. Multiple provisioners
are executed in the order they're defined in the configuration file.

You may also mix and match creation and destruction provisioners. Only
the provisioners that are valid for a given operation will be run. Those
valid provisioners will be run in the order they're defined in the configuration
file.

Example of multiple provisioners:

```
resource "aws_instance" "web" {
  # ...

  provisioner "local-exec" {
    command = "echo first"
  }

  provisioner "local-exec" {
    command = "echo second"
  }
}
```

## Failure Behavior

By default, provisioners that fail will also cause the Terraform apply
itself to error. The `on_failure` setting can be used to change this. The
allowed values are:

  * `"continue"` - Ignore the error and continue with creation or destruction.

  * `"fail"` - Error (the default behavior). If this is a creation provisioner,
    taint the resource.

Example:

```
resource "aws_instance" "web" {
  # ...

  provisioner "local-exec" {
    command    = "echo ${self.private_ip_address} > file.txt"
    on_failure = "continue"
  }
}
```
