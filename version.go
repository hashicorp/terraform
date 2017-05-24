package main

import "github.com/r3labs/terraform/terraform"

// The git commit that was compiled. This will be filled in by the compiler.
var GitCommit string

const Version = terraform.Version

var VersionPrerelease = terraform.VersionPrerelease
