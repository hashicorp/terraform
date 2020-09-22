// Package xdgbase is an implementation of the XDG Basedir Specification
// version 0.8, as published at https://specifications.freedesktop.org/basedir-spec/basedir-spec-0.8.html .
//
// This package has no checks for the host operating system, so it can in
// principle function on any operating system but in practice XDG conventions
// are followed only on some Unix-like systems, so using this library elsewhere
// would not be very useful and will produce undefined results.
package xdgbase
