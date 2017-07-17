// Package e2etest contains a small number of tests that run against a real
// Terraform binary, compiled on the fly at the start of the test run.
//
// These tests help ensure that key end-to-end Terraform use-cases are working
// for a real binary, whereas other tests always have at least _some_ amount
// of test stubbing.
//
// The goal of this package is not to duplicate the functional testing done
// in other packages but rather to fully exercise a few important workflows
// in a realistic way.
//
// These tests can be used in two ways. The simplest way is to just run them
// with "go test" as normal:
//
//     go test -v github.com/hashicorp/terraform/command/e2etest
//
// This will compile on the fly a Terraform binary and run the tests against
// it.
//
// Alternatively, the make-archive.sh script can be used to produce a
// self-contained zip file that can be shipped to another machine to run
// the tests there without needing a locally-installed Go compiler. This
// is primarily useful for testing cross-compiled builds. For more information,
// see the commentary in make-archive.sh.
//
// The TF_ACC environment variable must be set for the tests to reach out
// to external network services. Since these are end-to-end tests, only a
// few very basic tests can execute without this environment variable set.
package e2etest
