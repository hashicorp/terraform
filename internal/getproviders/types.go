package getproviders

import (
	"fmt"
	"runtime"
	"sort"
	"strings"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/apparentlymart/go-versions/versions/constraints"

	"github.com/hashicorp/terraform/internal/addrs"
)

// Version represents a particular single version of a provider.
type Version = versions.Version

// UnspecifiedVersion is the zero value of Version, representing the absense
// of a version number.
var UnspecifiedVersion Version = versions.Unspecified

// VersionList represents a list of versions. It is a []Version with some
// extra methods for convenient filtering.
type VersionList = versions.List

// VersionSet represents a set of versions, usually describing the acceptable
// versions that can be selected under a particular version constraint provided
// by the end-user.
type VersionSet = versions.Set

// VersionConstraints represents a set of version constraints, which can
// define the membership of a VersionSet by exclusion.
type VersionConstraints = constraints.IntersectionSpec

// Warnings represents a list of warnings returned by a Registry source.
type Warnings = []string

// Requirements gathers together requirements for many different providers
// into a single data structure, as a convenient way to represent the full
// set of requirements for a particular configuration or state or both.
//
// If an entry in a Requirements has a zero-length VersionConstraints then
// that indicates that the provider is required but that any version is
// acceptable. That's different than a provider being absent from the map
// altogether, which means that it is not required at all.
type Requirements map[addrs.Provider]VersionConstraints

// Merge takes the requirements in the receiever and the requirements in the
// other given value and produces a new set of requirements that combines
// all of the requirements of both.
//
// The resulting requirements will permit only selections that both of the
// source requirements would've allowed.
func (r Requirements) Merge(other Requirements) Requirements {
	ret := make(Requirements)
	for addr, constraints := range r {
		ret[addr] = constraints
	}
	for addr, constraints := range other {
		ret[addr] = append(ret[addr], constraints...)
	}
	return ret
}

// Selections gathers together version selections for many different providers.
//
// This is the result of provider installation: a specific version selected
// for each provider given in the requested Requirements, selected based on
// the given version constraints.
type Selections map[addrs.Provider]Version

// ParseVersion parses a "semver"-style version string into a Version value,
// which is the version syntax we use for provider versions.
func ParseVersion(str string) (Version, error) {
	return versions.ParseVersion(str)
}

// MustParseVersion is a variant of ParseVersion that panics if it encounters
// an error while parsing.
func MustParseVersion(str string) Version {
	ret, err := ParseVersion(str)
	if err != nil {
		panic(err)
	}
	return ret
}

// ParseVersionConstraints parses a "Ruby-like" version constraint string
// into a VersionConstraints value.
func ParseVersionConstraints(str string) (VersionConstraints, error) {
	return constraints.ParseRubyStyleMulti(str)
}

// MustParseVersionConstraints is a variant of ParseVersionConstraints that
// panics if it encounters an error while parsing.
func MustParseVersionConstraints(str string) VersionConstraints {
	ret, err := ParseVersionConstraints(str)
	if err != nil {
		panic(err)
	}
	return ret
}

// MeetingConstraints returns a version set that contains all of the versions
// that meet the given constraints, specified using the Spec type from the
// constraints package.
func MeetingConstraints(vc VersionConstraints) VersionSet {
	return versions.MeetingConstraints(vc)
}

// Platform represents a target platform that a provider is or might be
// available for.
type Platform struct {
	OS, Arch string
}

func (p Platform) String() string {
	return p.OS + "_" + p.Arch
}

// LessThan returns true if the receiver should sort before the other given
// Platform in an ordered list of platforms.
//
// The ordering is lexical first by OS and then by Architecture.
// This ordering is primarily just to ensure that results of
// functions in this package will be deterministic. The ordering is not
// intended to have any semantic meaning and is subject to change in future.
func (p Platform) LessThan(other Platform) bool {
	switch {
	case p.OS != other.OS:
		return p.OS < other.OS
	default:
		return p.Arch < other.Arch
	}
}

// ParsePlatform parses a string representation of a platform, like
// "linux_amd64", or returns an error if the string is not valid.
func ParsePlatform(str string) (Platform, error) {
	parts := strings.Split(str, "_")
	if len(parts) != 2 {
		return Platform{}, fmt.Errorf("must be two words separated by an underscore")
	}

	os, arch := parts[0], parts[1]
	if strings.ContainsAny(os, " \t\n\r") {
		return Platform{}, fmt.Errorf("OS portion must not contain whitespace")
	}
	if strings.ContainsAny(arch, " \t\n\r") {
		return Platform{}, fmt.Errorf("architecture portion must not contain whitespace")
	}

	return Platform{
		OS:   os,
		Arch: arch,
	}, nil
}

// CurrentPlatform is the platform where the current program is running.
//
// If attempting to install providers for use on the same system where the
// installation process is running, this is the right platform to use.
var CurrentPlatform = Platform{
	OS:   runtime.GOOS,
	Arch: runtime.GOARCH,
}

// PackageMeta represents the metadata related to a particular downloadable
// provider package targeting a single platform.
//
// Package findproviders does no signature verification or protocol version
// compatibility checking of its own. A caller receving a PackageMeta must
// verify that it has a correct signature and supports a protocol version
// accepted by the current version of Terraform before trying to use the
// described package.
type PackageMeta struct {
	Provider addrs.Provider
	Version  Version

	ProtocolVersions VersionList
	TargetPlatform   Platform

	Filename string
	Location PackageLocation

	// Authentication, if non-nil, is a request from the source that produced
	// this meta for verification of the target package after it has been
	// retrieved from the indicated Location.
	//
	// Different sources will support different authentication strategies --
	// or possibly no strategies at all -- depending on what metadata they
	// have available to them, such as checksums provided out-of-band by the
	// original package author, expected signing keys, etc.
	//
	// If Authentication is non-nil then no authentication is requested.
	// This is likely appropriate only for packages that are already available
	// on the local system.
	Authentication PackageAuthentication
}

// LessThan returns true if the receiver should sort before the given other
// PackageMeta in a sorted list of PackageMeta.
//
// Sorting preference is given first to the provider address, then to the
// taget platform, and the to the version number (using semver precedence).
// Packages that differ only in semver build metadata have no defined
// precedence and so will always return false.
//
// This ordering is primarily just to maximize the chance that results of
// functions in this package will be deterministic. The ordering is not
// intended to have any semantic meaning and is subject to change in future.
func (m PackageMeta) LessThan(other PackageMeta) bool {
	switch {
	case m.Provider != other.Provider:
		return m.Provider.LessThan(other.Provider)
	case m.TargetPlatform != other.TargetPlatform:
		return m.TargetPlatform.LessThan(other.TargetPlatform)
	case m.Version != other.Version:
		return m.Version.LessThan(other.Version)
	default:
		return false
	}
}

// UnpackedDirectoryPath determines the path under the given base
// directory where SearchLocalDirectory or the FilesystemMirrorSource would
// expect to find an unpacked copy of the receiving PackageMeta.
//
// The result always uses forward slashes as path separator, even on Windows,
// to produce a consistent result on all platforms. Windows accepts both
// direction of slash as long as each individual path string is self-consistent.
func (m PackageMeta) UnpackedDirectoryPath(baseDir string) string {
	return UnpackedDirectoryPathForPackage(baseDir, m.Provider, m.Version, m.TargetPlatform)
}

// PackedFilePath determines the path under the given base
// directory where SearchLocalDirectory or the FilesystemMirrorSource would
// expect to find packed copy (a .zip archive) of the receiving PackageMeta.
//
// The result always uses forward slashes as path separator, even on Windows,
// to produce a consistent result on all platforms. Windows accepts both
// direction of slash as long as each individual path string is self-consistent.
func (m PackageMeta) PackedFilePath(baseDir string) string {
	return PackedFilePathForPackage(baseDir, m.Provider, m.Version, m.TargetPlatform)
}

// AcceptableHashes returns a set of hashes that could be recorded for
// comparison to future results for the same provider version, to implement a
// "trust on first use" scheme.
//
// The AcceptableHashes result is a platform-agnostic set of hashes, with the
// intent that in most cases it will be used as an additional cross-check in
// addition to a platform-specific hash check made during installation. However,
// there are some situations (such as verifying an already-installed package
// that's on local disk) where Terraform would check only against the results
// of this function, meaning that it would in principle accept another
// platform's package as a substitute for the correct platform. That's a
// pragmatic compromise to allow lock files derived from the result of this
// method to be portable across platforms.
//
// Callers of this method should typically also verify the package using the
// object in the Authentication field, and consider how much trust to give
// the result of this method depending on the authentication result: an
// unauthenticated result or one that only verified a checksum could be
// considered less trustworthy than one that checked the package against
// a signature provided by the origin registry.
//
// The AcceptableHashes result is actually provided by the object in the
// Authentication field. AcceptableHashes therefore returns an empty result
// for a PackageMeta that has no authentication object, or has one that does
// not make use of hashes.
func (m PackageMeta) AcceptableHashes() []Hash {
	auth, ok := m.Authentication.(PackageAuthenticationHashes)
	if !ok {
		return nil
	}
	return auth.AcceptableHashes()
}

// PackageLocation represents a location where a provider distribution package
// can be obtained. A value of this type contains one of the following
// concrete types: PackageLocalArchive, PackageLocalDir, or PackageHTTPURL.
type PackageLocation interface {
	packageLocation()
	String() string
}

// PackageLocalArchive is the location of a provider distribution archive file
// in the local filesystem. Its value is a local filesystem path using the
// syntax understood by Go's standard path/filepath package on the operating
// system where Terraform is running.
type PackageLocalArchive string

func (p PackageLocalArchive) packageLocation() {}
func (p PackageLocalArchive) String() string   { return string(p) }

// PackageLocalDir is the location of a directory containing an unpacked
// provider distribution archive in the local filesystem. Its value is a local
// filesystem path using the syntax understood by Go's standard path/filepath
// package on the operating system where Terraform is running.
type PackageLocalDir string

func (p PackageLocalDir) packageLocation() {}
func (p PackageLocalDir) String() string   { return string(p) }

// PackageHTTPURL is a provider package location accessible via HTTP.
// Its value is a URL string using either the http: scheme or the https: scheme.
type PackageHTTPURL string

func (p PackageHTTPURL) packageLocation() {}
func (p PackageHTTPURL) String() string   { return string(p) }

// PackageMetaList is a list of PackageMeta. It's just []PackageMeta with
// some methods for convenient sorting and filtering.
type PackageMetaList []PackageMeta

func (l PackageMetaList) Len() int {
	return len(l)
}

func (l PackageMetaList) Less(i, j int) bool {
	return l[i].LessThan(l[j])
}

func (l PackageMetaList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

// Sort performs an in-place, stable sort on the contents of the list, using
// the ordering given by method Less. This ordering is primarily to help
// encourage deterministic results from functions and does not have any
// semantic meaning.
func (l PackageMetaList) Sort() {
	sort.Stable(l)
}

// FilterPlatform constructs a new PackageMetaList that contains only the
// elements of the receiver that are for the given target platform.
//
// Pass CurrentPlatform to filter only for packages targeting the platform
// where this code is running.
func (l PackageMetaList) FilterPlatform(target Platform) PackageMetaList {
	var ret PackageMetaList
	for _, m := range l {
		if m.TargetPlatform == target {
			ret = append(ret, m)
		}
	}
	return ret
}

// FilterProviderExactVersion constructs a new PackageMetaList that contains
// only the elements of the receiver that relate to the given provider address
// and exact version.
//
// The version matching for this function is exact, including matching on
// semver build metadata, because it's intended for handling a single exact
// version selected by the caller from a set of available versions.
func (l PackageMetaList) FilterProviderExactVersion(provider addrs.Provider, version Version) PackageMetaList {
	var ret PackageMetaList
	for _, m := range l {
		if m.Provider == provider && m.Version == version {
			ret = append(ret, m)
		}
	}
	return ret
}

// FilterProviderPlatformExactVersion is a combination of both
// FilterPlatform and FilterProviderExactVersion that filters by all three
// criteria at once.
func (l PackageMetaList) FilterProviderPlatformExactVersion(provider addrs.Provider, platform Platform, version Version) PackageMetaList {
	var ret PackageMetaList
	for _, m := range l {
		if m.Provider == provider && m.Version == version && m.TargetPlatform == platform {
			ret = append(ret, m)
		}
	}
	return ret
}

// VersionConstraintsString returns a canonical string representation of
// a VersionConstraints value.
func VersionConstraintsString(spec VersionConstraints) string {
	// (we have our own function for this because the upstream versions
	// library prefers to use npm/cargo-style constraint syntax, but
	// Terraform prefers Ruby-like. Maybe we can upstream a "RubyLikeString")
	// function to do this later, but having this in here avoids blocking on
	// that and this is the sort of thing that is unlikely to need ongoing
	// maintenance because the version constraint syntax is unlikely to change.)
	//
	// ParseVersionConstraints allows some variations for convenience, but the
	// return value from this function serves as the normalized form of a
	// particular version constraint, which is the form we require in dependency
	// lock files. Therefore the canonical forms produced here are a compatibility
	// constraint for the dependency lock file parser.

	if len(spec) == 0 {
		return ""
	}

	// VersionConstraints values are typically assembled by combining together
	// the version constraints from many separate declarations throughout
	// a configuration, across many modules. As a consequence, they typically
	// contain duplicates and the terms inside are in no particular order.
	// For our canonical representation we'll both deduplicate the items
	// and sort them into a consistent order.
	sels := make(map[constraints.SelectionSpec]struct{})
	for _, sel := range spec {
		// The parser allows writing abbreviated version (such as 2) which
		// end up being represented in memory with trailing unconstrained parts
		// (for example 2.*.*). For the purpose of serialization with Ruby
		// style syntax, these unconstrained parts can all be represented as 0
		// with no loss of meaning, so we make that conversion here. Doing so
		// allows us to deduplicate equivalent constraints, such as >= 2.0 and
		// >= 2.0.0.
		normalizedSel := constraints.SelectionSpec{
			Operator: sel.Operator,
			Boundary: sel.Boundary.ConstrainToZero(),
		}
		sels[normalizedSel] = struct{}{}
	}
	selsOrder := make([]constraints.SelectionSpec, 0, len(sels))
	for sel := range sels {
		selsOrder = append(selsOrder, sel)
	}
	sort.Slice(selsOrder, func(i, j int) bool {
		is, js := selsOrder[i], selsOrder[j]
		boundaryCmp := versionSelectionBoundaryCompare(is.Boundary, js.Boundary)
		if boundaryCmp == 0 {
			// The operator is the decider, then.
			return versionSelectionOperatorLess(is.Operator, js.Operator)
		}
		return boundaryCmp < 0
	})

	var b strings.Builder
	for i, sel := range selsOrder {
		if i > 0 {
			b.WriteString(", ")
		}
		switch sel.Operator {
		case constraints.OpGreaterThan:
			b.WriteString("> ")
		case constraints.OpLessThan:
			b.WriteString("< ")
		case constraints.OpGreaterThanOrEqual:
			b.WriteString(">= ")
		case constraints.OpGreaterThanOrEqualPatchOnly, constraints.OpGreaterThanOrEqualMinorOnly:
			// These two differ in how the version is written, not in the symbol.
			b.WriteString("~> ")
		case constraints.OpLessThanOrEqual:
			b.WriteString("<= ")
		case constraints.OpEqual:
			b.WriteString("")
		case constraints.OpNotEqual:
			b.WriteString("!= ")
		default:
			// The above covers all of the operators we support during
			// parsing, so we should not get here.
			b.WriteString("??? ")
		}

		// We use a different constraint operator to distinguish between the
		// two types of pessimistic constraint: minor-only and patch-only. For
		// minor-only constraints, we always want to display only the major and
		// minor version components, so we special-case that operator below.
		//
		// One final edge case is a minor-only constraint specified with only
		// the major version, such as ~> 2. We treat this the same as ~> 2.0,
		// because a major-only pessimistic constraint does not exist: it is
		// logically identical to >= 2.0.0.
		if sel.Operator == constraints.OpGreaterThanOrEqualMinorOnly {
			// The minor-pessimistic syntax uses only two version components.
			fmt.Fprintf(&b, "%s.%s", sel.Boundary.Major, sel.Boundary.Minor)
		} else {
			fmt.Fprintf(&b, "%s.%s.%s", sel.Boundary.Major, sel.Boundary.Minor, sel.Boundary.Patch)
		}
		if sel.Boundary.Prerelease != "" {
			b.WriteString("-" + sel.Boundary.Prerelease)
		}
		if sel.Boundary.Metadata != "" {
			b.WriteString("+" + sel.Boundary.Metadata)
		}
	}
	return b.String()
}

// Our sort for selection operators is somewhat arbitrary and mainly motivated
// by consistency rather than meaning, but this ordering does at least try
// to make it so "simple" constraint sets will appear how a human might
// typically write them, with the lower bounds first and the upper bounds
// last. Weird mixtures of different sorts of constraints will likely seem
// less intuitive, but they'd be unintuitive no matter the ordering.
var versionSelectionsBoundaryPriority = map[constraints.SelectionOp]int{
	// We skip zero here so that if we end up seeing an invalid
	// operator (which the string function would render as "???")
	// then it will have index zero and thus appear first.
	constraints.OpGreaterThan:                 1,
	constraints.OpGreaterThanOrEqual:          2,
	constraints.OpEqual:                       3,
	constraints.OpGreaterThanOrEqualPatchOnly: 4,
	constraints.OpGreaterThanOrEqualMinorOnly: 5,
	constraints.OpLessThanOrEqual:             6,
	constraints.OpLessThan:                    7,
	constraints.OpNotEqual:                    8,
}

func versionSelectionOperatorLess(i, j constraints.SelectionOp) bool {
	iPrio := versionSelectionsBoundaryPriority[i]
	jPrio := versionSelectionsBoundaryPriority[j]
	return iPrio < jPrio
}

func versionSelectionBoundaryCompare(i, j constraints.VersionSpec) int {
	// In the Ruby-style constraint syntax, unconstrained parts appear
	// only for omitted portions of a version string, like writing
	// "2" instead of "2.0.0". For sorting purposes we'll just
	// consider those as zero, which also matches how we serialize them
	// to strings.
	i, j = i.ConstrainToZero(), j.ConstrainToZero()

	// Once we've removed any unconstrained parts, we can safely
	// convert to our main Version type so we can use its ordering.
	iv := Version{
		Major:      i.Major.Num,
		Minor:      i.Minor.Num,
		Patch:      i.Patch.Num,
		Prerelease: versions.VersionExtra(i.Prerelease),
		Metadata:   versions.VersionExtra(i.Metadata),
	}
	jv := Version{
		Major:      j.Major.Num,
		Minor:      j.Minor.Num,
		Patch:      j.Patch.Num,
		Prerelease: versions.VersionExtra(j.Prerelease),
		Metadata:   versions.VersionExtra(j.Metadata),
	}
	if iv.Same(jv) {
		// Although build metadata doesn't normally weigh in to
		// precedence choices, we'll use it for our visual
		// ordering just because we need to pick _some_ order.
		switch {
		case iv.Metadata.Raw() == jv.Metadata.Raw():
			return 0
		case iv.Metadata.LessThan(jv.Metadata):
			return -1
		default:
			return 1 // greater, by elimination
		}
	}
	switch {
	case iv.LessThan(jv):
		return -1
	default:
		return 1 // greater, by elimination
	}
}
