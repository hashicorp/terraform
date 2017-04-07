---
layout: "docs"
page_title: "Creating Modules"
sidebar_current: "docs-modules-create"
description: How to create modules.
---

# Creating Modules

Creating modules in Terraform is easy. You may want to do this to better organize your code, to make a reusable component, or just to learn more about Terraform. For any reason, if you already know the basics of Terraform, then creating a module is a piece of cake.

Modules in Terraform are folders with Terraform files. In fact, when you run `terraform apply`, the current working directory holding
the Terraform files you're applying comprise what is called the _root module_. This itself is a valid module.

Therefore, you can enter the source of any module, satisfy any required variables, run `terraform apply`, and expect it to work.

## An Example Module

Within a folder containing Terraform configurations, create a subfolder called `child`. In this subfolder, make one empty `main.tf` file. Then, back in the root folder containing the `child` folder, add this to one of your Terraform configuration files:

```hcl
module "child" {
  source = "./child"
}
```

You've now created your first module! You can now add resources to the `child` module.

**Note:** Prior to running the above, you'll have to run [the get command](/docs/commands/get.html) for Terraform to sync
your modules. This should be instant since the module is a local path.

## Inputs/Outputs

To make modules more useful than simple isolated containers of Terraform configurations, modules can be configured and also have outputs that can be consumed by your Terraform configuration.

Inputs of a module are [variables](/docs/configuration/variables.html) and outputs are [outputs](/docs/configuration/outputs.html). There is no special syntax to define these, they're defined just like any other variables or outputs. You can think about these variables and outputs as the API interface to your module.

Let's add a variable and an output to our `child` module.

```hcl
variable "memory" {}

output "received" {
  value = "${var.memory}"
}
```

This will create a required variable, `memory`, and then an output, `received`, that will be the value of the `memory` variable.

You can then configure the module and use the output like so:

```hcl
module "child" {
  source = "./child"

  memory = "1G"
}

output "child_memory" {
  value = "${module.child.received}"
}
```

If you now run `terraform apply`, you see how this works.

## Paths and Embedded Files

It is sometimes useful to embed files within the module that aren't Terraform configuration files, such as a script to provision a resource or a file to upload.

In these cases, you can't use a relative path, since paths in Terraform are generally relative to the working directory from which Terraform was executed. Instead, you want to use a module-relative path. To do this, you should use the [path interpolated variables](/docs/configuration/interpolation.html).

```hcl
resource "aws_instance" "server" {
  # ...

  provisioner "remote-exec" {
    script = "${path.module}/script.sh"
  }
}
```

Here we use `${path.module}` to get a module-relative path.

## Nested Modules

You can nest a module within another module. This module will be hidden from your root configuration, so you'll have to re-expose any
variables and outputs you require.

The [get command](/docs/commands/get.html) will automatically get all nested modules.

You don't have to worry about conflicting versions of modules, since Terraform builds isolated subtrees of all dependencies. For example, one module might use version 1.0 of module `foo` and another module might use version 2.0, and this will all work fine within Terraform since the modules are created separately.
