---
layout: "docs"
page_title: "Using Modules"
sidebar_current: "docs-modules-usage"
description: Using modules in Terraform is very similar to defining resources.
---

# Module Usage

Using modules in Terraform is very similar to defining resources:

```
module "consul" {
  source  = "github.com/hashicorp/consul/terraform/aws"
  servers = 3
}
```

You can view the full documentation for the syntax of configuring
modules [here](/docs/configuration/modules.html).

As you can see, it is very similar to defining resources, with the exception
that we don't specify a type, and just a name. This name can be used elsewhere
in the configuration to reference the module and its variables.

The existence of the above configuration will tell Terraform to create
the resources in the "consul" module which can be found on GitHub with the
given URL. Just like a resource, the module configuration can be deleted
to remove the module.

## Multiple instances of a module

You can instantiate a module multiple times.

```
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
```
# publish_bucket/bucket-and-cloudfront.tf

variable "name" {} # this is the input parameter of the module

resource "aws_s3_bucket" "the_bucket" {
  # ...
}

resource "aws_iam_user" "deploy_user" {
  # ...
}
```

In this example you can provide module implementation in the `./publish_bucket`
subfolder - define there, how to create a bucket resource, set access and
caching rules, create e.g. a CloudFront resource, which wraps the bucket and
all the other implementation details, which are common to your project.

In the snippet above, you now use your module definition twice. The string
after the `module` keyword is a name of the instance of the module.

Note: the resource names in your implementation get prefixed by the
`module.<module-instance-name>` when instantiated. Example: your `publish_bucket`
implementation creates `aws_s3_bucket.the_bucket` and `aws_iam_access_key.deploy_user`.
The full name of the resulting resources will be `module.assets_bucket.aws_s3_bucket.the_bucket`
and `module.assets_bucket.aws_iam_access_key.deploy_user`. So beware, if you
extract your implementation to a module. The resource names will change and
this will lead to destroying s3 buckets and creating new ones - so always
check with `tf plan` before running `tf apply`.

## Source

The only required configuration key is the `source` parameter. The value of
this tells Terraform where the module can be downloaded, updated, etc.
Terraform comes with support for a variety of module sources. These
are documented on a [separate page](/docs/modules/sources.html).

Prior to running any command such as `plan` with a configuration that
uses modules, you'll have to [get](/docs/commands/get.html) the modules.
This is done using the [get command](/docs/commands/get.html).

```
$ terraform get
...
```

This command will download the modules if they haven't been already.
By default, the command will not check for updates, so it is safe (and fast)
to run multiple times. You can use the `-update` flag to check and download
updates.

## Configuration

The parameters used to configure modules, such as the `servers` parameter
above, map directly to [variables](/docs/configuration/variables.html) within
the module itself. Therefore, you can quickly discover all the configuration
for a module by inspecting the source of it very easily.

Additionally, because these map directly to variables, module configuration can
have any data type supported by variables, including maps and lists.

## Outputs

Modules can also specify their own [outputs](/docs/configuration/outputs.html).
These outputs can be referenced in other places in your configuration.
For example:

```
resource "aws_instance" "client" {
  ami               = "ami-408c7f28"
  instance_type     = "t1.micro"
  availability_zone = "${module.consul.server_availability_zone}"
}
```

This purposely is very similar to accessing resource attributes. But instead
of mapping to a resource, the variable in this case maps to an output of
a module.

Just like resources, this will create a dependency from the `aws_instance.client`
resource to the module, so the module will be built first.

## Plans and Graphs

With modules, commands such as the [plan command](/docs/commands/plan.html)
and
[graph command](/docs/commands/graph.html) will expand modules by default. You
can use the `-module-depth` parameter to limit the graph.

For example, with a configuration similar to what we've built above, here
is what the graph output looks like by default:

<div class="center">
![Terraform Expanded Module Graph](docs/module_graph_expand.png)
</div>

But if we set `-module-depth=0`, the graph will look like this:

<div class="center">
![Terraform Module Graph](docs/module_graph.png)
</div>

Other commands work similarly with modules. Note that the `-module-depth`
flag is purely a formatting flag; it doesn't affect what modules are created
or not.

## Tainting resources within a module

The [taint command](/docs/commands/taint.html) can be used to _taint_
specific resources within a module:

```
terraform taint -module=salt_master aws_instance.salt_master
```

It is not (yet) possible to taint an entire module.
