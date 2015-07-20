package main

import "github.com/hashicorp/terraform/terraform"

// The git commit that was compiled. This will be filled in by the compiler.
var GitCommit string

const Version = terraform.Version
const VersionPrerelease = terraform.VersionPrerelease
