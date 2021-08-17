# Forked IP address parsing functions

This directory contains a subset of code from the Go project's `net` package
as of Go 1.16, used under the Go project license which we've included here
in [`LICENSE`](LICENSE) and [`PATENTS`](PATENTS), which are also copied from
the Go project.

Terraform has its own fork of these functions because Go 1.17 included a
breaking change to reject IPv4 address octets written with leading zeros.

The Go project rationale for that change was that Go historically interpreted
leading-zero octets inconsistently with many other implementations, trimming
off the zeros and still treating the rest as decimal rather than treating the
octet as octal.

The Go team made the reasonable observation that having a function that
interprets a non-normalized form in a manner inconsistent with other
implementations may cause naive validation or policy checks to produce
incorrect results, and thus it's a potential security concern. For more
information, see [Go issue #30999](https://golang.org/issue/30999).

After careful consideration, the Terraform team has concluded that Terraform's
use of these functions as part of the implementation of the `cidrhost`,
`cidrsubnet`, `cidrsubnets`, and `cidrnetmask` functions has a more limited
impact than the general availability of these functions in the Go standard
library, and so we can't justify a similar exception to our Terraform 1.0
compatibility promises as the Go team made to their Go 1.0 compatibility
promises.

If you're considering using this package for new functionality _other than_ the
built-in functions mentioned above, please do so only if consistency with the
behavior of those functions is important. Otherwise, new features are not
burdened by the same compatibility constraints and so should typically prefer
to use the stricter interpretation of the upstream parsing functions.
