---
layout: "functions"
page_title: "cmdlineunix - Functions - Configuration Language"
sidebar_current: "docs-funcs-encoding-cmdlineunix"
description: |-
  The cmdlineunix function encodes a command line to be executed by a
  Unix-style shell.
---

# `cmdlineunix` Function

-> **Note:** This function is available only in Terraform 0.13 and later.

`cmdlineunix` encodes a command line to run a program via a Unix-style shell.

```hcl
cmdlineunix(argv...)
```

The command line is specified as one or more string arguments, where the
first string is the program to run and the subsequent strings are each a
single argument to that program.

The program name is always returned in single quotes, to avoid interpretation
as an alias and thus minimize the effect of the local configuration of the
shell that will ultimately execute the command.

Arguments will each be returned in single quotes only if necessary to ensure
literal interpretation. The purpose of this function is to ensure that the
given strings are taken literally; it is not possible to include actual
shell metacharacters like I/O redirection, pipes, etc.

This function is primarily intended for encoding an entire literal command
line at once, rather than individual arguments in a larger command line.
However, because the single quoting applied unconditionally to the first
argument is also suitable for subsequent arguments, the result of this
function can potentially be used for individual arguments too.

## Examples

```
> cmdlineunix("systemctl", "start", "example.service")
'systemctl' start example.service
> cmdlineunix("cat", "Example File.txt")
'cat' 'Example File.txt'
> "${cmdlineunix("whoami")} >/tmp/username.txt"
'whoami' >/tmp/username.txt
```
