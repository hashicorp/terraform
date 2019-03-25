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

```hcl
resource "aws_instance" "web" {
  # ...

  provisioner "local-exec" {
    command = "echo ${self.private_ip} > file.txt"
  }
}
```

For provisioners other than local execution, you must specify
[connection settings](/docs/provisioners/connection.html) so Terraform knows
how to communicate with the resource.

## Creation-Time Provisioners

By default, provisioners run when the resource they are defined within is
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

```hcl
resource "aws_instance" "web" {
  # ...

  provisioner "local-exec" {
    when    = "destroy"
    command = "echo 'Destroy-time provisioner'"
  }
}
```

Destroy provisioners are run before the resource is destroyed. If they
fail, Terraform will error and rerun the provisioners again on the next
`terraform apply`. Due to this behavior, care should be taken for destroy
provisioners to be safe to run multiple times.

Destroy-time provisioners can only run if they remain in the configuration
at the time a resource is destroyed. If a resource block with a destroy-time
provisioner is removed entirely from the configuration, its provisioner
configurations are removed along with it and thus the destroy provisioner
won't run. To work around this, a multi-step process can be used to safely
remove a resource with a destroy-time provisioner:

* Update the resource configuration to include `count = 0`.
* Apply the configuration to destroy any existing instances of the resource, including running the destroy provisioner.
* Remove the resource block entirely from configuration, along with its `provisioner` blocks.
* Apply again, at which point no further action should be taken since the resources were already destroyed.

This limitation may be addressed in future versions of Terraform. For now,
destroy-time provisioners must be used sparingly and with care.

~> **NOTE:** A destroy-time provisioner within a resource that is tainted _will not_ run. This includes resources that are marked tainted from a failed creation-time provisioner or tainted manually using `terraform taint`.

## Multiple Provisioners

Multiple provisioners can be specified within a resource. Multiple provisioners
are executed in the order they're defined in the configuration file.

You may also mix and match creation and destruction provisioners. Only
the provisioners that are valid for a given operation will be run. Those
valid provisioners will be run in the order they're defined in the configuration
file.

Example of multiple provisioners:

```hcl
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

- `"continue"` - Ignore the error and continue with creation or destruction.

- `"fail"` - Error (the default behavior). If this is a creation provisioner,
    taint the resource.

Example:

```hcl
resource "aws_instance" "web" {
  # ...

  provisioner "local-exec" {
    command    = "echo ${self.private_ip} > file.txt"
    on_failure = "continue"
  }
}
```
