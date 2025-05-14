# How Terraform Uses Unicode

The Terraform language uses the Unicode standards as the basis of various
different features. The Unicode Consortium publishes new versions of those
standards periodically, and we aim to adopt those new versions in new
minor releases of Terraform in order to support additional characters added
in those new versions.

Unfortunately due to those features being implemented by relying on a number
of external libraries, adopting a new version of Unicode is not as simple as
just updating a version number somewhere. This document aims to describe the
various steps required to adopt a new version of Unicode in Terraform.

We typically aim to be consistent across all of these dependencies as to which
major version of Unicode we currently conform to. The usual initial driver
for a Unicode upgrade is switching to new version of the Go runtime library
which itself uses a new version of Unicode, because Go itself does not provide
any way to select Unicode versions independently from Go versions. Therefore
we typically upgrade to a new Unicode version only in conjunction with
upgrading to a new Go version.

## Unicode tables in the Go standard library

Several Terraform language features are implemented in terms of functions in
[the Go `strings` package](https://pkg.go.dev/strings),
[the Go `unicode` package](https://pkg.go.dev/unicode), and other supporting
packages in the Go standard library.

The Go team maintains the Go standard library features to support a particular
Unicode version for each Go version. The specific Unicode version for a
particular Go version is available in
[`unicode.Version`](https://pkg.go.dev/unicode#Version).

We adopt a new version of Go by editing the `.go-version` file in the root
of this repository. Although it's typically possible to build Terraform with
other versions of Go, that file documents the version we intend to use for
official releases and thus the primary version we use for development and
testing. Adopting a new Go version typically also implies other behavior
changes inherited from the Go standard library, so it's important to review the
relevant version changelog(s) to note any behavior changes we'll need to pass
on to our own users via the Terraform changelog.

The other subsystems described below should always be set up to match
`unicode.Version`. In some cases those libraries automatically try to align
themselves with `unicode.Version` and generate an error if they cannot, but
that isn't true of all of them.

## Unicode Identifier Rules in HCL

_Identifier and Pattern Syntax_ (TF31) is a Unicode standards annex which
describe a set of rules for tokenizing "identifiers", such as variable names
in a programming language.

HCL uses a superset of that specification for its own identifier tokenization
rules, and so it includes some code derived from the TF31 data tables that
describe which characters belong to the "ID_Start" and "ID_Continue" classes.

Since Terraform is the primary user of HCL, it's typically Terraform's adoption
of a new Unicode version which drives HCL to adopt one. To update the Unicode
tables to a new version:
* Edit `hclsyntax/generate.go`'s line which runs `unicode2ragel.rb` to specify
  the URL of the `DerivedCoreProperties.txt` data file for the intended Unicode
  version.
* Run `go generate ./hclsyntax` to run the generation code to update both
  `unicode_derived.rl` and, indirectly, `scan_tokens.go`. (You will need both
  a Ruby interpreter and the Ragel state machine compiler on your system in
  order to complete this step.)
* Run all the tests to check for regressions: `go test ./...`
* If all looks good, commit all of the changes and open a PR to HCL.
* Once that PR is merged and released, update Terraform to use the new version
  of HCL.

## Unicode Text Segmentation

_Text Segmentation_ (TR29) is a Unicode standards annex which describes
algorithms for breaking strings into smaller units such as sentences, words,
and grapheme clusters.

Several Terraform language features make use of the _grapheme cluster_
algorithm in particular, because it provides a practical definition of
individual visible characters, taking into account combining sequences such
as Latin letters with separate diacritics or Emoji characters with gender
presentation and skin tone modifiers.

The text segmentation algorithms rely on supplementary data tables that are
not part of the core set encoded in the Go standard library's `unicode`
packages, and so instead we rely on the third-party module
[`github.com/apparentlymart/go-textseg`](http://pkg.go.dev/github.com/apparentlymart/go-textseg)
to provide those tables and a Go implementation of the grapheme cluster
segmentation algorithm in terms of the tables.

The `go-textseg` library is designed to allow calling programs to potentially
support multiple Unicode versions at once, by offering a separate module major
version for each Unicode major version. For example, the full module path for
the Unicode 13 implementation is `github.com/apparentlymart/go-textseg/v13`.

If that external library doesn't yet have support for the Unicode version we
intend to adopt then we'll first need to open a pull request to contribute
new language support. The details of how to do this will unfortunately vary
depending on how significantly the Text Segmentation annex has changed since
the most recently-supported Unicode version, but in many cases it can be
just a matter of editing that library's `make_tables.go`, `make_test_tables.go`,
and `generate.go` files to point to the URLs where the Unicode consortium
published new tables and then run `go generate` to rebuild the files derived
from those data sources. As long as the new Unicode version has only changed
the data tables and not also changed the algorithm, often no further changes
are needed.

Once a new Unicode version is included, the maintainer of that library will
typically publish a new major version that we can depend on. Two different
codebases included in Terraform all depend directly on the `go-textseg` module
for parts of their functionality:

* [`hashicorp/hcl`](https://github.com/hashicorp/hcl) uses text
  segmentation as part of producing visual column offsets in source ranges
  returned by the tokenizer and parser. Terraform in turn uses that library
  for the underlying syntax of the Terraform language, and so it passes on
  those source ranges to the end-user as part of diagnostic messages.
* The third-party module [`github.com/zclconf/go-cty`](https://github.com/zclconf/go-cty)
  provides several of the Terraform language built in functions, including
  functions like `substr` and `length` which need to count grapheme clusters
  as part of their implementation.

As part of upgrading Terraform's Unicode support we therefore typically also
open pull requests against these other codebases, and then adopt the new
versions that produces. Terraform work often drives the adoption of new Unicode
versions in those codebases, with other dependencies following along when they
next upgrade.

At the time of writing Terraform itself doesn't _directly_ depend on
`go-textseg`, and so there are no specific changes required in this Terraform
codebase aside from the `go.sum` file update that always follows from
changes to transitive dependencies.

The `go-textseg` library does have a different "auto-version" mechanism which
selects an appropriate module version based on the current Go language version,
but neither HCL nor cty use that because the auto-version package will not
compile for any Go version that doesn't have a corresponding Unicode version
explicitly recorded in that repository, and so that would be too harsh a
constraint for libraries like HCL which have many callers, many of which don't
care strongly about Unicode support, that may wish to upgrade Go before the
text segmentation library has been updated.
