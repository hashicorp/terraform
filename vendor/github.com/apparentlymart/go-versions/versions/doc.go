// Package versions is a library for wrangling version numbers in Go.
//
// There are many libraries offering some or all of this functionality.
// This package aims to distinguish itself by offering a more convenient and
// ergonomic API than seen in some other libraries. Code that is resolving
// versions and version constraints tends to be hairy and complex already, so
// an expressive API for talking about these concepts will hopefully help to
// make that code more readable.
//
// The version model is based on Semantic Versioning as defined at
// https://semver.org/ . Semantic Versioning does not include any specification
// for constraints, so the constraint model is based on that used by rubygems,
// allowing for upper and lower bounds as well as individual version exclusions.
package versions
