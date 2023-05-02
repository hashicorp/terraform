// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

// A PluginMetaSet is a set of PluginMeta objects meeting a certain criteria.
//
// Methods on this type allow filtering of the set to produce subsets that
// meet more restrictive criteria.
type PluginMetaSet map[PluginMeta]struct{}

// Add inserts the given PluginMeta into the receiving set. This is a no-op
// if the given meta is already present.
func (s PluginMetaSet) Add(p PluginMeta) {
	s[p] = struct{}{}
}

// Remove removes the given PluginMeta from the receiving set. This is a no-op
// if the given meta is not already present.
func (s PluginMetaSet) Remove(p PluginMeta) {
	delete(s, p)
}

// Has returns true if the given meta is in the receiving set, or false
// otherwise.
func (s PluginMetaSet) Has(p PluginMeta) bool {
	_, ok := s[p]
	return ok
}

// Count returns the number of metas in the set
func (s PluginMetaSet) Count() int {
	return len(s)
}

// ValidateVersions returns two new PluginMetaSets, separating those with
// versions that have syntax-valid semver versions from those that don't.
//
// Eliminating invalid versions from consideration (and possibly warning about
// them) is usually the first step of working with a meta set after discovery
// has completed.
func (s PluginMetaSet) ValidateVersions() (valid, invalid PluginMetaSet) {
	valid = make(PluginMetaSet)
	invalid = make(PluginMetaSet)
	for p := range s {
		if _, err := p.Version.Parse(); err == nil {
			valid.Add(p)
		} else {
			invalid.Add(p)
		}
	}
	return
}

// WithName returns the subset of metas that have the given name.
func (s PluginMetaSet) WithName(name string) PluginMetaSet {
	ns := make(PluginMetaSet)
	for p := range s {
		if p.Name == name {
			ns.Add(p)
		}
	}
	return ns
}

// WithVersion returns the subset of metas that have the given version.
//
// This should be used only with the "valid" result from ValidateVersions;
// it will ignore any plugin metas that have invalid version strings.
func (s PluginMetaSet) WithVersion(version Version) PluginMetaSet {
	ns := make(PluginMetaSet)
	for p := range s {
		gotVersion, err := p.Version.Parse()
		if err != nil {
			continue
		}
		if gotVersion.Equal(version) {
			ns.Add(p)
		}
	}
	return ns
}

// ByName groups the metas in the set by their Names, returning a map.
func (s PluginMetaSet) ByName() map[string]PluginMetaSet {
	ret := make(map[string]PluginMetaSet)
	for p := range s {
		if _, ok := ret[p.Name]; !ok {
			ret[p.Name] = make(PluginMetaSet)
		}
		ret[p.Name].Add(p)
	}
	return ret
}

// Newest returns the one item from the set that has the newest Version value.
//
// The result is meaningful only if the set is already filtered such that
// all of the metas have the same Name.
//
// If there isn't at least one meta in the set then this function will panic.
// Use Count() to ensure that there is at least one value before calling.
//
// If any of the metas have invalid version strings then this function will
// panic. Use ValidateVersions() first to filter out metas with invalid
// versions.
//
// If two metas have the same Version then one is arbitrarily chosen. This
// situation should be avoided by pre-filtering the set.
func (s PluginMetaSet) Newest() PluginMeta {
	if len(s) == 0 {
		panic("can't call NewestStable on empty PluginMetaSet")
	}

	var first = true
	var winner PluginMeta
	var winnerVersion Version
	for p := range s {
		version, err := p.Version.Parse()
		if err != nil {
			panic(err)
		}

		if first || version.NewerThan(winnerVersion) {
			winner = p
			winnerVersion = version
			first = false
		}
	}

	return winner
}

// ConstrainVersions takes a set of requirements and attempts to
// return a map from name to a set of metas that have the matching
// name and an appropriate version.
//
// If any of the given requirements match *no* plugins then its PluginMetaSet
// in the returned map will be empty.
//
// All viable metas are returned, so the caller can apply any desired filtering
// to reduce down to a single option. For example, calling Newest() to obtain
// the highest available version.
//
// If any of the metas in the set have invalid version strings then this
// function will panic. Use ValidateVersions() first to filter out metas with
// invalid versions.
func (s PluginMetaSet) ConstrainVersions(reqd PluginRequirements) map[string]PluginMetaSet {
	ret := make(map[string]PluginMetaSet)
	for p := range s {
		name := p.Name
		allowedVersions, ok := reqd[name]
		if !ok {
			continue
		}
		if _, ok := ret[p.Name]; !ok {
			ret[p.Name] = make(PluginMetaSet)
		}
		version, err := p.Version.Parse()
		if err != nil {
			panic(err)
		}
		if allowedVersions.Allows(version) {
			ret[p.Name].Add(p)
		}
	}
	return ret
}

// OverridePaths returns a new set where any existing plugins with the given
// names are removed and replaced with the single path given in the map.
//
// This is here only to continue to support the legacy way of overriding
// plugin binaries in the .terraformrc file. It treats all given plugins
// as pre-versioning (version 0.0.0). This mechanism will eventually be
// phased out, with vendor directories being the intended replacement.
func (s PluginMetaSet) OverridePaths(paths map[string]string) PluginMetaSet {
	ret := make(PluginMetaSet)
	for p := range s {
		if _, ok := paths[p.Name]; ok {
			// Skip plugins that we're overridding
			continue
		}

		ret.Add(p)
	}

	// Now add the metadata for overriding plugins
	for name, path := range paths {
		ret.Add(PluginMeta{
			Name:    name,
			Version: VersionZero,
			Path:    path,
		})
	}

	return ret
}
