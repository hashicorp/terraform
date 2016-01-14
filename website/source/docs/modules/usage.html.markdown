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
	source = "github.com/hashicorp/consul/terraform/aws"
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

Additionally, because these map directly to variables, they're always simple
key/value pairs. Modules can't have complex variable inputs.

## Dealing with parameters of the list type

Variables are currently unable to hold the list type. Sometimes, though, it's
desirable to parameterize a module's resource with an attribute that is of the
list type, for example `aws_instance.security_groups`. 

Until a future release broadens the functionality of variables to include list
types, the way to work around this limitation is to pass a delimited string as
a module parameter, and then "unpack" that parameter using
[`split`](/docs/configuration/interpolation.html) interpolation function within
the module definition. 

Depending on the resource parameter in question, you may have to 
indicate that the unpacked string is actually a list by using list notation.
For example:

```
resource_param = ["${split(",", var.CSV_STRING)}"]
```

## Outputs

Modules can also specify their own [outputs](/docs/configuration/outputs.html).
These outputs can be referenced in other places in your configuration.
For example:

```
resource "aws_instance" "client" {
  ami = "ami-408c7f28"
  instance_type = "t1.micro"  
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
[graph command](/docs/commands/graph.html) will show the module as a single
unit by default. You can use the `-module-depth` parameter to expand this
graph further.

For example, with a configuration similar to what we've built above, here
is what the graph output looks like by default:

<div class="center">
![Terraform Module Graph](docs/module_graph.png)
</div>

But if we set `-module-depth=-1`, the graph will look like this:

<div class="center">
![Terraform Expanded Module Graph](docs/module_graph_expand.png)
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
