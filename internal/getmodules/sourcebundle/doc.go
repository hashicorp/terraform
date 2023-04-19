// Package sourcebundle deals with the problem of fetching a bunch of distinct
// module packages and packaging them up together os that they can be sent
// to a different execution context without the need to separately re-fetch
// the same source code.
//
// This package has no direct awareness of the Terraform language and so it
// needs some help from its caller to discover dependencies in the loaded
// modules and chase down other module packages needed to work with those.
// See [DependencyFinder] for more on that.
package sourcebundle
