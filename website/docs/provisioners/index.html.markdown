---
layout: "docs"
page_title: "Provisioners"
sidebar_current: "docs-provisioners"
description: |-
  Provisioners are used to execute scripts on a local or remote machine as part of resource creation or destruction.
---

# Provisioners

Provisioners can be used to model specific actions on the local machine or on
a remote machine in order to prepare servers or other infrastructure objects
for service.

## Provisioners are a Last Resort

Terraform includes the concept of provisioners as a measure of pragmatism,
knowing that there will always be certain behaviors that can't be directly
represented in Terraform's declarative model.

However, they also add a considerable amount of complexity and uncertainty to
Terraform usage. Firstly, Terraform cannot model the actions of provisioners
as part of a plan because they can in principle take any action. Secondly,
successful use of provisioners requires coordinating many more details than
Terraform usage usually requires: direct network access to your servers,
issuing Terraform credentials to log in, making sure that all of the necessary
external software is installed, etc.

The following sections describe some situations which can be solved with
provisioners in principle, but where better solutions are also available. We do
not recommend using provisioners for any of the use-cases described in the
following sections.

Even if your specific use-case is not described in the following sections, we
still recommend attempting to solve it using other techniques first, and use
provisioners only if there is no other option.

### Passing data into virtual machines and other compute resources

When deploying virtual machines or other similar compute resources, we often
need to pass in data about other related infrastructure that the software on
that server will need to do its job.

The various provisioners that interact with remote servers over SSH or WinRM
can potentially be used to pass such data by logging in to the server and
providing it directly, but most cloud computing platforms provide mechanisms
to pass data to instances at the time of their creation such that the data
is immediately available on system boot. For example:

* Alibaba Cloud: `user_data` on
  [`alicloud_instance`](/docs/providers/alicloud/r/instance.html)
  or [`alicloud_launch_template`](/docs/providers/alicloud/r/launch_template.html).
* Amazon EC2: `user_data` or `user_data_base64` on
  [`aws_instance`](/docs/providers/aws/r/instance.html),
  [`aws_launch_template`](/docs/providers/aws/r/launch_template.html),
  and [`aws_launch_configuration`](/docs/providers/aws/r/launch_configuration.html).
* Amazon Lightsail: `user_data` on
  [`aws_lightsail_instance`](/docs/providers/aws/r/lightsail_instance.html).
* Microsoft Azure: `custom_data` on
  [`azurerm_virtual_machine`](/docs/providers/azurerm/r/virtual_machine.html)
  or [`azurerm_virtual_machine_scale_set`](/docs/providers/azurerm/r/virtual_machine_scale_set.html).
* Google Cloud Platform: `metadata` on
  [`google_compute_instance`](/docs/providers/google/r/compute_instance.html)
  or [`google_compute_instance_group`](/docs/providers/google/r/compute_instance_group.html).
* Oracle Cloud Infrastructure: `metadata` or `extended_metadata` on
  [`oci_core_instance`](/docs/providers/oci/r/core_instance.html)
  or [`oci_core_instance_configuration`](/docs/providers/oci/r/core_instance_configuration.html).
* VMWare vSphere: Attach a virtual CDROM to
  [`vsphere_virtual_machine`](/docs/providers/vsphere/r/virtual_machine.html)
  using the `cdrom` block, containing a file called `user-data.txt`.

Many official Linux distribution disk images include software called
[cloud-init](https://cloudinit.readthedocs.io/en/latest/) that can automatically
process in various ways data passed via the means described above, allowing
you to run arbitrary scripts and do basic system configuration immediately
during the boot process and without the need to access the machine over SSH.

If you are building custom machine images, you can make use of the "user data"
or "metadata" passed by the above means in whatever way makes sense to your
application, by referring to your vendor's documentation on how to access the
data at runtime.

This approach is _required_ if you intend to use any mechanism in your cloud
provider for automatically launching and destroying servers in a group,
because in that case individual servers will launch unattended while Terraform
is not around to provision them.

Even if you're deploying individual servers directly with Terraform, passing
data this way will allow faster boot times and simplify deployment by avoiding
the need for direct network access from Terraform to the new server and for
remote access credentials to be provided.

### Running configuration management software

As a convenience to users who are forced to use generic operating system
distribution images, Terraform includes a number of specialized provisioners
for launching specific configuration management products.

We strongly recommend not using these, and instead running system configuration
steps during a custom image build process. For example,
[HashiCorp Packer](https://packer.io/) offers a similar complement of
configuration management provisioners and can run their installation steps
during a separate build process, before creating a system disk image that you
can deploy many times.

If you are using configuration management software that has a centralized server
component, you will need to delay the _registration_ step until the final
system is booted from your custom image. To achieve that, use one of the
mechanisms described above to pass the necessary information into each instance
so that it can register itself with the configuration management server
immediately on boot, without the need to accept commands from Terraform over
SSH or WinRM.

### First-class Terraform provider functionality may be available

It is technically possible to use the `local-exec` provisioner to run the CLI
for your target system in order to create, update, or otherwise interact with
remote objects in that system.

If you are trying to use a new feature of the remote system that isn't yet
supported in its Terraform provider, that might be the only option. However,
if there _is_ provider support for the feature you intend to use, prefer to
use that provider functionality rather than a provisioner so that Terraform
can be fully aware of the object and properly manage ongoing changes to it.

Even if the functionality you need is not available in a provider today, we
suggest to consider `local-exec` usage a temporary workaround and to also
open an issue in the relevant provider's repository to discuss adding
first-class provider support. Provider development teams often prioritize
features based on interest, so opening an issue is a way to record your
interest in the feature.

Provisioners are used to execute scripts on a local or remote machine
as part of resource creation or destruction. Provisioners can be used to
bootstrap a resource, cleanup before destroy, run configuration management, etc.

## How to use Provisioners

-> **Note:** Provisioners should only be used as a last resort. For most
common situations there are better alternatives. For more information, see
the sections above.

If you are certain that provisioners are the best way to solve your problem
after considering the advice in the sections above, you can add a
`provisioner` block inside the `resource` block for your compute instance.

```hcl
resource "aws_instance" "web" {
  # ...

  provisioner "local-exec" {
    command = "echo The server's IP address is ${self.private_ip}"
  }
}
```

The `local-exec` provisioner requires no other configuration, but most other
provisioners must connect to the remote system using SSH or WinRM.
You must write [a `connection` block](./connection.html) so that Terraform
will know how to communicate with the server.

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
    command    = "echo The server's IP address is ${self.private_ip}"
    on_failure = "continue"
  }
}
```
