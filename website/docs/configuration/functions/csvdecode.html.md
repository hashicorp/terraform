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
    "a" = 1
    "b" = 2
    "c" = 3
  },
  {
    "a" = 4
    "b" = 5
    "c" = 6
  }
]
```

## Use with the `count` meta-argument

It can be tempting to use `csvdecode` to generate a set of similar resources
using the `count` meta-argument, as in this example:

```hcl
locals {
  instances = csvdecode(file("${path.module}/instances.csv"))
}

resource "aws_instance" "example" {
  count = len(local.instances) # Beware! (see below)

  instance_type = local.instances[count.index].instance_type
  ami           = local.instances[count.index].ami
}
```

The above example will work on initial creation, but if any rows are removed
from the CSV file, or if the records in the CSV file are re-ordered, Terraform
will not understand that the ordering has changed and will instead interpret
this as requests for changes to many or all of the instances, which will in
turn force these instances to be destroyed and re-created.

The above pattern can be used with care in situations where, for example, the
CSV file is only ever appended to, or if mass-updating the resources would
not be harmful, but in general we recommend avoiding the above pattern.
