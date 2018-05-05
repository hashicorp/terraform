---
layout: "intro"
page_title: "Input Variables"
sidebar_current: "gettingstarted-variables"
description: |-
  You now have enough Terraform knowledge to create useful configurations, but we're still hardcoding values. To become truly shareable and committable to version control, we need to parameterize the configurations. This page introduces input variables as a way to do this.
---

# Input Variables

You now have enough Terraform knowledge to create useful
configurations, but we're still hard-coding things that we might wish 
to change. Infrastructure as Code allows us to define variables for anything 
that needs to be changed or adjusted by the user.

## Defining Variables

Let's create a couple of variables. Create another file `variables.tf` with
the following contents.

-> **Note**: that the file can be named anything, since Terraform loads all
files ending in `.tf` in a directory.

```hcl
variable "myname" {}
variable "region" {
  default = "us-east-1"
}
```

This defines two variables within your Terraform configuration. The first
one, `myname` has an empty block `{}`. The region variable sets a default. 
If a default value is set, the variable is optional. Otherwise, the variable
is required. If you run `terraform plan` now, Terraform will prompt you for 
the value of the `myname` variable. You can define a variable even if you're 
not using it yet.

## Using Variables in Configuration

Next, replace the AWS provider configuration with the following:

```hcl
provider "aws" {
  region     = "${var.region}"
}
```

Here we're telling Terraform to replace the "${var.region}" with your
desired region. Since we set a default value, the infrastructure will be 
built in the `us-east-1` AWS region.

## Assigning Variables

There are multiple ways to assign variables. Below is also the order
in which variable values are chosen. The following is the *descending* order
of precedence in which variables are considered.

#### Command-line flags

Variables defined on the command line will override any other settings.
You can set variables directly on the command-line with the
`-var` flag. Any command in Terraform that inspects the configuration
accepts this flag, such as `apply`, `plan`, and `refresh`:

```
$ terraform apply \
  -var 'myname=alice'
# ...
```

Once again, setting variables this way will not save them, and they'll
have to be input repeatedly as commands are executed.

#### From a file

To persist variable values, you can create a file and assign 
variables within this file. Create a file named `terraform.tfvars` 
with the following contents:

```hcl
myname = "alice"
```

For all files which match `terraform.tfvars` or `*.auto.tfvars` present in the
current directory, Terraform automatically loads them to populate variables. If
the file is named something else, you can use the `-var-file` flag directly to
specify a file. These files use the same syntax as Terraform
configuration files. And like Terraform configuration files, these files
can also be JSON.

You can use multiple `-var-file` arguments in a single command, with some
checked in to version control and others not checked in. For example:

```
$ terraform apply \
  -var-file="secret.tfvars" \
  -var-file="production.tfvars"
```

#### From environment variables

Terraform will read environment variables in the form of `TF_VAR_name`
to find the value for a variable. For example, the `TF_VAR_region`
variable can be set to set the `region` environment variable.

-> **Note**: Environment variables can only populate string-type variables.
List and map type variables must be populated via one of the other mechanisms.

#### UI Input

If you execute `terraform apply` with certain variables unspecified,
Terraform will ask you to input their values interactively.  These
values are not saved, but this provides a convenient workflow when getting
started with Terraform. UI Input is not recommended for everyday use of
Terraform.

-> **Note**: UI Input is only supported for string variables. List and map
variables must be populated via one of the other mechanisms.

#### Variable Defaults

If no value is assigned to a variable via any of these methods and the
variable has a `default` key in its declaration, that value will be used
for the variable.

<a id="lists"></a>
## Lists

Lists are defined either explicitly or implicitly

```hcl
# implicitly by using brackets [...]
variable "cidrs" { default = [] }

# explicitly
variable "cidrs" { type = "list" }
```

You can specify lists in a `terraform.tfvars` file:

```hcl
cidrs = [ "10.0.0.0/16", "10.1.0.0/16" ]
```

## Maps

We've replaced our region with a variable, but we are still
hard-coding the AMI. Unfortunately, AMIs are specific to the region
that is in use. One option is to just ask the user to input the proper
AMI for the region, but Terraform can do better than that with
_maps_.

Maps are a way to create variables that are lookup tables. An example
will show this best. Let's extract our AMIs into a map and add
support for the `us-west-2` region as well:

```hcl
variable "amis" {
  type = "map"
  default = {
    "us-east-1" = "ami-b374d5a5"
    "us-west-2" = "ami-4b32be2b"
  }
}
```

A variable can have a `map` type assigned explicitly, or it can be implicitly
declared as a map by specifying a default value that is a map. The above
demonstrates both.

Then, replace the `aws_instance` with the following:

```hcl
resource "aws_instance" "example" {
  ami           = "${lookup(var.amis, var.region)}"
  instance_type = "t2.micro"
}
```

This introduces a new type of interpolation: a function call. The
`lookup` function does a dynamic lookup in a map for a key. The
key is `var.region`, which specifies that the value of the region
variables is the key.

While we don't use it in our example, it is worth noting that you
can also do a static lookup of a map directly with
`${var.amis["us-east-1"]}`.

## Assigning Maps

We set defaults above, but maps can also be set using the `-var` and
`-var-file` values. For example:

```
$ terraform apply -var 'amis={ us-east-1 = "foo", us-west-2 = "bar" }'
# ...
```

-> **Note**: Even if every key will be assigned as input, the variable must be
established as a map by setting its default to `{}`.

Here is an example of setting a map's keys from a file. Starting with these
variable definitions:

```hcl
variable "region" {}
variable "amis" {
  type = "map"
}
```

You can specify keys in a `terraform.tfvars` file:

```hcl
amis = {
  "us-east-1" = "ami-abc123"
  "us-west-2" = "ami-def456"
}
```

And access them via `lookup()`:

```hcl
output "ami" {
  value = "${lookup(var.amis, var.region)}"
}
```

Like so:

```
$ terraform apply -var region=us-west-2

Apply complete! Resources: 0 added, 0 changed, 0 destroyed.

Outputs:

  ami = ami-def456
```

## Next

Terraform provides variables for parameterizing your configurations.
Maps let you build lookup tables in cases where that makes sense.
Setting and using variables is uniform throughout your configurations.

In the next section, we'll take a look at
[output variables](/intro/getting-started/outputs.html) as a mechanism
to expose certain values more prominently to the Terraform operator.
