package sourcebundle

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"golang.org/x/mod/sumdb/dirhash"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getmodules"
	"github.com/hashicorp/terraform/internal/registry"
	"github.com/hashicorp/terraform/internal/registry/regsrc"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Builder builds source bundles containing zero or more explicitly-added
// module sources, along with zero or more other module packages that the
// given module sources depend on.
type Builder struct {
	// These are values or dependencies provided by the caller.
	workDir string
	fetcher *getmodules.PackageFetcher
	reg     *registry.Client

	// manifest is where we remember what we've got in our source bundle.
	// Callers can retrieve a reference to this object and then save it
	// for later use when loading content from the source bundle.
	manifest *Manifest

	// These fields are caches of information we generate during building.
	knownPackageHashes      map[string]struct{}
	registryPackageVersions map[addrs.ModuleRegistryPackage]versions.List
}

// NewBuilder constructs a new builder with a given filesystem directory as
// its working directory.
//
// The working directory should already exist, be a directory, and should be
// empty at the point of calling this function. Any content added to this
// directory through actions on the builder is an implementation detail
// which external callers should not depend on. The directory must not be
// deleted for as long as the resulting Builder is still alive, but may be
// deleted after all pointers to it are out of scope.
//
// packageFetcher and registryClient must both be valid clients that are
// ready to use. Callers should configure packageFetcher so that it can only
// install from desirable source types; Builder does not constrain in any way
// which source types are accepted.
func NewBuilder(workDir string, packageFetcher *getmodules.PackageFetcher, registryClient *registry.Client) *Builder {
	return &Builder{
		workDir: workDir,
		fetcher: packageFetcher,
		reg:     registryClient,

		manifest: newManifest(),

		knownPackageHashes:      make(map[string]struct{}),
		registryPackageVersions: make(map[addrs.ModuleRegistryPackage]versions.List),
	}
}

// Manifest returns the builder's manifest.
//
// The builder will continue to mutate the returned object if any new module
// packages are added after this call, so it's unsafe to use the returned
// object concurrently with any other methods of this [Builder].
func (b *Builder) Manifest() *Manifest {
	return b.manifest
}

// AddRemoteSource adds a new remote source to the builder, retrieving its
// containing module package if it isn't already known to the builder. It will
// also pass the resulting directory to the given [DependencyFinder] to
// recursively resolve any dependencies of the given source.
//
// AddRemoteSource is not concurrency-safe. Don't call it concurrently with
// other calls or with calls to [Builder.AddRegistrySource].
//
// AddRemoteSource attempts to avoid re-fetching the same module package
// multiple times when it's requested in different parts of the dependency tree,
// but Terraform module source addresses are not designed to be inherently
// comparable and so this is a best-effort optimization rather than a guarantee.
// The result should be equivalent anyway, and the builder will just waste a
// little time re-fetching the same source code.
//
// If there are any errors either fetching the source package, finding its
// dependencies, or internally in updating the builder's working directory then
// AddSource will return error diagnostics describing the problem. If AddSource
// returns error diagnostics then the [Builder] may be left in an inconsistent
// state and so should not be used any further.
func (b *Builder) AddRemoteSource(ctx context.Context, src addrs.ModuleSourceRemote, srcRng *tfdiags.SourceRange, depFinder DependencyFinder) tfdiags.Diagnostics {
	// we preallocate some queue buffers so that in many cases we can avoid
	// reallocating these slices throughout, unless one of the downstream
	// modules has a large number of dependencies itself.
	remoteQueue := make([]remoteQueueItem, 1, 8)
	regQueue := make([]registryQueueItem, 0, 8)
	remoteQueue[0] = remoteQueueItem{
		sourceAddr: src,
		depFinder:  depFinder,
		srcRng:     srcRng,
	}
	return b.addPackages(ctx, regQueue, remoteQueue)
}

// AddRegistrySource adds a new remote source to the builder, retrieving its
// containing module package if it isn't already known to the builder. It will
// also pass the resulting directory to the given [DependencyFinder] to
// recursively resolve any dependencies of the given source.
//
// Because registry module addresses are just an indirection over remote
// module source addresses, this method asks the relevant registry to return
// the real source address and then internally calls [Builder.AddRemoteSource]
// to add it. The same considerations as that method therefore also apply to
// this one.
func (b *Builder) AddRegistrySource(ctx context.Context, src addrs.ModuleSourceRegistry, allowedVersions versions.Set, srcRng *tfdiags.SourceRange, depFinder DependencyFinder) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	remoteSrc, moreDiags := b.findRegistryModulePackage(ctx, src.Package, allowedVersions, srcRng)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return diags
	}

	// We need to adjust the remote source address to take into account any
	// additional subdirectory component that was present in "src". This
	// is important because the registry is allowed to return a subdir part
	// itself, and so the two subdir portions must be combined together if so.
	remoteSrc = remoteSrc.FromRegistry(src)

	moreDiags = b.AddRemoteSource(ctx, remoteSrc, srcRng, depFinder)
	diags = diags.Append(moreDiags)
	return diags
}

// addPackages is the main mechanism of the builder, which deals with
// everything in the given "queue" slices, but also appends new items to the
// queues as they are discovered and then depletes them to empty.
//
// After calling this method it exclusively owns the backing arrays of the
// two "queue" slices and will make arbitrary modifications to them.
func (b *Builder) addPackages(ctx context.Context, regQueue []registryQueueItem, remoteQueue []remoteQueueItem) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// We'll keep iterating until both of our queues are empty.
	for len(regQueue) > 0 && len(remoteQueue) > 0 {

		// First we'll deal with registry queue items, because each iteration
		// of this will generate another entry in pkgQueue which we can then
		// process all together afterwards.
		for len(regQueue) > 0 {
			// Although we've called it a "queue", the processing order doesn't
			// actually matter here so we're going to consume it in LIFO order
			// (like a stack) to make more efficient use of the slice.
			item := regQueue[len(regQueue)-1]
			regQueue = regQueue[:len(regQueue)-1]

			remoteSrc, moreDiags := b.findRegistryModulePackage(ctx, item.sourceAddr.Package, item.allowedVersions, item.srcRng)
			diags = diags.Append(moreDiags)
			if diags.HasErrors() {
				return diags
			}

			// We need to adjust the remote source address to take into account any
			// additional subdirectory component that was present in "src". This
			// is important because the registry is allowed to return a subdir part
			// itself, and so the two subdir portions must be combined together if so.
			remoteSrc = remoteSrc.FromRegistry(item.sourceAddr)

			// The underlying real remote source address now transfers directly
			// into our "remote queue" for processing in the other loop below.
			remoteQueue = append(remoteQueue, remoteQueueItem{
				sourceAddr: remoteSrc,
				depFinder:  item.depFinder,
				srcRng:     item.srcRng,
			})
		}

		// By the time we get here we probably have a mixture of both
		// directly-detected remote sources and those we detected indirectly
		// through a module registry. We deal with them both in the same way.
		for len(remoteQueue) > 0 {
			// Although we've called it a "queue", the processing order doesn't
			// actually matter here so we're going to consume it in LIFO order
			// (like a stack) to make more efficient use of the slice.
			item := remoteQueue[len(remoteQueue)-1]
			remoteQueue = remoteQueue[:len(remoteQueue)-1]

			localDir, moreDiags := b.fetchRemotePackage(ctx, item.sourceAddr.Package, item.srcRng, item.depFinder)
			diags = diags.Append(moreDiags)
			if diags.HasErrors() {
				return diags
			}
		}

	}

	return diags
}

func (b *Builder) findRegistryModulePackage(ctx context.Context, pkgAddr addrs.ModuleRegistryPackage, allowedVersions versions.Set, srcRng *tfdiags.SourceRange) (addrs.ModuleSourceRemote, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// The registry client has its own address type, so we need to convert.
	// Note that the module registry has some historical bad naming: it uses
	// the term "module" but it's actually an index of module _packages_.
	regsrcAddr := regsrc.ModuleFromRegistryPackageAddr(pkgAddr)

	var hclRng *hcl.Range
	if srcRng != nil {
		hclRng = srcRng.ToHCL().Ptr()
	}

	var vs versions.List
	if known, isKnown := b.registryPackageVersions[pkgAddr]; isKnown {
		vs = known
	} else {
		resp, err := b.reg.ModuleVersions(ctx, regsrcAddr)
		if err != nil {
			if registry.IsModuleNotFound(err) {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Module not found",
					Detail:   fmt.Sprintf("Module package %q doesn't exist in module registry %s.", pkgAddr, pkgAddr.Host),
					Subject:  hclRng,
				})
			} else if errors.Is(err, context.Canceled) {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Module installation was interrupted",
					Detail:   fmt.Sprintf("Received interrupt signal while retrieving available versions for registry module package %q.", pkgAddr),
				})
			} else {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Error accessing remote module registry",
					Detail:   fmt.Sprintf("Failed to retrieve available versions for module package %q from %s: %s.", pkgAddr, pkgAddr.Host, err),
					Subject:  hclRng,
				})
			}
			return addrs.ModuleSourceRemote{}, diags
		}
		if len(resp.Modules) > 0 && len(resp.Modules[0].Versions) > 0 {
			vs = make(versions.List, len(resp.Modules[0].Versions))
			for i, apiVersion := range resp.Modules[0].Versions {
				v, err := versions.ParseVersion(apiVersion.Version)
				if err != nil {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid response from module registry",
						Detail:   fmt.Sprintf("Module registry %s returned invalid version number %q: %s.", pkgAddr.Host, apiVersion.Version, err),
						Subject:  hclRng,
					})
					return addrs.ModuleSourceRemote{}, diags
				}
				vs[i] = v
			}
			vs.Sort()
		}
	}

	selectedVersion := vs.NewestInSet(allowedVersions)
	if selectedVersion == versions.Unspecified {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Module not found",
			Detail:   fmt.Sprintf("Module package %q exists in the module registry, but no versions match the specified version constraint.", pkgAddr),
			Subject:  hclRng,
		})
		return addrs.ModuleSourceRemote{}, diags
	}

	if known, isKnown := b.manifest.getRegistryPackageSource(pkgAddr, selectedVersion); isKnown {
		return known, nil
	}

	realAddrRaw, err := b.reg.ModuleLocation(ctx, regsrcAddr, selectedVersion.String())
	if err != nil {
		// It's weird to get here, because we'd only be requesting
		// selectedVersion if it was previously reported by the registry as
		// avaiable, but perhaps the registry index changed while we were
		// running?
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Module not found",
			Detail:   fmt.Sprintf("Failed to retrieve a download URL for %s %s from %s: %s", pkgAddr, selectedVersion, pkgAddr.Host, err),
			Subject:  hclRng,
		})
		return addrs.ModuleSourceRemote{}, diags
	}

	realAddr, err := addrs.ParseModuleSource(realAddrRaw)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid package location from module registry",
			Detail:   fmt.Sprintf("Module registry %s returned invalid source location %q for %s %s: %s.", pkgAddr.Host, realAddrRaw, pkgAddr, selectedVersion, err),
		})
		return addrs.ModuleSourceRemote{}, diags
	}
	switch realAddr := realAddr.(type) {
	// Only a remote source address is allowed here: a registry isn't
	// allowed to return a local path (because it doesn't know what
	// its being called from) and we also don't allow recursively pointing
	// at another registry source for simplicity's sake.
	case addrs.ModuleSourceRemote:
		b.manifest.saveRegistryPackageSource(pkgAddr, selectedVersion, realAddr)
		return realAddr, diags
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid package location from module registry",
			Detail:   fmt.Sprintf("Module registry %s returned invalid source location %q for %s %s: must be a direct remote package address.", pkgAddr.Host, realAddrRaw, pkgAddr, selectedVersion),
		})
		return addrs.ModuleSourceRemote{}, diags
	}
}

// fetchRemotePackage ensures that the given remote module package is present
// in the builder's work directory, and returns the name of the bundle
// subdirectory it's been installed to.
//
// If the given package address has been requested before, or if another
// package address yielding identical source code was requested before, then
// the returned subdirectory will be shared with those other requests. This
// in particular means that if everything is coming from the same monorepo
// then we should end up storing only one copy of its contents, no matter
// how many times it is mentioned with different sub-paths.
func (b *Builder) fetchRemotePackage(ctx context.Context, pkgAddr addrs.ModulePackage, srcRng *tfdiags.SourceRange, depFinder DependencyFinder) (string, tfdiags.Diagnostics) {
	// We retrieve remote module packages using go-getter internally, since
	// Terraform's remote module package addresses are really just go-getter
	// addresses.

	var diags tfdiags.Diagnostics
	var hclRng *hcl.Range
	if srcRng != nil {
		hclRng = srcRng.ToHCL().Ptr()
	}

	// TODO: Check if we already have a local copy of this source's module
	// package.

	// We'll first retrieve each package into a temporary directory so that we
	// can tidy it and analyze it a bit before moving it to its final directory
	// name. We only want to keep the files that are actually relevant to
	// Terraform, and want to detect if we end up retrieving the same source
	// package twice (due to slightly-different source addresses that are
	// actually equivalent) so that in situations like a big monorepo we can
	// minimize the chance of ending up storing the whole monorepo multiple
	// times.
	tmpDir, err := ioutil.TempDir(b.workDir, "installtemp-")
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to create temporary directory",
			fmt.Sprintf("Failed to create directory under %q for module package installation: %s.", b.workDir, err),
		))
		return "", diags
	}

	err = b.fetcher.FetchPackage(ctx, tmpDir, pkgAddr.String())
	if err != nil {
		// FIXME: go-getter generates a poor error for an invalid relative
		// path, so we should generate a better error if err is of type
		// *getmodules.MaybeRelativePathErr. See package initwd for
		// an example.

		// Errors returned by go-getter have very inconsistent quality as
		// end-user error messages, but for now we're accepting that because
		// we have no way to recognize any specific errors to improve them
		// and masking the error entirely would hide valuable diagnostic
		// information from the user.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to fetch module package",
			Detail:   fmt.Sprintf("Could not download module package %q: %s.", pkgAddr, err),
			Subject:  hclRng,
		})
		return "", diags
	}

	// We'll delete from the package any paths that are matched by the
	// .terraformignore file, if present, or by our default ignore rules.
	// This also validates the contents to make sure we only have regular
	// files, directories, and symlinks only within the package root.
	ignoreRules, err := parseIgnoreFile(tmpDir)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid remote module package",
			Detail:   fmt.Sprintf("Could not load the .terraformignore rules from the remote module package: %s.", err),
			Subject:  hclRng,
		})
		return "", diags
	}
	// NOTE: The checks in packagePrepareWalkFn are safe only if we are sure
	// that no other process is concurrently modifying our temporary directory.
	// Source bundle building should only occur on hosts that are trusted by
	// whoever will ultimately be using the generated bundle.
	err = filepath.Walk(tmpDir, packagePrepareWalkFn(tmpDir, ignoreRules))
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid remote module package",
			Detail:   fmt.Sprintf("Module package from %q has invalid contents: %s.", pkgAddr, err),
			Subject:  hclRng,
		})
		return "", diags
	}

	// If we got here then our tmpDir contains the final source code of a valid
	// module package. We'll compute a hash of its contents so we can notice
	// if it is identical to some other package we already installed, and then
	// if not rename it into its final directory name.
	// For this purpose we reuse the same directory tree hashing scheme that
	// Go uses for its own modules, although that's an implementation detail
	// subject to change in future versions: callers should always resolve
	// paths through the source bundle's manifest rather than assuming a path.
	hash, err := dirhash.HashDir(tmpDir, "", dirhash.Hash1)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to calculate module package checksum",
			Detail:   fmt.Sprintf("Cannot calculate checksum for contents of package %q: %s.", pkgAddr, err),
			Subject:  hclRng,
		})
		return "", diags
	}

	finalPath := filepath.Join(b.workDir, hash)

	if _, exists := b.knownPackageHashes[hash]; exists {
		// We already fetched the same package content from another location,
		// so we'll just discard this directory and use the earlier one.
		err := os.RemoveAll(tmpDir)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to clean up temporary directory",
				fmt.Sprintf("Failed to clean up module package working directory %q: %s.", tmpDir, err),
			))
			return "", diags
		}
	} else {
		// We've not encountered this package yet, so we'll rename the
		// temporary directory into its final location within the bundle.
		err := os.Rename(tmpDir, finalPath)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to prepare module package directory",
				fmt.Sprintf("Failed to move module package %q to its final bundle location: %s.", pkgAddr, err),
			))
			return "", diags
		}
	}

	b.manifest.saveModulePackageBundleDir(pkgAddr, hash)

	return hash, diags
}

type registryQueueItem struct {
	sourceAddr      addrs.ModuleSourceRegistry
	allowedVersions versions.Set
	depFinder       DependencyFinder
	srcRng          *tfdiags.SourceRange
}

type remoteQueueItem struct {
	sourceAddr addrs.ModuleSourceRemote
	depFinder  DependencyFinder
	srcRng     *tfdiags.SourceRange
}
