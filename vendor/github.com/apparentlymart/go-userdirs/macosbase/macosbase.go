package macosbase

import (
	"path/filepath"
)

// ApplicationSupportDir returns the path to the current user's
// "Application Support" library directory.
func ApplicationSupportDir() string {
	return filepath.Join(home(), "Library", "Application Support")
}

// CachesDir returns the path to the current user's "Caches" library directory.
func CachesDir() string {
	return filepath.Join(home(), "Library", "Caches")
}

// FrameworksDir returns the path to the current user's "Frameworks" library directory.
func FrameworksDir() string {
	return filepath.Join(home(), "Library", "Frameworks")
}

// PreferencesDir returns the path to the current user's "Preferences" library directory.
func PreferencesDir() string {
	return filepath.Join(home(), "Library", "Preferences")
}
