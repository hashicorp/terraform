package discovery

import (
	"bytes"
)

// PluginRequirements describes a set of plugins (assumed to be of a consistent
// kind) that are required to exist and have versions within the given
// corresponding sets.
type PluginRequirements map[string]*PluginConstraints

// PluginConstraints represents an element of PluginRequirements describing
// the constraints for a single plugin.
type PluginConstraints struct {
	// Specifies that the plugin's version must be within the given
	// constraints.
	Versions Constraints

	// If non-nil, the hash of the on-disk plugin executable must exactly
	// match the SHA256 hash given here.
	SHA256 []byte
}

// Allows returns true if the given version is within the receiver's version
// constraints.
func (s *PluginConstraints) Allows(v Version) bool {
	return s.Versions.Allows(v)
}

// AcceptsSHA256 returns true if the given executable SHA256 hash is acceptable,
// either because it matches the constraint or because there is no such
// constraint.
func (s *PluginConstraints) AcceptsSHA256(digest []byte) bool {
	if s.SHA256 == nil {
		return true
	}
	return bytes.Equal(s.SHA256, digest)
}

// Merge takes the contents of the receiver and the other given requirements
// object and merges them together into a single requirements structure
// that satisfies both sets of requirements.
//
// Note that it doesn't make sense to merge two PluginRequirements with
// differing required plugin SHA256 hashes, since the result will never
// match any plugin.
func (r PluginRequirements) Merge(other PluginRequirements) PluginRequirements {
	ret := make(PluginRequirements)
	for n, c := range r {
		ret[n] = &PluginConstraints{
			Versions: Constraints{}.Append(c.Versions),
			SHA256:   c.SHA256,
		}
	}
	for n, c := range other {
		if existing, exists := ret[n]; exists {
			ret[n].Versions = ret[n].Versions.Append(c.Versions)

			if existing.SHA256 != nil {
				if c.SHA256 != nil && !bytes.Equal(c.SHA256, existing.SHA256) {
					// If we've been asked to merge two constraints with
					// different SHA256 hashes then we'll produce a dummy value
					// that can never match anything. This is a silly edge case
					// that no reasonable caller should hit.
					ret[n].SHA256 = []byte(invalidProviderHash)
				}
			} else {
				ret[n].SHA256 = c.SHA256 // might still be nil
			}
		} else {
			ret[n] = &PluginConstraints{
				Versions: Constraints{}.Append(c.Versions),
				SHA256:   c.SHA256,
			}
		}
	}
	return ret
}

// LockExecutables applies additional constraints to the receiver that
// require plugin executables with specific SHA256 digests. This modifies
// the receiver in-place, since it's intended to be applied after
// version constraints have been resolved.
//
// The given map must include a key for every plugin that is already
// required. If not, any missing keys will cause the corresponding plugin
// to never match, though the direct caller doesn't necessarily need to
// guarantee this as long as the downstream code _applying_ these constraints
// is able to deal with the non-match in some way.
func (r PluginRequirements) LockExecutables(sha256s map[string][]byte) {
	for name, cons := range r {
		digest := sha256s[name]

		if digest == nil {
			// Prevent any match, which will then presumably cause the
			// downstream consumer of this requirements to report an error.
			cons.SHA256 = []byte(invalidProviderHash)
			continue
		}

		cons.SHA256 = digest
	}
}

const invalidProviderHash = "<invalid>"
