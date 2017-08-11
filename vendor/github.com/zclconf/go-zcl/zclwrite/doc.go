// Package zclwrite deals with the problem of generating zcl configuration
// and of making specific surgical changes to existing zcl configurations.
//
// It operates at a different level of abstraction that the main zcl parser
// and AST, since details such as the placement of comments and newlines
// are preserved when unchanged.
package zclwrite
