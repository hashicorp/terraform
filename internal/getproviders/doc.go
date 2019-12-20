// Package getproviders is the lowest-level provider automatic installation
// functionality. It can answer questions about what providers and provider
// versions are available in a registry, and it can retrieve the URL for
// the distribution archive for a specific version of a specific provider
// targeting a particular platform.
//
// This package is not responsible for choosing the best version to install
// from a set of available versions, or for any signature verification of the
// archives it fetches. Callers will use this package in conjunction with other
// logic elsewhere in order to construct a full provider installer.
package getproviders
