---
layout: "language"
page_title: "snakecase - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-snakecase"
description: |-
  The snakecase function converts a string into snake_case.
---

# `snakecase` Function

`snakecase` converts a string in to snake_case: non-alphanumeric characters are removed
and the words are delimited by an underscore. The resulting string is lowercase.


## Examples

```
> snakecase("hello world")
hello_world
> snakecase("hello-world")
hello_world
> snakecase("helloWorld")
hello_world
```

## Related Functions

* [`camelcase`](./camelcase.html) converts a string into camelCase.
* [`kebabcase`](./kebabcase.html) converts a string into kebab-case.
