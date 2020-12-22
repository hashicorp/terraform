---
layout: "language"
page_title: "kebabcase - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-kebabcase"
description: |-
  The kebabcase function converts a string into kebab-case.
---

# `kebabcase` Function

`kebabcase` converts a string in to kebab-case: non-alphanumeric characters are removed
and the words are delimited by a hyphen. The resulting string is lowercase.


## Examples

```
> kebabcase("hello world")
hello-world
> kebabcase("hello-world")
hello-world
> kebabcase("helloWorld")
hello-world
```

## Related Functions

* [`camelcase`](./camelcase.html) converts a string into camelCase.
* [`snakecase`](./snakecase.html) converts a string into snake_case.
