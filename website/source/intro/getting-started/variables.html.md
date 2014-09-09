---
layout: "intro"
page_title: "Input Variables"
sidebar_current: "gettingstarted-variables"
---

# Input Variables

You now have enough Terraform knowledge to create useful
configurations, but we're still hardcoding access keys,
AMIs, etc. To become truly shareable and commitable to version
control, we need to parameterize the configurations. This page
introduces input variables as a way to do this.

## Defining Variables

Let's first extract our access key, secret key, and region
into a few variables. Create another file `variables.tf` with
the following contents. Note that the file can be named anything,
since Terraform loads all files ending in `.tf` in a directory.

```
variable "access_key" {}
variable "secret_key" {}
variable "region" {
	default = "us-east-1"
}
```

This defines three variables within your Terraform configuration.
The first two have empty blocks `{}`. The third sets a default. If
a default value is set, the variable is optional. Otherwise, the
variable is required. If you run `terraform plan` now, Terraform will
error since the required variables are not set.

## Using Variables in Configuration

Next, replace the AWS provider configuration with the following:

```
provider "aws" {
	access_key = "${var.access_key}"
	secret_key = "${var.secret_key}"
	region = "${var.region}"
}
```

This uses more interpolations, this time prefixed with `var.`. This
tells Terraform that you're accessing variables. This configures
the AWS provider with the given variables.

## Assigning Variables

There are two ways to assign variables.

First, you can set it directly on the command-line with the
`-var` flag. Any command in Terraform that inspects the configuration
accepts this flag, such as `apply`, `plan`, and `refresh`:

```
$ terraform plan \
  -var 'access_key=foo' \
  -var 'secret_key=bar'
...
```

Second, you can create a file and assign variables directly. Create
a file named "terraform.tfvars" with the following contents:

```
access_key = "foo"
secret_key = "bar"
```

If a "terraform.tfvars" file is present in the current directory,
Terraform automatically loads it to populate variables. If the file is
named something else, you can use the `-var-file` flag directly to
specify a file. Like configuration files, variable files can also be
JSON.

We recommend using the "terraform.tfvars" file, and ignoring it from
version control.

## Mappings

We've replaced our sensitive strings with variables, but we still
are hardcoding AMIs. Unfortunately, AMIs are specific to the region
that is in use. One option is to just ask the user to input the proper
AMI for the region, but Terraform can do better than that with
_mappings_.

Mappings are a way to create variables that are lookup tables. An example
will show this best. Let's extract our AMIs into a mapping and add
support for the "us-west-2" region as well:

```
variable "amis" {
	default = {
		us-east-1 = "ami-aa7ab6c2"
		us-west-2 = "ami-23f78e13"
	}
}
```

A variable becomes a mapping when it has a default value that is a
map like above. There is no way to create a required map.

Then, replace the "aws\_instance" with the following:

```
resource "aws_instance" "example" {
	ami = "${lookup(var.amis, var.region)}"
	instance_type = "t1.micro"
}
```

This introduces a new type of interpolation: a function call. The
`lookup` function does a dynamic lookup in a map for a key. The
key is `var.region`, which specifies that the value of the region
variables is the key.

While we don't use it in our example, it is worth noting that you
can also do a static lookup of a mapping directly with
`${var.amis.us-east-1}`.

We set defaults, but mappings can also be overridden using the
`-var` and `-var-file` values. For example, if the user wanted to
specify an alternate AMI for us-east-1:

```
$ terraform plan -var 'amis.us-east-1=foo'
...
```

## Next

Terraform provides variables for parameterizing your configurations.
Mappings let you build lookup tables in cases where that makes sense.
Setting and using variables is uniform throughout your configurations.

In the next section, we'll take a look at
[output variables](/intro/getting-started/outputs.html) as a mechanism
to expose certain values more prominently to the Terraform operator.
