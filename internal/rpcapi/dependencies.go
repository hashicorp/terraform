package rpcapi

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type dependenciesServer struct {
	terraform1.UnimplementedDependenciesServer

	handles *handleTable
}

func newDependenciesServer(handles *handleTable) *dependenciesServer {
	return &dependenciesServer{
		handles: handles,
	}
}

func (s *dependenciesServer) OpenSourceBundle(ctx context.Context, req *terraform1.OpenSourceBundle_Request) (*terraform1.OpenSourceBundle_Response, error) {
	localDir := filepath.Clean(req.LocalPath)
	sources, err := sourcebundle.OpenDir(localDir)
	if err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	}
	hnd := s.handles.NewSourceBundle(sources)
	return &terraform1.OpenSourceBundle_Response{
		SourceBundleHandle: hnd.ForProtobuf(),
	}, err
}

func (s *dependenciesServer) CloseSourceBundle(ctx context.Context, req *terraform1.CloseSourceBundle_Request) (*terraform1.CloseSourceBundle_Response, error) {
	hnd := handle[*sourcebundle.Bundle](req.SourceBundleHandle)
	err := s.handles.CloseSourceBundle(hnd)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &terraform1.CloseSourceBundle_Response{}, nil
}

func (s *dependenciesServer) OpenDependencyLockFile(ctx context.Context, req *terraform1.OpenDependencyLockFile_Request) (*terraform1.OpenDependencyLockFile_Response, error) {
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
		return &terraform1.OpenDependencyLockFile_Response{
			Diagnostics: diagnosticsToProto(diags),
		}, nil
	}

	locksHnd := s.handles.NewDependencyLocks(locks)
	return &terraform1.OpenDependencyLockFile_Response{
		DependencyLocksHandle: locksHnd.ForProtobuf(),
		Diagnostics:           diagnosticsToProto(diags),
	}, nil
}

func (s *dependenciesServer) CloseDependencyLocks(ctx context.Context, req *terraform1.CloseDependencyLocks_Request) (*terraform1.CloseDependencyLocks_Response, error) {
	hnd := handle[*depsfile.Locks](req.DependencyLocksHandle)
	err := s.handles.CloseDependencyLocks(hnd)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid dependency locks handle")
	}
	return &terraform1.CloseDependencyLocks_Response{}, nil
}

func (s *dependenciesServer) GetLockedProviderDependencies(ctx context.Context, req *terraform1.GetLockedProviderDependencies_Request) (*terraform1.GetLockedProviderDependencies_Response, error) {
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

	return &terraform1.GetLockedProviderDependencies_Response{
		SelectedProviders: protoProviders,
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
