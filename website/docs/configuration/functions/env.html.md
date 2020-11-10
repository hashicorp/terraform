---
layout: "functions"
page_title: "env - Functions - Configuration Language"
sidebar_current: "docs-funcs-env-env"
description: |-
  The env function provides the value of a given environment
  variable.
---

# `env` Function

`env` provides the string value of a specified environment variable.

If the environment variable doesn't exist, this function will generate an
error. Use the `envexists` function to test for the existence of an 
environment variable.

## Examples

```
> env("environment")
development
```