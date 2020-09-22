---
layout: "functions"
page_title: "csvdecode - Functions - Configuration Language"
sidebar_current: "docs-funcs-encoding-csvdecode"
description: |-
  The csvdecode function decodes CSV data into a list of maps.
---

# `csvdecode` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`csvdecode` decodes a string containing CSV-formatted data and produces a
list of maps representing that data.

CSV is _Comma-separated Values_, an encoding format for tabular data. There
are many variants of CSV, but this function implements the format defined
in [RFC 4180](https://tools.ietf.org/html/rfc4180).

The first line of the CSV data is interpreted as a "header" row: the values
given are used as the keys in the resulting maps. Each subsequent line becomes
a single map in the resulting list, matching the keys from the header row
with the given values by index. All lines in the file must contain the same
number of fields, or this function will produce an error.

## Examples

```
> csvdecode("a,b,c\n1,2,3\n4,5,6")
[
  {
    "a" = "1"
    "b" = "2"
    "c" = "3"
  },
  {
    "a" = "4"
    "b" = "5"
    "c" = "6"
  }
]
```

## Use with the `for_each` meta-argument

You can use the result of `csvdecode` with
[the `for_each` meta-argument](/docs/configuration/resources.html#for_each-multiple-resource-instances-defined-by-a-map-or-set-of-strings)
to describe a collection of similar objects whose differences are
described by the rows in the given CSV file.

There must be one column in the CSV file that can serve as a unique id for each
row, which we can then use as the tracking key for the individual instances in
the `for_each` expression. For example:

```hcl
locals {
  # We've included this inline to create a complete example, but in practice
  # this is more likely to be loaded from a file using the "file" function.
  csv_data = <<-CSV
    local_id,instance_type,ami
    foo1,t2.micro,ami-54d2a63b
    foo2,t2.micro,ami-54d2a63b
    foo3,t2.micro,ami-54d2a63b
    bar1,m3.large,ami-54d2a63b
  CSV

  instances = csvdecode(local.csv_data)
}

resource "aws_instance" "example" {
  for_each = { for inst in local.instances : inst.local_id => inst }

  instance_type = each.value.instance_type
  ami           = each.value.ami
}
```

The `for` expression in our `for_each` argument transforms the list produced
by `csvdecode` into a map using the `local_id` as a key, which tells
Terraform to use the `local_id` value to track each instance it creates.
Terraform will create and manage the following instance addresses:

- `aws_instance.example["foo1"]`
- `aws_instance.example["foo2"]`
- `aws_instance.example["foo3"]`
- `aws_instance.example["bar1"]`

If you modify a row in the CSV on a subsequent plan, Terraform will interpret
that as an update to the existing object as long as the `local_id` value is
unchanged. If you add or remove rows from the CSV then Terraform will plan to
create or destroy associated instances as appropriate.

If there is no reasonable value you can use as a unique identifier in your CSV
then you could instead use
[the `count` meta-argument](/docs/configuration/resources.html#count-multiple-resource-instances-by-count)
to define an object for each CSV row, with each one identified by its index into
the list returned by `csvdecode`. However, in that case any future updates to
the CSV may be disruptive if they change the positions of particular objects in
the list. We recommend using `for_each` with a unique id column to make
behavior more predictable on future changes.
