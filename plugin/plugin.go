// The plugin package exposes functions and helpers for communicating to
// Terraform plugins which are implemented as standalone binary applications.
//
// plugin.Client fully manages the lifecycle of executing the application,
// connecting to it, and returning the RPC client and service names for
// connecting to it using the terraform/rpc package.
//
// plugin.Serve fully manages listeners to expose an RPC server from a binary
// that plugin.Client can connect to.
package plugin
