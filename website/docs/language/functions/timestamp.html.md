---
layout: "language"
page_title: "timestamp - Functions - Configuration Language"
sidebar_current: "docs-funcs-datetime-timestamp"
description: |-
  The timestamp function returns a string representation of the current date
  and time.
---

# `timestamp` Function

`timestamp` returns a UTC timestamp string in [RFC 3339](https://datatracker.ietf.org/doc/html/rfc3339) format.

In the Terraform language, timestamps are conventionally represented as
strings using [RFC 3339](https://datatracker.ietf.org/doc/html/rfc3339)
"Date and Time format" syntax, and so `timestamp` returns a string
in this format.

The result of this function will change every second, so using this function
directly with resource attributes will cause a diff to be detected on every
Terraform run. We do not recommend using this function in resource attributes,
but in rare cases it can be used in conjunction with
[the `ignore_changes` lifecycle meta-argument](/docs/language/meta-arguments/lifecycle.html#ignore_changes)
to take the timestamp only on initial creation of the resource. For more stable
time handling, see the [Time Provider](https://registry.terraform.io/providers/hashicorp/time/).

If this function produced a value during the plan step, it would cause the final configuration during the apply step not to match the actions shown in the plan (since the function is called again in the apply step, and would return a different value), which violates the Terraform execution model. For that reason, this function produces an unknown value result during the plan step, with the real result being decided only during the apply step. This means that the recorded time will be the instant when Terraform began _applying_ the change, rather than when Terraform _planned_ the change.

## Examples

```
> timestamp()
2018-05-13T07:44:12Z
```

## Related Functions

* [`formatdate`](./formatdate.html) can convert the resulting timestamp to
  other date and time formats.
