// Package syntax deals with the low-level syntax details of the CLI
// configuration, exposing an HCL-compatible API to callers.
//
// The CLI configuration continutes to use HCL 1 syntax primitives because it
// must remain compatible with the processing done by earlier Terraform
// versions, but the HCL 1 API makes it difficult to write a robust
// configuration decoder that gives good user feedback.
//
// Therefore this package takes the rather unusual strategy of implementing
// HCL 2's syntax-agnostic decoding API on top of a subset of the HCL 1 API
// that was exercised by previous versions of the CLI config decoder. Using
// the HCL 2 API conventions here might allow mixed-mode parsing in future
// versions where some files can use real HCL 2 syntax, but for now the CLI
// config format doesn't need any HCL-2-unique features and so we're supporting
// HCL 1 syntax (with environment variable substitution in a few specific spots)
// only.
package syntax
