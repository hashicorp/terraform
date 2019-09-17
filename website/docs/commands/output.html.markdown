---
layout: "docs"
page_title: "Command: output"
sidebar_current: "docs-commands-output"
description: |-
  The `terraform output` command is used to extract the value of an output variable from the state file.
---

# Command: output

The `terraform output` command is used to extract the value of
an output variable from the state file.

## Usage

Usage: `terraform output [options] [NAME]`

With no additional arguments, `output` will display all the outputs for
the root module. If an output `NAME` is specified, only the value of that
output is printed.

The command-line flags are all optional. The list of available flags are:

* `-json` - If specified, the outputs are formatted as a JSON object, with
    a key per output. If `NAME` is specified, only the output specified will be
    returned. This can be piped into tools such as `jq` for further processing.
* `-no-color` - If specified, output won't contain any color.
* `-state=path` - Path to the state file. Defaults to "terraform.tfstate".
    Ignored when [remote state](/docs/state/remote.html) is used.

## Examples

These examples assume the following Terraform output snippet.

```hcl
output "lb_address" {
  value = "${aws_alb.web.public_dns}"
}

output "instance_ips" {
  value = ["${aws_instance.web.*.public_ip}"]
}

output "password" {
  sensitive = true
  value = ["${var.secret_password}"]
}
```

To list all outputs:

```shell
$ terraform output
```

Note that outputs with the `sensitive` attribute will be redacted:
```shell
$ terraform output password
password = <sensitive>
```

To query for the DNS address of the load balancer:

```shell
$ terraform output lb_address
my-app-alb-1657023003.us-east-1.elb.amazonaws.com
```

To query for all instance IP addresses:

```shell
$ terraform output instance_ips
test = [
    54.43.114.12,
    52.122.13.4,
    52.4.116.53
]
```

To query for a particular value in a list, use `-json` and a JSON
command-line parser such as [jq](https://stedolan.github.io/jq/).
For example, to query for the first instance's IP address:

```shell
$ terraform output -json instance_ips | jq '.value[0]'
```
