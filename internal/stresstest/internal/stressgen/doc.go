// Package stressgen is an internal part of the stresstest program that
// knows how to generate valid but potentially stressful Terraform
// configurations, how to verify that a final state matches the
// intent of each object in the configuration, and also how to potentially
// reduce the complexity of a configuration to automatically find a smaller
// reproduction case (although it may not always succeed).
package stressgen
