// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/hashicorp/terraform-svchost/disco"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getmodules"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/providercache"
	"github.com/hashicorp/terraform/internal/registry"
	"github.com/hashicorp/terraform/internal/registry/regsrc"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/packages"
)

var _ packages.PackagesServer = (*packagesServer)(nil)

func newPackagesServer(services *disco.Disco) *packagesServer {
	return &packagesServer{
		services: services,

		// This function lets us control the provider source during tests.
		providerSourceFn: func(services *disco.Disco) getproviders.Source {
			// TODO: Implement loading from alternate sources like network or filesystem
			//  mirrors.
			return getproviders.NewRegistrySource(services)
		},
	}
}

type providerSourceFn func(services *disco.Disco) getproviders.Source

type packagesServer struct {
	packages.UnimplementedPackagesServer

	services         *disco.Disco
	providerSourceFn providerSourceFn
}

func (p *packagesServer) ProviderPackageVersions(ctx context.Context, request *packages.ProviderPackageVersions_Request) (*packages.ProviderPackageVersions_Response, error) {
	response := new(packages.ProviderPackageVersions_Response)

	source := p.providerSourceFn(p.services)
	provider, diags := addrs.ParseProviderSourceString(request.SourceAddr)
	response.Diagnostics = append(response.Diagnostics, diagnosticsToProto(diags)...)
	if diags.HasErrors() {
		return response, nil
	}

	versions, warnings, err := source.AvailableVersions(ctx, provider)

	displayWarnings := make([]string, len(warnings))
	for ix, warning := range warnings {
		displayWarnings[ix] = fmt.Sprintf("- %s", warning)
	}
	if len(displayWarnings) > 0 {
		response.Diagnostics = append(response.Diagnostics, &terraform1.Diagnostic{
			Severity: terraform1.Diagnostic_WARNING,
			Summary:  "Additional provider information from registry",
			Detail:   fmt.Sprintf("The remote registry returned warnings for %s:\n%s", provider.ForDisplay(), strings.Join(displayWarnings, "\n")),
		})
	}

	if err != nil {
		// TODO: Parse the different error types so we can provide specific
		//  error diagnostics, see commands/init.go:621.
		response.Diagnostics = append(response.Diagnostics, &terraform1.Diagnostic{
			Severity: terraform1.Diagnostic_ERROR,
			Summary:  "Failed to query available provider packages",
			Detail:   fmt.Sprintf("Could not retrieve the list of available versions for provider %s: %s.", provider.ForDisplay(), err),
		})
		return response, nil
	}

	for _, version := range versions {
		response.Versions = append(response.Versions, version.String())
	}
	return response, nil
}

func (p *packagesServer) FetchProviderPackage(ctx context.Context, request *packages.FetchProviderPackage_Request) (*packages.FetchProviderPackage_Response, error) {

	response := new(packages.FetchProviderPackage_Response)

	version, err := versions.ParseVersion(request.Version)
	if err != nil {
		response.Diagnostics = append(response.Diagnostics, &terraform1.Diagnostic{
			Severity: terraform1.Diagnostic_ERROR,
			Summary:  "Invalid platform",
			Detail:   fmt.Sprintf("The requested version %s is invalid: %s.", request.Version, err),
		})
		return response, nil
	}

	source := p.providerSourceFn(p.services)
	provider, diags := addrs.ParseProviderSourceString(request.SourceAddr)
	response.Diagnostics = append(response.Diagnostics, diagnosticsToProto(diags)...)
	if diags.HasErrors() {
		return response, nil
	}

	var allowedHashes []getproviders.Hash
	for _, hash := range request.Hashes {
		allowedHashes = append(allowedHashes, getproviders.Hash(hash))
	}

	for _, requestPlatform := range request.Platforms {
		result := new(packages.FetchProviderPackage_PlatformResult)
		response.Results = append(response.Results, result)

		platform, err := getproviders.ParsePlatform(requestPlatform)
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, &terraform1.Diagnostic{
				Severity: terraform1.Diagnostic_ERROR,
				Summary:  "Invalid platform",
				Detail:   fmt.Sprintf("The requested platform %s is invalid: %s.", requestPlatform, err),
			})
			continue
		}

		meta, err := source.PackageMeta(ctx, provider, version, platform)
		if err != nil {
			// TODO: Parse the different error types so we can provide specific
			//  error diagnostics, see commands/init.go:731.
			result.Diagnostics = append(result.Diagnostics, &terraform1.Diagnostic{
				Severity: terraform1.Diagnostic_ERROR,
				Summary:  "Failed to query provider package metadata",
				Detail:   fmt.Sprintf("Could not retrieve package metadata for provider %s@%s for %s: %s.", provider.ForDisplay(), version.String(), platform.String(), err),
			})
			continue
		}

		into := providercache.NewDirWithPlatform(request.CacheDir, platform)
		authResult, err := into.InstallPackage(ctx, meta, allowedHashes)
		if err != nil {
			// TODO: Parse the different error types so we can provide specific
			//  error diagnostics, see commands/init.go:731.
			result.Diagnostics = append(result.Diagnostics, &terraform1.Diagnostic{
				Severity: terraform1.Diagnostic_ERROR,
				Summary:  "Failed to download provider package",
				Detail:   fmt.Sprintf("Could not download provider %s@%s for %s: %s.", provider.ForDisplay(), version.String(), platform.String(), err),
			})
			continue
		}

		var hashes []string
		if authResult.SignedByAnyParty() {
			for _, hash := range meta.AcceptableHashes() {
				hashes = append(hashes, string(hash))
			}
		}

		providerPackage := into.ProviderVersion(provider, version)
		hash, err := providerPackage.Hash()
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, &terraform1.Diagnostic{
				Severity: terraform1.Diagnostic_ERROR,
				Summary:  "Failed to hash provider package",
				Detail:   fmt.Sprintf("Could not hash provider %s@%s for %s: %s.", provider.ForDisplay(), version.String(), platform.String(), err),
			})
			continue
		}
		hashes = append(hashes, string(hash))
		result.Provider = &terraform1.ProviderPackage{
			SourceAddr: request.SourceAddr,
			Version:    request.Version,
			Hashes:     hashes,
		}
	}

	return response, nil
}

func (p *packagesServer) ModulePackageVersions(ctx context.Context, request *packages.ModulePackageVersions_Request) (*packages.ModulePackageVersions_Response, error) {
	response := new(packages.ModulePackageVersions_Response)

	module, err := regsrc.ParseModuleSource(request.SourceAddr)
	if err != nil {
		response.Diagnostics = append(response.Diagnostics, &terraform1.Diagnostic{
			Severity: terraform1.Diagnostic_ERROR,
			Summary:  "Invalid module source",
			Detail:   fmt.Sprintf("Module source %s is invalid: %s.", request.SourceAddr, err),
		})
		return response, nil
	}

	client := registry.NewClient(p.services, nil)
	versions, err := client.ModuleVersions(ctx, module)
	if err != nil {
		response.Diagnostics = append(response.Diagnostics, &terraform1.Diagnostic{
			Severity: terraform1.Diagnostic_ERROR,
			Summary:  "Failed to query available module packages",
			Detail:   fmt.Sprintf("Could not retrieve the list of available modules for module %s: %s.", module.Display(), err),
		})
		return response, nil
	}

	for _, module := range versions.Modules {
		for _, version := range module.Versions {
			response.Versions = append(response.Versions, version.Version)
		}
	}

	return response, nil
}

func (p *packagesServer) ModulePackageSourceAddr(ctx context.Context, request *packages.ModulePackageSourceAddr_Request) (*packages.ModulePackageSourceAddr_Response, error) {
	response := new(packages.ModulePackageSourceAddr_Response)

	module, err := regsrc.ParseModuleSource(request.SourceAddr)
	if err != nil {
		response.Diagnostics = append(response.Diagnostics, &terraform1.Diagnostic{
			Severity: terraform1.Diagnostic_ERROR,
			Summary:  "Invalid module source",
			Detail:   fmt.Sprintf("Module source %s is invalid: %s.", request.SourceAddr, err),
		})
		return response, nil
	}

	client := registry.NewClient(p.services, nil)
	location, err := client.ModuleLocation(ctx, module, request.Version)
	if err != nil {
		response.Diagnostics = append(response.Diagnostics, &terraform1.Diagnostic{
			Severity: terraform1.Diagnostic_ERROR,
			Summary:  "Failed to query module package metadata",
			Detail:   fmt.Sprintf("Could not retrieve package metadata for provider %s at %s: %s.", module.Display(), request.Version, err),
		})
		return response, nil
	}
	response.Url = location

	return response, nil
}

func (p *packagesServer) FetchModulePackage(ctx context.Context, request *packages.FetchModulePackage_Request) (*packages.FetchModulePackage_Response, error) {
	response := new(packages.FetchModulePackage_Response)

	fetcher := getmodules.NewPackageFetcher()
	if err := fetcher.FetchPackage(ctx, request.CacheDir, request.Url); err != nil {
		response.Diagnostics = append(response.Diagnostics, &terraform1.Diagnostic{
			Severity: terraform1.Diagnostic_ERROR,
			Summary:  "Failed to download module package",
			Detail:   fmt.Sprintf("Could not download provider from %s: %s.", request.Url, err),
		})
		return response, nil
	}

	return response, nil
}
