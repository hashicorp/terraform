Columnize
=========

Easy column-formatted output for golang

[![Build Status](https://travis-ci.org/ryanuber/columnize.svg)](https://travis-ci.org/ryanuber/columnize)

Columnize is a really small Go package that makes building CLI's a little bit
easier. In some CLI designs, you want to output a number similar items in a
human-readable way with nicely aligned columns. However, figuring out how wide
to make each column is a boring problem to solve and eats your valuable time.

Here is an example:

```go
package main

import (
    "fmt"
    "github.com/ryanuber/columnize"
)

func main() {
    output := []string{
        "Name | Gender | Age",
        "Bob | Male | 38",
        "Sally | Female | 26",
    }
    result := columnize.SimpleFormat(output)
    fmt.Println(result)
}
```

As you can see, you just pass in a list of strings. And the result:

```
Name   Gender  Age
Bob    Male    38
Sally  Female  26
```

Columnize is tolerant of missing or empty fields, or even empty lines, so
passing in extra lines for spacing should show up as you would expect.

Configuration
=============

Columnize is configured using a `Config`, which can be obtained by calling the
`DefaultConfig()` method. You can then tweak the settings in the resulting
`Config`:

```
config := columnize.DefaultConfig()
config.Delim = "|"
config.Glue = "  "
config.Prefix = ""
config.Empty = ""
```

* `Delim` is the string by which columns of **input** are delimited
* `Glue` is the string by which columns of **output** are delimited
* `Prefix` is a string by which each line of **output** is prefixed
* `Empty` is a string used to replace blank values found in output

You can then pass the `Config` in using the `Format` method (signature below) to
have text formatted to your liking.

Usage
=====

```go
SimpleFormat(intput []string) string

Format(input []string, config *Config) string
```
