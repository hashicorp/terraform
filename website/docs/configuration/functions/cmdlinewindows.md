---
layout: "functions"
page_title: "cmdlinewindows - Functions - Configuration Language"
sidebar_current: "docs-funcs-encoding-cmdlinewindows"
description: |-
  The cmdlinewindows function encodes a Microsoft Visual C++ runtime command
  line to be executed by the Windows command interpreter (cmd.exe).
---

# `cmdlineunix` Function

-> **Note:** This function is available only in Terraform 0.13 and later.

`cmdlinewindows` encodes a command line to run some Windows command line
programs via the Windows command interpreter.

```hcl
cmdlinewindows(arguments...)
```

The command line is specified as one or more string arguments, where the
first string is the program to run and the subsequent strings are each a
single argument to that program.

There is no standard encoding of command line arguments on Windows: each
program is responsible for processing the single command line string it
receives. However, the Microsoft Visual C++ runtime library startup code
has established
[some quoting conventions](https://docs.microsoft.com/en-us/cpp/c-language/parsing-c-command-line-arguments?view=vs-2019)
that are used by command line programs written in C++, and these conventions
are also implemented by other language runtimes such as Go and Python.

This function quotes each argument to be interpreted by those C++ runtime
library conventions. The result may not be interpreted as expected by programs
using incompatible command line processing rules, such as Windows Script Host
programs, command scripts, and batch files.

The resulting string will also include `cmd.exe` escape sequences using the
`^` character, ensuring that no characters are interpreted as special
characters (such as I/O redirection, pipes, etc) by the Windows command
interpreter, and therefore all of the characters can be processed by the
target program.

The Windows command interpreter offers no syntax for escaping double quotes
in the command name, so `cmdlinewindows` will simply remote any `"` characters
in the first argument. Quotes in subsequent arguments will be preserved and
escaped.

## Examples

```
> cmdlinewindows("shutdown", "/r", "/t", "0")
^"shutdown^" /r /t 0
> cmdlinewindows("python", "-c", "print(\"Hello World\")")
^"python^" -c ^"print(\^"Hello World\^")^"
```
