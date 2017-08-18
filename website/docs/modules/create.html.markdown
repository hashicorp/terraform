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

Modules that are created for reuse should follow the
[standard structure](#standard-structure). This structure enables tooling
such as the [Terraform Registry](/docs/registry/index.html) to inspect and
generate documentation, read examples, and more.

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

## Standard Module Structure

The standard module structure is a file and folder layout we recommend for
reusable modules. Terraform tooling is built to understand the standard
module structure and use that structure to generate documentation, index
modules for the registry, and more.

The standard module expects the structure documented below. The list may appear
long, but everything is optional except for the root module. All items are
documented in detail. Most modules don't need to do any work to follow the
standard structure.

* **Root module**. This is the **only required element** for the standard
  module structure. Terraform files must exist in the root directory of
  the module. This should be the primary entrypoint for the module and is
  expected to be opinionated. For the
  [Consul module](#)
  the root module sets up a complete Consul cluster. A lot of assumptions
  are made, however, and it is fully expected that advanced users will use
  specific nested modules to more carefully control what they want.

* **README**. The root module and any nested modules should have README
  files. This file should be named `README` or `README.md`. The latter will
  be treated as markdown. The README doesn't need to document inputs or
  outputs of the module because tooling will automatically generate this.


* **main.tf, variables.tf, outputs.tf**. These are the recommended filenames for
  a minimal module, even if they're empty. `main.tf` should be the primary
  entrypoint. For a simple module, this may be where all the resources are
  created. For a complex module, resource creation may be split into multiple
  files but all nested module usage should be in the main file. `variables.tf`
  and `outputs.tf` should contain the declarations for variables and outputs,
  respectively.

* **Variables and outputs should have descriptions.** All variables and
  outputs should have one or two sentence descriptions that explain their
  purpose. This is used for documentation. See the documentation for
  [variable configuration](/docs/configuration/variables.html) and
  [output configuration](/docs/configuration/outputs.html) for more details.

* **Nested modules**. Nested modules should exist under the `modules/`
  subdirectory. Any nested module with a `README.md` is considered usable
  by an external user. If a README doesn't exist, it is considered for internal
  use only. These are purely advisory; Terraform will not actively deny usage
  of internal modules. Nested modules should be used to split complex behavior
  into multiple small modules that advanced users can carefully pick and
  choose. For example, the
  [Consul module](#)
  has a nested module for creating the Cluster that is separate from the
  module to setup necessary IAM policies. This allows a user to bring in their
  own IAM policy choices.

* **Examples**. Examples of using the module should exist under the
  `examples/` subdirectory. Each example may have a README to explain the
  goal and usage of the example.

A minimal recommended module following the standard structure is shown below.
While the root module is the only required element, we recommend below as
a minimum structure:

```sh
$ tree minimal-module/
.
├── README.md
├── main.tf
├── variables.tf
├── outputs.tf
```

A complete example of a module following the standard structure is shown below.
This example includes all optional elements and is therefore the most
complex a module can become:

```sh
$ tree complete-module/
.
├── README.md
├── main.tf
├── variables.tf
├── outputs.tf
├── ...
├── modules/
│   ├── nestedA/
│   │   ├── variables.tf
│   │   ├── main.tf
│   │   ├── outputs.tf
│   ├── nestedB/
│   ├── .../
├── examples/
│   ├── exampleA/
│   │   ├── main.tf
│   ├── exampleB/
│   ├── .../
```

## Publishing Modules

If you've built a module that you intend to be reused, we recommend
[publishing the module](/docs/registry/module/publish.html) on the
[Terraform Registry](https://registry.terraform.io). This will version
your module, generate documentation, and more.

Published modules can be easily consumed by Terraform, and in Terraform
0.11 you'll also be able to constrain module versions for safe and predictable
updates. The following example shows how easy it is to consume a module
from the registry:

```hcl
module "consul" {
  source = "hashicorp/consul/aws"
}
```

You can also gain all the benefits of the registry for private modules
by signing up for a [private registry](/docs/registry/private.html).
