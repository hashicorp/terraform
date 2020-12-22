---
layout: "language"
page_title: "camelcase - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-camelcase"
description: |-
  The camelcase function converts a string into camelCase.
---

# `camelcase` Function

`camelcase` converts a string in to camelCase: non-alphanumeric characters are removed
and the words joined by capitalising the first letter, and lower-casing the rest.

The very first letter will be lowercased. Additionally substrings which already camelCased
will be unchanged.


## Examples

```
> camelcase("hello world")
helloWorld
> camelcase("hello-world")
helloWorld
> camelcase("helloWorld")
helloWorld
```

## Related Functions

* [`kebabcase`](./kebabcase.html) converts a string into kebab-case.
* [`snakecase`](./snakecase.html) converts a string into snake_case.
