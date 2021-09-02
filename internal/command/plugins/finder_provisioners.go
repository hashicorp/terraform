package plugins

// WithProvisionerSearchDirs returns a Finder which has the same settings
// as the receiver except that its sequence of provisioner search directories
// is entirely replaced with the given sequence.
//
// Unlike most other "With..." methods on this type, not that this one
// overrides something that would typically be set in the initial call to
// NewFinder, and so callers should use this method only when applying a
// local override to the default settings. (In practice, that's the result
// of passing -plugin-dir=... options to the "terraform init" command,
// which overrides the plugin search directories for a particular working
// directory.)
func (f Finder) WithProvisionerSearchDirs(dirs []string) Finder {
	f.provisionerSearchDirs = dirs
	return f
}
