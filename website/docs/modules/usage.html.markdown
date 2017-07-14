---
layout: "docs"
page_title: "Using Modules"
sidebar_current: "docs-modules-usage"
description: Using modules in Terraform is very similar to defining resources.
---

# Module Usage

Using modules in Terraform is very similar to defining resources:

```shell
module "consul" {
  source  = "github.com/hashicorp/consul/terraform/aws"
  servers = 3
}
```

You can view the full documentation for configuring modules in the [Module Configuration](/docs/configuration/modules.html) section.

In modules we only specify a name rather than a name and a type (as in resources). This name can be used elsewhere in the configuration to reference the module and its variables.

The existence of the above configuration will tell Terraform to create the resources in the `consul` module which can be found on GitHub at the given URL. Just like a resource, the module configuration can be deleted to remove the module.

## Multiple instances of a module

You can instantiate a module multiple times.

```hcl
# my_buckets.tf

module "assets_bucket" {
  source = "./publish_bucket"
  name   = "assets"
}

module "media_bucket" {
  source = "./publish_bucket"
  name   = "media"
}
```

```hcl
# publish_bucket/bucket-and-cloudfront.tf

variable "name" {} # this is the input parameter of the module

resource "aws_s3_bucket" "the_bucket" {
  # ...
}

resource "aws_iam_user" "deploy_user" {
  # ...
}
```

In this example you define a module in the `./publish_bucket` subdirectory. That module has configuration to create a bucket resource, set access and caching rules. The module wraps the bucket and all the other implementation details required to configure a bucket.

We can then define the module multiple times in our configuration by naming each instantiation of the module uniquely, here `module "assets_bucket"` and `module "media_bucket"`, whilst specifying the same module `source`.

The resource names in your module  get prefixed by `module.<module-instance-name>` when instantiated, for example the `publish_bucket` module creates `aws_s3_bucket.the_bucket` and `aws_iam_access_key.deploy_user`. The full name of the resulting resources will be `module.assets_bucket.aws_s3_bucket.the_bucket` and `module.assets_bucket.aws_iam_access_key.deploy_user`. Be cautious of this when extracting configuration from your files into a module, the name of your resources will change and Terraform will potentially destroy and recreate them. Always check your configuration with `terraform plan` before running `terraform apply`.

## Source

The only required configuration key for a module is the `source` parameter. The value of this tells Terraform where the module can be downloaded, updated, etc. Terraform comes with support for a variety of module sources. These
are documented in the [Module sources documentation](/docs/modules/sources.html).

Prior to running any Terraform command with a configuration that uses modules, you'll have to [get](/docs/commands/get.html) the modules. This is done using the [get command](/docs/commands/get.html).

```shell
$ terraform get
```

This command will download the modules if they haven't been already.

By default, the command will not check for updates, so it is safe (and fast) to run multiple times. You can use the `-update` flag to check and download updates.

## Configuration

The parameters used to configure modules, such as the `servers` parameter above, map directly to [variables](/docs/configuration/variables.html) within the module itself. Therefore, you can quickly discover all the configuration
for a module by inspecting the source of it.

Additionally, because these map directly to variables, module configuration can have any data type available for variables, including maps and lists.

## Outputs

Modules can also specify their own [outputs](/docs/configuration/outputs.html). These outputs can be referenced in other places in your configuration, for example:

```hcl
resource "aws_instance" "client" {
  ami               = "ami-408c7f28"
  instance_type     = "t1.micro"
  availability_zone = "${module.consul.server_availability_zone}"
}
```

This purposely is very similar to accessing resource attributes. Instead of mapping to a resource, however, the variable in this case maps to an output of a module.

Just like resources, this will create a dependency from the `aws_instance.client` resource to the module, so the module will be built first.

To use module outputs via command line you have to specify the module name before the variable, for example:

```shell
$ terraform output -module=consul server_availability_zone
```

## Plans and Graphs

Commands such as the [plan command](/docs/commands/plan.html) and [graph command](/docs/commands/graph.html) will expand modules by default. You can use the `-module-depth` parameter to limit the graph.

For example, with a configuration similar to what we've built above, here is what the graph output looks like by default:

![Terraform Expanded Module Graph](docs/module_graph_expand.png)

If instead we set `-module-depth=0`, the graph will look like this:

![Terraform Module Graph](docs/module_graph.png)

Other commands work similarly with modules. Note that the `-module-depth` flag is purely a formatting flag; it doesn't affect what modules are created or not.

## Tainting resources within a module

The [taint command](/docs/commands/taint.html) can be used to _taint_ specific resources within a module:

```shell
$ terraform taint -module=salt_master aws_instance.salt_master
```

It is currently not possible to taint an entire module.
