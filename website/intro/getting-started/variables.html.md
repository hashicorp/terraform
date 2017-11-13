---
layout: "intro"
page_title: "Input Variables"
sidebar_current: "gettingstarted-variables"
description: |-
  You now have enough Terraform knowledge to create useful configurations, but we're still hardcoding access keys, AMIs, etc. To become truly shareable and committable to version control, we need to parameterize the configurations. This page introduces input variables as a way to do this.
---

# Input Variables

You now have enough Terraform knowledge to create useful
configurations, but we're still hard-coding access keys,
AMIs, etc. To become truly shareable and version
controlled, we need to parameterize the configurations. This page
introduces input variables as a way to do this.

## Defining Variables

Let's first extract our access key, secret key, and region
into a few variables. Create another file `variables.tf` with
the following contents.

-> **Note**: that the file can be named anything, since Terraform loads all
files ending in `.tf` in a directory.

```hcl
variable "access_key" {}
variable "secret_key" {}
variable "region" {
  default = "us-east-1"
}
```

This defines three variables within your Terraform configuration.  The first
two have empty blocks `{}`. The third sets a default. If a default value is
set, the variable is optional. Otherwise, the variable is required. If you run
`terraform plan` now, Terraform will prompt you for the values for unset string
variables.

## Using Variables in Configuration

Next, replace the AWS provider configuration with the following:

```hcl
provider "aws" {
  access_key = "${var.access_key}"
  secret_key = "${var.secret_key}"
  region     = "${var.region}"
}
```

This uses more interpolations, this time prefixed with `var.`. This
tells Terraform that you're accessing variables. This configures
the AWS provider with the given variables.

## Assigning Variables

There are multiple ways to assign variables. Below is also the order
in which variable values are chosen. The following is the descending order
of precedence in which variables are considered.

#### Command-line flags

You can set variables directly on the command-line with the
`-var` flag. Any command in Terraform that inspects the configuration
accepts this flag, such as `apply`, `plan`, and `refresh`:

```
$ terraform apply \
  -var 'access_key=foo' \
  -var 'secret_key=bar'
# ...
```

Once again, setting variables this way will not save them, and they'll
have to be input repeatedly as commands are executed.

#### From a file

To persist variable values, create a file and assign variables within
this file. Create a file named `terraform.tfvars` with the following
contents:

```hcl
access_key = "foo"
secret_key = "bar"
```

For all files which match `terraform.tfvars` or `*.auto.tfvars` present in the
current directory, Terraform automatically loads them to populate variables. If
the file is named something else, you can use the `-var-file` flag directly to
specify a file. These files are the same syntax as Terraform
configuration files. And like Terraform configuration files, these files
can also be JSON.

We don't recommend saving usernames and password to version control, But you
can create a local secret variables file and use `-var-file` to load it.

You can use multiple `-var-file` arguments in a single command, with some
checked in to version control and others not checked in. For example:

```
$ terraform apply \
  -var-file="secret.tfvars" \
  -var-file="production.tfvars"
```

#### From environment variables

Terraform will read environment variables in the form of `TF_VAR_name`
to find the value for a variable. For example, the `TF_VAR_access_key`
variable can be set to set the `access_key` variable.

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

We've replaced our sensitive strings with variables, but we still
are hard-coding AMIs. Unfortunately, AMIs are specific to the region
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
