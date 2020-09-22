package userdirs

// ForApp returns a set of user-specific directories for a particular
// application.
//
// The three arguments are used in different ways depending on the current
// host operating system, because each OS has different conventions.
//
//    - On Windows, the vendor and string are used to construct a two-level
//      heirarchy, vendor/name, under each namespaced directory prefix.
//      The bundleID is ignored.
//    - On Linux and other similar Unix systems, the name is converted to
//      lowercase and any spaces changed to dashes and used as a subdirectory
//      name. The vendor and bundleID are ignored.
//    - On Mac OS X, the bundleID is used and the name and vendor are ignored.
//
// For best results, the name and vendor arguments should contain
// space-separated words using title case, like "Image Editor" and "Contoso",
// and the bundleID should be a reverse-DNS-style string, like
// "com.example.appname".
func ForApp(name string, vendor string, bundleID string) Dirs {
	// Delegate to OS-specific implementation
	return forApp(name, vendor, bundleID)
}

// SupportedOS returns true if the current operating system is supported by
// this package. If this function returns false, any call to ForApp will
// panic.
func SupportedOS() bool {
	// Delegate to OS-specific implementation
	return supportedOS()
}
