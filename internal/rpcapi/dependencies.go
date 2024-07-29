// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/apparentlymart/go-versions/versions"
	plugin "github.com/hashicorp/go-plugin"
	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/hashicorp/terraform-svchost/disco"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/terraform/internal/addrs"
	terraformProvider "github.com/hashicorp/terraform/internal/builtin/providers/terraform"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/logging"
	tfplugin "github.com/hashicorp/terraform/internal/plugin"
	tfplugin6 "github.com/hashicorp/terraform/internal/plugin6"
	"github.com/hashicorp/terraform/internal/providercache"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/dependencies"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/hashicorp/terraform/version"
)

type dependenciesServer struct {
	dependencies.UnimplementedDependenciesServer

	handles  *handleTable
	services *disco.Disco
}

func newDependenciesServer(handles *handleTable, services *disco.Disco) *dependenciesServer {
	return &dependenciesServer{
		handles:  handles,
		services: services,
	}
}

func (s *dependenciesServer) OpenSourceBundle(ctx context.Context, req *dependencies.OpenSourceBundle_Request) (*dependencies.OpenSourceBundle_Response, error) {
	localDir := filepath.Clean(req.LocalPath)
	sources, err := sourcebundle.OpenDir(localDir)
	if err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	}
	hnd := s.handles.NewSourceBundle(sources)
	return &dependencies.OpenSourceBundle_Response{
		SourceBundleHandle: hnd.ForProtobuf(),
	}, err
}

func (s *dependenciesServer) CloseSourceBundle(ctx context.Context, req *dependencies.CloseSourceBundle_Request) (*dependencies.CloseSourceBundle_Response, error) {
	hnd := handle[*sourcebundle.Bundle](req.SourceBundleHandle)
	err := s.handles.CloseSourceBundle(hnd)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &dependencies.CloseSourceBundle_Response{}, nil
}

func (s *dependenciesServer) OpenDependencyLockFile(ctx context.Context, req *dependencies.OpenDependencyLockFile_Request) (*dependencies.OpenDependencyLockFile_Response, error) {
	sourcesHnd := handle[*sourcebundle.Bundle](req.SourceBundleHandle)
	sources := s.handles.SourceBundle(sourcesHnd)
	if sources == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid source bundle handle")
	}

	lockFileSource, err := resolveFinalSourceAddr(req.SourceAddress, sources)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid source address: %s", err)
	}

	lockFilePath, err := sources.LocalPathForSource(lockFileSource)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "specified lock file is not available: %s", err)
	}

	locks, diags := depsfile.LoadLocksFromFile(lockFilePath)
	if diags.HasErrors() {
		return &dependencies.OpenDependencyLockFile_Response{
			Diagnostics: diagnosticsToProto(diags),
		}, nil
	}

	locksHnd := s.handles.NewDependencyLocks(locks)
	return &dependencies.OpenDependencyLockFile_Response{
		DependencyLocksHandle: locksHnd.ForProtobuf(),
		Diagnostics:           diagnosticsToProto(diags),
	}, nil
}

func (s *dependenciesServer) CreateDependencyLocks(ctx context.Context, req *dependencies.CreateDependencyLocks_Request) (*dependencies.CreateDependencyLocks_Response, error) {
	locks := depsfile.NewLocks()
	for _, provider := range req.ProviderSelections {
		addr, diags := addrs.ParseProviderSourceString(provider.SourceAddr)
		if diags.HasErrors() {
			return nil, status.Errorf(codes.InvalidArgument, "invalid provider source string %q", provider.SourceAddr)
		}
		version, err := getproviders.ParseVersion(provider.Version)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid version %q for %q: %s", provider.Version, addr.ForDisplay(), err)
		}
		hashes := make([]getproviders.Hash, len(provider.Hashes))
		for i, hashStr := range provider.Hashes {
			hash, err := getproviders.ParseHash(hashStr)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid hash %q for %q: %s", hashStr, addr.ForDisplay(), err)
			}
			hashes[i] = hash
		}

		if existing := locks.Provider(addr); existing != nil {
			return nil, status.Errorf(codes.InvalidArgument, "duplicate entry for provider %q", addr.ForDisplay())
		}

		if !depsfile.ProviderIsLockable(addr) {
			if addr.IsBuiltIn() {
				status.Errorf(codes.InvalidArgument, "cannot lock builtin provider %q", addr.ForDisplay())
			}
			return nil, status.Errorf(codes.InvalidArgument, "provider %q does not support dependency locking", addr.ForDisplay())
		}

		locks.SetProvider(
			addr, version,
			nil, hashes,
		)
	}

	locksHnd := s.handles.NewDependencyLocks(locks)
	return &dependencies.CreateDependencyLocks_Response{
		DependencyLocksHandle: locksHnd.ForProtobuf(),
	}, nil
}

func (s *dependenciesServer) CloseDependencyLocks(ctx context.Context, req *dependencies.CloseDependencyLocks_Request) (*dependencies.CloseDependencyLocks_Response, error) {
	hnd := handle[*depsfile.Locks](req.DependencyLocksHandle)
	err := s.handles.CloseDependencyLocks(hnd)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid dependency locks handle")
	}
	return &dependencies.CloseDependencyLocks_Response{}, nil
}

func (s *dependenciesServer) GetLockedProviderDependencies(ctx context.Context, req *dependencies.GetLockedProviderDependencies_Request) (*dependencies.GetLockedProviderDependencies_Response, error) {
	hnd := handle[*depsfile.Locks](req.DependencyLocksHandle)
	locks := s.handles.DependencyLocks(hnd)
	if locks == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid dependency locks handle")
	}

	providers := locks.AllProviders()
	protoProviders := make([]*terraform1.ProviderPackage, 0, len(providers))
	for _, lock := range providers {
		hashes := lock.PreferredHashes()
		var hashStrs []string
		if len(hashes) != 0 {
			hashStrs = make([]string, len(hashes))
		}
		for i, hash := range hashes {
			hashStrs[i] = hash.String()
		}
		protoProviders = append(protoProviders, &terraform1.ProviderPackage{
			SourceAddr: lock.Provider().String(),
			Version:    lock.Version().String(),
			Hashes:     hashStrs,
		})
	}

	// This is just to make the result be consistent between requests. This
	// _particular_ ordering is not guaranteed to callers.
	sort.Slice(protoProviders, func(i, j int) bool {
		return protoProviders[i].SourceAddr < protoProviders[j].SourceAddr
	})

	return &dependencies.GetLockedProviderDependencies_Response{
		SelectedProviders: protoProviders,
	}, nil
}

func (s *dependenciesServer) BuildProviderPluginCache(req *dependencies.BuildProviderPluginCache_Request, evts dependencies.Dependencies_BuildProviderPluginCacheServer) error {
	ctx := evts.Context()

	hnd := handle[*depsfile.Locks](req.DependencyLocksHandle)
	locks := s.handles.DependencyLocks(hnd)
	if locks == nil {
		return status.Error(codes.InvalidArgument, "invalid dependency locks handle")
	}

	selectors := make([]getproviders.MultiSourceSelector, 0, len(req.InstallationMethods))
	for _, protoMethod := range req.InstallationMethods {
		var source getproviders.Source
		switch arg := protoMethod.Source.(type) {
		case *dependencies.BuildProviderPluginCache_Request_InstallMethod_Direct:
			source = getproviders.NewRegistrySource(s.services)
		case *dependencies.BuildProviderPluginCache_Request_InstallMethod_LocalMirrorDir:
			source = getproviders.NewFilesystemMirrorSource(arg.LocalMirrorDir)
		case *dependencies.BuildProviderPluginCache_Request_InstallMethod_NetworkMirrorUrl:
			u, err := url.Parse(arg.NetworkMirrorUrl)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "invalid network mirror URL %q", arg.NetworkMirrorUrl)
			}
			source = getproviders.NewHTTPMirrorSource(u, s.services.CredentialsSource())
		default:
			// The above should be exhaustive for all variants defined in
			// the protocol buffers schema.
			return status.Errorf(codes.Internal, "unsupported installation method source type %T", arg)
		}

		if len(protoMethod.Include) != 0 || len(protoMethod.Exclude) != 0 {
			return status.Error(codes.InvalidArgument, "include/exclude for installation methods is not yet implemented")
		}

		selectors = append(selectors, getproviders.MultiSourceSelector{
			Source: source,
			// TODO: Deal with the include/exclude options
		})
	}
	instSrc := getproviders.MultiSource(selectors)

	var cacheDir *providercache.Dir
	if req.OverridePlatform == "" {
		cacheDir = providercache.NewDir(req.CacheDir)
	} else {
		platform, err := getproviders.ParsePlatform(req.OverridePlatform)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "invalid overridden platform name %q: %s", req.OverridePlatform, err)
		}
		cacheDir = providercache.NewDirWithPlatform(req.CacheDir, platform)
	}
	inst := providercache.NewInstaller(cacheDir, instSrc)

	// The provider installer was originally built to install providers needed
	// by a configuration/state with reference to a dependency locks object,
	// but the model here is different: we are aiming to install exactly the
	// providers selected in the locks. To get there with the installer as
	// currently designed, we'll build some synthetic provider requirements
	// that call for any version of each of the locked providers, and then
	// the lock file will dictate which version we select.
	wantProviders := locks.AllProviders()
	reqd := make(getproviders.Requirements, len(wantProviders))
	for addr := range wantProviders {
		reqd[addr] = nil
	}

	// We'll translate most events from the provider installer directly into
	// RPC-shaped events, so that the caller can use these to drive
	// progress-reporting UI if needed.
	sentErrorDiags := false
	instEvts := providercache.InstallerEvents{
		PendingProviders: func(reqs map[addrs.Provider]getproviders.VersionConstraints) {
			// This one announces which providers we are expecting to install,
			// which could potentially help drive a percentage-based progress
			// bar or similar in the UI by correlating with the "FetchSuccess"
			// events.
			protoConstraints := make([]*dependencies.BuildProviderPluginCache_Event_ProviderConstraints, 0, len(reqs))
			for addr, constraints := range reqs {
				protoConstraints = append(protoConstraints, &dependencies.BuildProviderPluginCache_Event_ProviderConstraints{
					SourceAddr: addr.ForDisplay(),
					Versions:   getproviders.VersionConstraintsString(constraints),
				})
			}
			evts.Send(&dependencies.BuildProviderPluginCache_Event{
				Event: &dependencies.BuildProviderPluginCache_Event_Pending_{
					Pending: &dependencies.BuildProviderPluginCache_Event_Pending{
						Expected: protoConstraints,
					},
				},
			})
		},
		ProviderAlreadyInstalled: func(provider addrs.Provider, selectedVersion getproviders.Version) {
			evts.Send(&dependencies.BuildProviderPluginCache_Event{
				Event: &dependencies.BuildProviderPluginCache_Event_AlreadyInstalled{
					AlreadyInstalled: &dependencies.BuildProviderPluginCache_Event_ProviderVersion{
						SourceAddr: provider.ForDisplay(),
						Version:    selectedVersion.String(),
					},
				},
			})
		},
		BuiltInProviderAvailable: func(provider addrs.Provider) {
			evts.Send(&dependencies.BuildProviderPluginCache_Event{
				Event: &dependencies.BuildProviderPluginCache_Event_BuiltIn{
					BuiltIn: &dependencies.BuildProviderPluginCache_Event_ProviderVersion{
						SourceAddr: provider.ForDisplay(),
					},
				},
			})
		},
		BuiltInProviderFailure: func(provider addrs.Provider, err error) {
			evts.Send(&dependencies.BuildProviderPluginCache_Event{
				Event: &dependencies.BuildProviderPluginCache_Event_Diagnostic{
					Diagnostic: diagnosticToProto(tfdiags.Sourceless(
						tfdiags.Error,
						"Built-in provider unavailable",
						fmt.Sprintf(
							"Terraform v%s does not support the provider %q.",
							version.SemVer.String(), provider.ForDisplay(),
						),
					)),
				},
			})
			sentErrorDiags = true
		},
		QueryPackagesBegin: func(provider addrs.Provider, versionConstraints getproviders.VersionConstraints, locked bool) {
			evts.Send(&dependencies.BuildProviderPluginCache_Event{
				Event: &dependencies.BuildProviderPluginCache_Event_QueryBegin{
					QueryBegin: &dependencies.BuildProviderPluginCache_Event_ProviderConstraints{
						SourceAddr: provider.ForDisplay(),
						Versions:   getproviders.VersionConstraintsString(versionConstraints),
					},
				},
			})
		},
		QueryPackagesSuccess: func(provider addrs.Provider, selectedVersion getproviders.Version) {
			evts.Send(&dependencies.BuildProviderPluginCache_Event{
				Event: &dependencies.BuildProviderPluginCache_Event_QuerySuccess{
					QuerySuccess: &dependencies.BuildProviderPluginCache_Event_ProviderVersion{
						SourceAddr: provider.ForDisplay(),
						Version:    selectedVersion.String(),
					},
				},
			})
		},
		QueryPackagesWarning: func(provider addrs.Provider, warn []string) {
			evts.Send(&dependencies.BuildProviderPluginCache_Event{
				Event: &dependencies.BuildProviderPluginCache_Event_QueryWarnings{
					QueryWarnings: &dependencies.BuildProviderPluginCache_Event_ProviderWarnings{
						SourceAddr: provider.ForDisplay(),
						Warnings:   warn,
					},
				},
			})
		},
		QueryPackagesFailure: func(provider addrs.Provider, err error) {
			evts.Send(&dependencies.BuildProviderPluginCache_Event{
				Event: &dependencies.BuildProviderPluginCache_Event_Diagnostic{
					Diagnostic: diagnosticToProto(tfdiags.Sourceless(
						tfdiags.Error,
						"Provider is unavailable",
						fmt.Sprintf(
							"Failed to query for provider %s: %s.",
							provider.ForDisplay(),
							tfdiags.FormatError(err),
						),
					)),
				},
			})
			sentErrorDiags = true
		},
		FetchPackageBegin: func(provider addrs.Provider, version getproviders.Version, location getproviders.PackageLocation) {
			evts.Send(&dependencies.BuildProviderPluginCache_Event{
				Event: &dependencies.BuildProviderPluginCache_Event_FetchBegin_{
					FetchBegin: &dependencies.BuildProviderPluginCache_Event_FetchBegin{
						ProviderVersion: &dependencies.BuildProviderPluginCache_Event_ProviderVersion{
							SourceAddr: provider.ForDisplay(),
							Version:    version.String(),
						},
						Location: location.String(),
					},
				},
			})
		},
		FetchPackageSuccess: func(provider addrs.Provider, version getproviders.Version, localDir string, authResult *getproviders.PackageAuthenticationResult) {
			var protoAuthResult dependencies.BuildProviderPluginCache_Event_FetchComplete_AuthResult
			var keyID string
			if authResult != nil {
				keyID = authResult.KeyID
				switch {
				case authResult.SignedByHashiCorp():
					protoAuthResult = dependencies.BuildProviderPluginCache_Event_FetchComplete_OFFICIAL_SIGNED
				default:
					// TODO: The getproviders.PackageAuthenticationResult type
					// only exposes the full detail of the signing outcome as
					// a string intended for direct display in the UI, which
					// means we can't populate this in full detail. For now
					// we'll treat anything signed by a non-HashiCorp key as
					// "unknown" and then rationalize this later.
					protoAuthResult = dependencies.BuildProviderPluginCache_Event_FetchComplete_UNKNOWN
				}
			}
			evts.Send(&dependencies.BuildProviderPluginCache_Event{
				Event: &dependencies.BuildProviderPluginCache_Event_FetchComplete_{
					FetchComplete: &dependencies.BuildProviderPluginCache_Event_FetchComplete{
						ProviderVersion: &dependencies.BuildProviderPluginCache_Event_ProviderVersion{
							SourceAddr: provider.ForDisplay(),
							Version:    version.String(),
						},
						KeyIdForDisplay: keyID,
						AuthResult:      protoAuthResult,
					},
				},
			})
		},
		FetchPackageFailure: func(provider addrs.Provider, version getproviders.Version, err error) {
			evts.Send(&dependencies.BuildProviderPluginCache_Event{
				Event: &dependencies.BuildProviderPluginCache_Event_Diagnostic{
					Diagnostic: diagnosticToProto(tfdiags.Sourceless(
						tfdiags.Error,
						"Failed to fetch provider package",
						fmt.Sprintf(
							"Failed to fetch provider %s v%s: %s.",
							provider.ForDisplay(), version.String(),
							tfdiags.FormatError(err),
						),
					)),
				},
			})
			sentErrorDiags = true
		},
	}
	ctx = instEvts.OnContext(ctx)

	_, err := inst.EnsureProviderVersions(ctx, locks, reqd, providercache.InstallNewProvidersOnly)
	if err != nil {
		// If we already emitted errors in the form of diagnostics then
		// err will typically just duplicate them, so we'll skip emitting
		// another diagnostic in that case.
		if !sentErrorDiags {
			evts.Send(&dependencies.BuildProviderPluginCache_Event{
				Event: &dependencies.BuildProviderPluginCache_Event_Diagnostic{
					Diagnostic: diagnosticToProto(tfdiags.Sourceless(
						tfdiags.Error,
						"Failed to install providers",
						fmt.Sprintf(
							"Cannot install the selected provider plugins: %s.",
							tfdiags.FormatError(err),
						),
					)),
				},
			})
			sentErrorDiags = true
		}
	}

	// "Success" for this RPC just means that the call was valid and we ran
	// to completion. We only return an error for situations that appear to be
	// bugs in the calling program, rather than problems with the installation
	// process.
	return nil
}

func (s *dependenciesServer) OpenProviderPluginCache(ctx context.Context, req *dependencies.OpenProviderPluginCache_Request) (*dependencies.OpenProviderPluginCache_Response, error) {
	var cacheDir *providercache.Dir
	if req.OverridePlatform == "" {
		cacheDir = providercache.NewDir(req.CacheDir)
	} else {
		platform, err := getproviders.ParsePlatform(req.OverridePlatform)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid overridden platform name %q: %s", req.OverridePlatform, err)
		}
		cacheDir = providercache.NewDirWithPlatform(req.CacheDir, platform)
	}

	hnd := s.handles.NewProviderPluginCache(cacheDir)
	return &dependencies.OpenProviderPluginCache_Response{
		ProviderCacheHandle: hnd.ForProtobuf(),
	}, nil
}

func (s *dependenciesServer) CloseProviderPluginCache(ctx context.Context, req *dependencies.CloseProviderPluginCache_Request) (*dependencies.CloseProviderPluginCache_Response, error) {
	hnd := handle[*providercache.Dir](req.ProviderCacheHandle)
	err := s.handles.CloseProviderPluginCache(hnd)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid provider plugin cache handle")
	}
	return &dependencies.CloseProviderPluginCache_Response{}, nil
}

func (s *dependenciesServer) GetCachedProviders(ctx context.Context, req *dependencies.GetCachedProviders_Request) (*dependencies.GetCachedProviders_Response, error) {
	hnd := handle[*providercache.Dir](req.ProviderCacheHandle)
	cacheDir := s.handles.ProviderPluginCache(hnd)
	if cacheDir == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid provider plugin cache handle")
	}

	avail := cacheDir.AllAvailablePackages()
	ret := make([]*terraform1.ProviderPackage, 0, len(avail))
	for addr, pkgs := range avail {
		for _, pkg := range pkgs {
			hash, err := pkg.Hash()
			var protoHashes []string
			// We silently invalid hashes here so we can make a best
			// effort to return as much information as possible, rather
			// than failing if the cache is partially inaccessible.
			// Callers can detect this situation by the hash sequence being
			// empty.
			if err == nil {
				protoHashes = append(protoHashes, hash.String())
			}

			ret = append(ret, &terraform1.ProviderPackage{
				SourceAddr: addr.String(),
				Version:    pkg.Version.String(),
				Hashes:     protoHashes,
			})
		}
	}

	return &dependencies.GetCachedProviders_Response{
		AvailableProviders: ret,
	}, nil
}

func (s *dependenciesServer) GetBuiltInProviders(ctx context.Context, req *dependencies.GetBuiltInProviders_Request) (*dependencies.GetBuiltInProviders_Response, error) {
	ret := make([]*terraform1.ProviderPackage, 0, len(builtinProviders))
	for typeName := range builtinProviders {
		ret = append(ret, &terraform1.ProviderPackage{
			SourceAddr: addrs.NewBuiltInProvider(typeName).ForDisplay(),
		})
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].SourceAddr < ret[j].SourceAddr
	})
	return &dependencies.GetBuiltInProviders_Response{
		AvailableProviders: ret,
	}, nil
}

func (s *dependenciesServer) GetProviderSchema(ctx context.Context, req *dependencies.GetProviderSchema_Request) (*dependencies.GetProviderSchema_Response, error) {
	var cacheHnd handle[*providercache.Dir]
	var cacheDir *providercache.Dir
	if req.GetProviderCacheHandle() != 0 {
		cacheHnd = handle[*providercache.Dir](req.ProviderCacheHandle)
		cacheDir = s.handles.ProviderPluginCache(cacheHnd)
		if cacheDir == nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid provider cache handle")
		}
	}
	// NOTE: cacheDir will be nil if the cache handle was absent. We'll
	// check that below once we know if the requested provider is a built-in.

	var err error

	providerAddr, diags := addrs.ParseProviderSourceString(req.ProviderAddr)
	if diags.HasErrors() {
		return nil, status.Error(codes.InvalidArgument, "invalid provider source address syntax")
	}
	var providerVersion getproviders.Version
	if req.ProviderVersion != "" {
		if providerAddr.IsBuiltIn() {
			return nil, status.Errorf(codes.InvalidArgument, "can't specify version for built-in provider")
		}
		providerVersion, err = getproviders.ParseVersion(req.ProviderVersion)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid provider version string: %s", err)
		}
	}

	// For non-builtin providers the caller MUST provide a provider cache
	// handle. For built-in providers it's optional.
	if cacheHnd.IsNil() && !providerAddr.IsBuiltIn() {
		return nil, status.Errorf(codes.InvalidArgument, "provider cache handle is required for non-builtin provider")
	}

	schemaResp, err := loadProviderSchema(providerAddr, providerVersion, cacheDir)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &dependencies.GetProviderSchema_Response{
		Schema: providerSchemaToProto(schemaResp),
	}, nil
}

func resolveFinalSourceAddr(protoSourceAddr *terraform1.SourceAddress, sources *sourcebundle.Bundle) (sourceaddrs.FinalSource, error) {
	sourceAddr, err := sourceaddrs.ParseSource(protoSourceAddr.Source)
	if err != nil {
		return nil, fmt.Errorf("invalid location: %w", err)
	}
	var allowedVersions versions.Set
	if sourceAddr.SupportsVersionConstraints() {
		allowedVersions, err = versions.MeetingConstraintsStringRuby(protoSourceAddr.Versions)
		if err != nil {
			return nil, fmt.Errorf("invalid version constraints: %w", err)
		}
	} else {
		if protoSourceAddr.Versions != "" {
			return nil, fmt.Errorf("can't use version constraints with this source type")
		}
	}

	switch sourceAddr := sourceAddr.(type) {
	case sourceaddrs.FinalSource:
		// Easy case: it's already a final source so we can just return it.
		return sourceAddr, nil
	case sourceaddrs.RegistrySource:
		// Turning a RegistrySource into a final source means we need to
		// figure out which exact version the source address is selecting.
		availableVersions := sources.RegistryPackageVersions(sourceAddr.Package())
		selectedVersion := availableVersions.NewestInSet(allowedVersions)
		return sourceAddr.Versioned(selectedVersion), nil
	default:
		// Should not get here; if sourceaddrs gets any new non-final source
		// types in future then we ought to add a cases for them above at the
		// same time as upgrading the go-slug dependency.
		return nil, fmt.Errorf("unsupported source address type %T (this is a bug in Terraform)", sourceAddr)
	}
}

// builtinProviders provides the instantiation functions for each of the
// built-in providers that are available when using Terraform Core through
// its RPC API.
//
// TODO: Prior to the RPC API the built-in providers were architecturally
// the responsibility of Terraform CLI, which is a bit strange and means
// we can't readily share this definition with the CLI-driven usage patterns.
// In future it would be nice to factor out the table of built-in providers
// into a common location that both can share, or ideally change Terraform CLI
// to consume this RPC API through an internal API bridge so that the
// architectural divide between CLI and Core is more explicit.
var builtinProviders map[string]func() providers.Interface

func init() {
	builtinProviders = map[string]func() providers.Interface{
		"terraform": func() providers.Interface {
			return terraformProvider.NewProvider()
		},
	}
}

// providerFactoriesForLocks builds a map of factory functions for all of the
// providers selected by the given locks and also all of the built-in providers.
//
// Non-builtin providers are assumed to be plugins available in the given
// plugin cache directory. pluginsDir can be nil if and only if the given
// locks is empty of provider selections, in which case the result contains
// only the built-in providers.
//
// If any of the selected providers are not available as plugins in the cache
// directory, returns an error describing a problem with at least one of
// of them.
func providerFactoriesForLocks(locks *depsfile.Locks, pluginsDir *providercache.Dir) (map[addrs.Provider]providers.Factory, error) {
	var err error
	ret := make(map[addrs.Provider]providers.Factory)
	for name, infallibleFactory := range builtinProviders {
		infallibleFactory := infallibleFactory // each iteration must have its own symbol
		ret[addrs.NewBuiltInProvider(name)] = func() (providers.Interface, error) {
			return infallibleFactory(), nil
		}
	}
	selectedProviders := locks.AllProviders()
	if pluginsDir == nil {
		if len(selectedProviders) != 0 {
			return nil, fmt.Errorf("only built-in providers are available without a plugin cache directory")
		}
		return ret, nil // just the built-in providers then
	}

	for addr, lock := range selectedProviders {
		addr := addr
		lock := lock

		selectedVersion := lock.Version()
		cached := pluginsDir.ProviderVersion(addr, selectedVersion)
		if cached == nil {
			err = errors.Join(err, fmt.Errorf("plugin cache directory does not contain %s v%s", addr, selectedVersion))
			continue
		}

		// The cached package must match at least one of the locked
		// package checksums.
		matchesChecksums, checksumErr := cached.MatchesAnyHash(lock.PreferredHashes())
		if checksumErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to calculate checksum for cached %s v%s: %w", addr, selectedVersion, checksumErr))
			continue
		}
		if !matchesChecksums {
			err = errors.Join(err, fmt.Errorf("cached package for %s v%s does not match any of the locked checksums", addr, selectedVersion))
			continue
		}

		exeFilename, exeErr := cached.ExecutableFile()
		if exeErr != nil {
			err = errors.Join(err, fmt.Errorf("unusuable cached package for %s v%s: %w", addr, selectedVersion, exeErr))
			continue
		}

		ret[addr] = func() (providers.Interface, error) {
			config := &plugin.ClientConfig{
				HandshakeConfig:  tfplugin.Handshake,
				Logger:           logging.NewProviderLogger(""),
				AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
				Managed:          true,
				Cmd:              exec.Command(exeFilename),
				AutoMTLS:         true,
				VersionedPlugins: tfplugin.VersionedPlugins,
				SyncStdout:       logging.PluginOutputMonitor(fmt.Sprintf("%s:stdout", addr)),
				SyncStderr:       logging.PluginOutputMonitor(fmt.Sprintf("%s:stderr", addr)),
			}

			client := plugin.NewClient(config)
			rpcClient, err := client.Client()
			if err != nil {
				return nil, err
			}

			raw, err := rpcClient.Dispense(tfplugin.ProviderPluginName)
			if err != nil {
				return nil, err
			}

			protoVer := client.NegotiatedVersion()
			switch protoVer {
			case 5:
				p := raw.(*tfplugin.GRPCProvider)
				p.PluginClient = client
				p.Addr = addr
				return p, nil
			case 6:
				p := raw.(*tfplugin6.GRPCProvider)
				p.PluginClient = client
				p.Addr = addr
				return p, nil
			default:
				panic("unsupported protocol version")
			}
		}
	}
	return ret, err
}
