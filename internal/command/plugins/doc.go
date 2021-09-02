// Package plugins is the home of various rules and logic for deciding which
// plugins (providers and provisioners) are to be used for actions in a
// particular working directory.
//
// The main type for this package is plugins.Manager, which encapsulates the
// various concerns around locating, verifying, and executing already-installed
// plugins.
//
// Plugin installation is _not_ the responsibility of this package. The logic
// here assumes any necessary external plugins will be already installed by
// some other subsystem.
package plugins
