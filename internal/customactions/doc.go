// Package customactions contains the main domain logic related to scheduling
// custom actions for execution.
//
// A custom action invocation always happens as an optional extension of one
// of Terraform core's base actions. For example, the custom action
// "restore a database snapshot to create a new database" could be understood
// as an extension of the core "create" action, assuming that the remote
// API represents restoring from a snapshot in that way.
package customactions
