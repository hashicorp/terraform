---
layout: "functions"
page_title: "envexists - Functions - Configuration Language"
sidebar_current: "docs-funcs-env-envexists"
description: |-
  The envexists function determines whether a specified environment variable exists.
---

# `envexists` Function

`envexists` determines whether a specified environment variable exists.

The `env` function can be used to retrieve the string value of an environment variable.

## Examples

```
> envexists("environment")
true
```