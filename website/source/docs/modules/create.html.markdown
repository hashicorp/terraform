---
layout: "docs"
page_title: "Creating Modules"
sidebar_current: "docs-modules-create"
description: |-
  Creating modules in Terraform is easy. You may want to do this to better organize your code, to make a reusable component, or just to learn more about Terraform. For any reason, if you already know the basics of Terraform, creating a module is a piece of cake.
---

# Creating Modules

Creating modules in Terraform is easy. You may want to do this to better
organize your code, to make a reusable component, or just to learn more about
Terraform. For any reason, if you already know the basics of Terraform,
creating a module is a piece of cake.

Modules in Terraform are just folders with Terraform files. In fact,
when you run `terraform apply`, the current working directory holding
the Terraform files you're applying comprise what is called the
_root module_. It itself is a valid module.

Therefore, you can enter the source of any module, run `terraform apply`,
and expect it to work (assuming you satisfy the required variables, if any).

## An Example

Within a folder containing Terraform configurations, create a subfolder
"child". In this subfolder, make one empty "main.tf" file. Then, back in
the root folder containing the "child" folder, add this to one of the
Terraform files:

```
module "child" {
	source = "./child"
}
```

This will work. You've created your first module! You can add resources
to the child module to see how that interaction works.

Note: Prior to running the above, you'll have to run
[the get command](/docs/commands/get.html) for Terraform to sync
your modules. This should be instant since the module is just a local path.

## Inputs/Outputs

To make modules more useful than simple isolated containers of Terraform
configurations, modules can be configured and also have outputs that can be
consumed by the configuration using the module.

Inputs of a module are [variables](/docs/configuration/variables.html)
and outputs are [outputs](/docs/configuration/outputs.html). There is no
special syntax to define these, they're defined just like any other
variables or outputs.

In the "child" module we created above, add the following:

```
variable "memory" {}

output "received" {
	value = "${var.memory}"
}
```

This will create a required variable "memory" and then an output "received"
that will simply be the value of the memory variable.

You can then configure the module and use the output like so:

```
module "child" {
	source = "./child"

	memory = "1G"
}

output "child_memory" {
	value = "${module.child.received}"
}
```

If you run `apply`, you'll again see that this works.

And that is all there is to it. Variables and outputs are used to configure
modules and provide results. Resources within a module are isolated,
and the whole thing is managed as a single unit.

## Paths and Embedded Files

It is sometimes useful to embed files within the module that aren't
Terraform configuration files, such as a script to provision a resource
or a file to upload.

In these cases, you can't use a relative path, since paths in Terraform
are generally relative to the working directory that Terraform was executed
from. Instead, you want to use a module-relative path. To do this, use
the [path interpolated variables](/docs/configuration/interpolation.html).

An example is shown below:

```
resource "aws_instance" "server" {
	...

	provisioner "remote-exec" {
		script = "${path.module}/script.sh"
	}
}
```

In the above, we use `${path.module}` to get a module-relative path. This
is usually what you'll want in any case.

## Nested Modules

You can use a module within a module just like you would anywhere else.
This module will be hidden from the root user, so you'll have re-expose any
variables if you need to, as well as outputs.

The [get command](/docs/commands/get.html) will automatically get all
nested modules as well.

You don't have to worry about conflicting versions of modules, since
Terraform builds isolated subtrees of all dependencies. For example,
one module might use version 1.0 of module "foo" and another module
might use version 2.0 of module "foo", and this would all work fine
within Terraform since the modules are created separately.
