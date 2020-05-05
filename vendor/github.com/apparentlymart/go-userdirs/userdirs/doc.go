// Package userdirs is a utility for building user-specific filesystem paths
// for applications to store configuration information, caches, etc.
//
// It aims to conform to the conventions of the operating system where it is
// running, allowing applications using this package to follow the relevant
// conventions automatically.
//
// Because the behavior of this library must be tailored for each operating
// system it supports, it supports only a subset of Go's own supported
// operating system targets. Others are intentionally not supported (rather than
// mapped on to some default OS) to avoid the situation where adding a new
// supported OS would change the behavior of existing applications built for
// that OS.
//
// Currently this package supports Mac OS X ("darwin"), Linux and Windows
// first-class. It also maps AIX, Dragonfly, FreeBSD, NetBSD, and Solaris,
// following the same rules as for Linux.
//
// On Mac OS X, we follow the Standard Directories guidelines from
// https://developer.apple.com/library/archive/documentation/FileManagement/Conceptual/FileSystemProgrammingGuide/FileSystemOverview/FileSystemOverview.html#//apple_ref/doc/uid/TP40010672-CH2-SW6 .
//
// On Linux and other Unix-like systems, we follow the XDG Base Directory
// specification version 0.8, from https://specifications.freedesktop.org/basedir-spec/basedir-spec-0.8.html .
//
// On Windows, we use the Known Folder API and follow relevant platform conventions
// https://docs.microsoft.com/en-us/windows/desktop/shell/knownfolderid .
//
// On all other systems, the directory-construction functions will panic. Use
// the SupportedOS function to determine whether this function's packages are
// available on the current operating system target, to avoid those panics.
// However, in practice it probably doesn't make much sense to use this package
// when building for an unsupported operating system anyway.
//
// Additional operating systems may be supported in future releases. Once an
// operating system is supported, the constructed paths are frozen to ensure
// that applications can find their same files on future versions. Therefore
// the bar for adding support for a new operating system is there being a
// committed standard published by the operating system vendor.
package userdirs
