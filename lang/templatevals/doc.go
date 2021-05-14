// Package templatevals deals with the idea of "template values" in the
// Terraform language, which allow passing around not-yet-evaluated string
// templates in a structured way that allows for type checking and avoids
// confusing additional template escaping.
package templatevals
