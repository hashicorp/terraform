package module

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/registry/response"
)

const anyVersion = ">=0.0.0"

var explicitEqualityConstraint = regexp.MustCompile("^=[0-9]")

// return the newest version that satisfies the provided constraint
func newest(versions []string, constraint string) (string, error) {
	if constraint == "" {
		constraint = anyVersion
	}
	cs, err := version.NewConstraint(constraint)
	if err != nil {
		return "", err
	}

	// Find any build metadata in the constraints, and
	// store whether the constraint is an explicit equality that
	// contains a build metadata requirement, so we can return a specific,
	// if requested, build metadata version
	var constraintMetas []string
	var equalsConstraint bool
	for i := range cs {
		constraintMeta := strings.SplitAfterN(cs[i].String(), "+", 2)
		if len(constraintMeta) > 1 {
			constraintMetas = append(constraintMetas, constraintMeta[1])
		}
	}

	if len(cs) == 1 {
		equalsConstraint = explicitEqualityConstraint.MatchString(cs.String())
	}

	// If the version string includes metadata, this is valid in go-version,
	// However, it's confusing as to what expected behavior should be,
	// so give an error so the user can do something more logical
	if (len(cs) > 1 || !equalsConstraint) && len(constraintMetas) > 0 {
		return "", fmt.Errorf("Constraints including build metadata must have explicit equality, or are otherwise too ambiguous: %s", cs.String())
	}

	switch len(versions) {
	case 0:
		return "", errors.New("no versions found")
	case 1:
		v, err := version.NewVersion(versions[0])
		if err != nil {
			return "", err
		}

		if !cs.Check(v) {
			return "", fmt.Errorf("no version found matching constraint %q", constraint)
		}
		return versions[0], nil
	}

	sort.Slice(versions, func(i, j int) bool {
		// versions should have already been validated
		// sort invalid version strings to the end
		iv, err := version.NewVersion(versions[i])
		if err != nil {
			return true
		}
		jv, err := version.NewVersion(versions[j])
		if err != nil {
			return true
		}
		return iv.GreaterThan(jv)
	})

	// versions are now in order, so just find the first which satisfies the
	// constraint
	for i := range versions {
		v, err := version.NewVersion(versions[i])
		if err != nil {
			continue
		}
		if cs.Check(v) {
			// Constraint has metadata and is explicit equality
			if equalsConstraint && len(constraintMetas) > 0 {
				if constraintMetas[0] != v.Metadata() {
					continue
				}
			}
			return versions[i], nil
		}
	}

	return "", nil
}

// return the newest *moduleVersion that matches the given constraint
// TODO: reconcile these two types and newest* functions
func newestVersion(moduleVersions []*response.ModuleVersion, constraint string) (*response.ModuleVersion, error) {
	var versions []string
	modules := make(map[string]*response.ModuleVersion)

	for _, m := range moduleVersions {
		versions = append(versions, m.Version)
		modules[m.Version] = m
	}

	match, err := newest(versions, constraint)
	return modules[match], err
}

// return the newest moduleRecord that matches the given constraint
func newestRecord(moduleVersions []moduleRecord, constraint string) (moduleRecord, error) {
	var versions []string
	modules := make(map[string]moduleRecord)

	for _, m := range moduleVersions {
		versions = append(versions, m.Version)
		modules[m.Version] = m
	}

	match, err := newest(versions, constraint)
	return modules[match], err
}
