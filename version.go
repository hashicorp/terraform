package main

import (
	"github.com/hashicorp/terraform/version"
)

// The git commit that was compiled. This will be filled in by the compiler.
var GitCommit string

var Version = version.Version

var VersionPrerelease = version.Prerelease
