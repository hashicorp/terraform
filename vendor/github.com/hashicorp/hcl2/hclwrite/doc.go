// Package hclwrite deals with the problem of generating HCL configuration
// and of making specific surgical changes to existing HCL configurations.
//
// It operates at a different level of abstraction than the main HCL parser
// and AST, since details such as the placement of comments and newlines
// are preserved when unchanged.
package hclwrite
