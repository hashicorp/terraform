package e2etest

import (
	"debug/buildinfo"
	"testing"
)

// The following Go Modules are forbidden as dependencies of Terraform CLI,
// and TestForbiddenDependencies will fail if it finds any of them in the
// Terraform CLI executable we're testing.
var forbiddenDependencies = map[string]struct{}{
	// At the time of writing this module is unmaintained and has an unpatched
	// security vulnerability where it can generate considerbly-non-random
	// UUIDs. Some of our dependencies use it in ways that don't contribute
	// to the Terraform CLI executable, but we want to catch and reject
	// any changes that would make it be included in the CLI executable.
	"github.com/satori/go.uuid": {},
}

func TestForbiddenDependencies(t *testing.T) {
	// terraformBin is set up by TestMain to refer to the Terraform CLI
	// executable we're running our tests against.
	executable := terraformBin
	info, err := buildinfo.ReadFile(executable)
	if err != nil {
		t.Fatalf("can't read build information from %s: %s", executable, err)
	}

	for _, dep := range info.Deps {
		if _, forbidden := forbiddenDependencies[dep.Path]; forbidden {
			t.Errorf("executable %s includes forbidden dependency %q", executable, dep.Path)
		}
	}
}
