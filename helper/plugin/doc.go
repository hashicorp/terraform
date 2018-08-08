// Package plugin contains types and functions to help Terraform plugins
// implement the plugin rpc interface.
// The primary Provider type will be responsible for converting from the grpc
// wire protocol to the types and methods known to the provider
// implementations.
package plugin
