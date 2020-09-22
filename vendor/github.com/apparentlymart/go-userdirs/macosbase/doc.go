// Package macosbase contains helper functions that construct base paths
// conforming to the Mac OS user-specific file layout guidelines as
// documented in https://developer.apple.com/library/archive/documentation/FileManagement/Conceptual/FileSystemProgrammingGuide/FileSystemOverview/FileSystemOverview.html#//apple_ref/doc/uid/TP40010672-CH2-SW6 .
//
// This package only does path construction, and doesn't depend on any Mac OS
// system APIs, so in principle it can run on other platforms but the results
// it produces in that case are undefined and unlikely to be useful.
package macosbase
