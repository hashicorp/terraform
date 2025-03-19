// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stacksplugin

import (
	"io"

	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/stacks"
	"github.com/hashicorp/terraform/internal/stacksplugin/stacksproto1/dependencies"
	"github.com/hashicorp/terraform/internal/stacksplugin/stacksproto1/packages"
)

type Dependencies interface {
	// Opens a source bundle that was already extracted into the filesystem
	// somewhere, returning an opaque source bundle handle that can be used for
	// subsequent operations.
	OpenSourceBundle(dependencies.OpenSourceBundle_Request) dependencies.OpenSourceBundle_Response

	// Closes a previously-opened source bundle, invalidating the given handle
	// and therefore making it safe to delete or modify the bundle directory
	// on disk.
	CloseSourceBundle(dependencies.CloseSourceBundle_Request) dependencies.CloseSourceBundle_Response

	// Reads and parses an existing dependency lock file from the filesystem,
	// returning a dependency locks handle.
	//
	// This function parses a user-provided source file, and so invalid content
	// in that file is treated as diagnostics in a successful response rather
	// than as an error. Callers must check whether the dependency locks
	// handle in the response is set (dependencies.non-zero) before using it, and treat
	// an unset handle as indicating a user error which is described in the
	// accompanying diagnostics. Diagnostics can also be returned along with
	// a valid handle, e.g. if there are non-blocking warning diagnostics.
	OpenDependencyLockFile(dependencies.OpenDependencyLockFile_Request) dependencies.OpenDependencyLockFile_Response

	// Creates an in-memory-only dependency locks handle with a fixed set of
	// dependency selections provided as arguments.
	CreateDependencyLocks(dependencies.CreateDependencyLocks_Request) dependencies.CreateDependencyLocks_Response

	CloseDependencyLocks(dependencies.CloseDependencyLocks_Request) dependencies.CloseDependencyLocks_Response

	// information about the provider version selections in a
	// dependency locks object.
	GetLockedProviderDependencies(dependencies.GetLockedProviderDependencies_Request) dependencies.GetLockedProviderDependencies_Response

	// Populates a new provider plugin cache directory in the local filesystem
	// based on the provider version selections in a given dependency locks
	// object.
	//
	// This particular can only install already-selected provider packages
	// recorded in a dependency locks object; it does not support "upgrading"
	// provider selections to newer versions as a CLI user would do with
	// "terraform init -upgrade", because there would be no way to then
	// commit the updated locks to disk as a lock file.
	BuildProviderPluginCache(dependencies.BuildProviderPluginCache_Request) (stream dependencies.BuildProviderPluginCache_Event)

	// Opens an existing local filesystem directory as a provider plugin cache
	// directory, returning a plugin cache handle that can be used with other
	// operations.
	OpenProviderPluginCache(dependencies.OpenProviderPluginCache_Request) dependencies.OpenProviderPluginCache_Response

	CloseProviderPluginCache(dependencies.CloseProviderPluginCache_Request) dependencies.CloseProviderPluginCache_Response

	// information about the specific provider packages that are
	// available in the given provider plugin cache.
	GetCachedProviders(dependencies.GetCachedProviders_Request) dependencies.GetCachedProviders_Response

	// information about the built-in providers that are compiled in
	// to this Terraform Core server.
	GetBuiltInProviders(dependencies.GetBuiltInProviders_Request) dependencies.GetBuiltInProviders_Response

	// a description of the schema for a particular provider in a
	// given provider plugin cache, or of a particular built-in provider
	// known to this version of Terraform Core.
	//
	// WARNING: This operation requires executing the selected provider plugin,
	// which therefore allows it to run arbitrary code as a child process of
	// this Terraform Core server, with access to all of the same resources.
	// This should typically be used only with providers explicitly selected
	// in a dependency lock file, so users can control what external code
	// has the potential to run in a context that probably has access to
	// private source code and other sensitive information.
	GetProviderSchema(dependencies.GetProviderSchema_Request) dependencies.GetProviderSchema_Response
}

type Packages interface {
	ProviderPackageVersions(packages.ProviderPackageVersions_Request) packages.ProviderPackageVersions_Response
	FetchProviderPackage(packages.FetchProviderPackage_Request) packages.FetchProviderPackage_Response
	ModulePackageVersions(packages.ModulePackageVersions_Request) packages.ModulePackageVersions_Response
	ModulePackageSourceAddr(packages.ModulePackageSourceAddr_Request) packages.ModulePackageSourceAddr_Response
	FetchModulePackage(packages.FetchModulePackage_Request) packages.FetchModulePackage_Response
}

type Stacks interface {
	// Load and perform initial static validation of a stack configuration
	// in a previously-opened source bundle. If successful, returns a
	// stack configuration handle that can be used with other operations.
	OpenStackConfiguration(stacks.OpenStackConfiguration_Request) stacks.OpenStackConfiguration_Response
	// Close a previously-opened stack configuration using its handle.
	CloseStackConfiguration(stacks.CloseStackConfiguration_Request) stacks.CloseStackConfiguration_Response
	// Validate an open stack configuration.
	ValidateStackConfiguration(stacks.ValidateStackConfiguration_Request) stacks.ValidateStackConfiguration_Response
	// Analyze a stack configuration to find all of the components it declares.
	// This is static analysis only, so it cannot produce dynamic information
	// such as the number of instances of each component.
	FindStackConfigurationComponents(stacks.FindStackConfigurationComponents_Request) stacks.FindStackConfigurationComponents_Response
	// Load a stack state by sending a stream of raw state objects that were
	// streamed from a previous ApplyStackChanges response.
	OpenState(stream stacks.OpenStackState_RequestItem) stacks.OpenStackState_Response
	// Close a stack state handle, discarding the associated state.
	CloseState(stacks.CloseStackState_Request) stacks.CloseStackState_Response
	// Calculate a desired state from the given configuration and compare it
	// with the current state to propose a set of changes to converge the
	// current state with the desired state, at least in part.
	PlanStackChanges(stacks.PlanStackChanges_Request) (stream stacks.PlanStackChanges_Event)
	// Load a previously-created plan by sending a stream of raw change objects
	// that were streamed from a previous PlanStackChanges response.
	OpenPlan(stream stacks.OpenStackPlan_RequestItem) stacks.OpenStackPlan_Response
	// Close a saved plan handle, discarding the associated saved plan.
	ClosePlan(stacks.CloseStackPlan_Request) stacks.CloseStackPlan_Response
	// Execute the changes proposed by an earlier call to PlanStackChanges.
	ApplyStackChanges(stacks.ApplyStackChanges_Request) (stream stacks.ApplyStackChanges_Event)
	// OpenStackInspector creates a stack inspector handle that can be used
	// with subsequent calls to the "Inspect"-prefixed functions.
	OpenStackInspector(stacks.OpenStackInspector_Request) stacks.OpenStackInspector_Response
	// InspectExpressionResult evaluates an arbitrary expression in the context
	// of a stack inspector handle.
	InspectExpressionResult(stacks.InspectExpressionResult_Request) stacks.InspectExpressionResult_Response
}

// Stacks1 interface for Terraform plugin operations
type Stacks1 interface {
	// Execute runs a command with the provided arguments and returns the exit code
	Execute(args []string, stdout, stderr io.Writer, dependencies Dependencies, packages Packages, stacks Stacks) int
}
