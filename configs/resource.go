package configs

import (
	"github.com/hashicorp/hcl2/hcl"
)

// ManagedResource represents a "resource" block in a module or file.
type ManagedResource struct {
	Name    string
	Type    string
	Config  hcl.Body
	Count   hcl.Expression
	ForEach hcl.Expression

	ProviderConfigAddr hcl.Traversal

	DependsOn []hcl.Traversal

	Connection   *Connection
	Provisioners []*Provisioner

	CreateBeforeDestroy bool
	PreventDestroy      bool
	IgnoreChanges       []hcl.Traversal

	DeclRange hcl.Range
}

// DataResource represents a "data" block in a module or file.
type DataResource struct {
	Name    string
	Type    string
	Config  hcl.Body
	Count   hcl.Expression
	ForEach hcl.Expression

	ProviderConfigAddr hcl.Traversal

	DependsOn []hcl.Traversal

	DeclRange hcl.Range
}

// Provisioner represents a "provisioner" block when used within a
// "resource" block in a module or file.
type Provisioner struct {
	Type       string
	Config     hcl.Body
	Connection *Connection
	When       ProvisionerWhen
	OnFailure  ProvisionerOnFailure

	DeclRange hcl.Range
}

// Connection represents a "connection" block when used within either a
// "resource" or "provisioner" block in a module or file.
type Connection struct {
	Type   string
	Config hcl.Body

	DeclRange hcl.Range
}

// ProvisionerWhen is an enum for valid values for when to run provisioners.
type ProvisionerWhen int

//go:generate stringer -type ProvisionerWhen

const (
	ProvisionerWhenInvalid ProvisionerWhen = iota
	ProvisionerWhenCreate
	ProvisionerWhenDestroy
)

// ProvisionerOnFailure is an enum for valid values for on_failure options
// for provisioners.
type ProvisionerOnFailure int

//go:generate stringer -type ProvisionerOnFailure

const (
	ProvisionerOnFailureInvalid ProvisionerOnFailure = iota
	ProvisionerOnFailureContinue
	ProvisionerOnFailureFail
)
