// This package contains a separate entry point to Terraform (i.e. it is a
// separate package main) which aims to find unanticipated defects in the
// current version of Terraform by randomly generating a variety of different
// configurations and running the usual Terraform plan/apply cycle against
// them, before verifying that the final state meets expectations.
//
// This is intended as a complement for unit tests, integration tests, and
// acceptance tests elsewhere in this codebase. If the stresstest program
// detects any failures then the first step should usually be to copy the
// failed example into a fixed test somewhere else in the codebase and attempt
// to reduce it to a minimum case that fails in the same way. Once you've
// found the cause of the failure, retain that fixed test as additional unit
// or integration test coverage.
//
// The design of this package is a compromise with the aim of avoiding the need
// for this package to change with every change to Terraform itself, but yet
// to still exercise enough functionality to be worth running. A key part of
// this design is that it's constrained to generating distinct configuration
// constructs (resources, etc) and then verifying that each one had the
// expected impact on the state. It doesn't aim to verify any of the mechanics
// of how that happened, and the test assertions can't take into account
// anything that isn't visible in the state of each specific generated object.
//
// It also uses fake providers and thus may not detect incorrect behaviors
// that originate in misbehavior in a provider or in a plugin SDK. Although it
// would be nice to also test against real providers, that would significantly
// increase the maintenence burden of this testing technique due to the
// providers themselves being under constant maintenence and improvement.
// Instead, we try to write fake provider implementations that behave as much
// like "real" providers as is practical, within the constraint that we're
// only exercising "happy paths" here -- anything where an error is the
// expected result cannot be verified by stresstest and so must be verified
// elsewhere, such as in unit or integration tests.
//
// If you are adding a new feature to Terraform which adds new configuration
// surface area, consider how best to incorporate that feature into the
// random configuration generator. Even if your feature only affects the
// internal behavior and not the end result visible in the state, it can still
// be valuable for stresstest to try to enable it and verify see that the
// resulting configuration can still be planned and applied without errors.
//
// This is a separate program rather than just part of our normal test suite
// because it can generate configurations that are potentially quite expensive
// to run and there'll likely be quite a bit of variation in what it tests
// between runs. Instead, the intent is to use it to "soak" a forthcoming
// release towards the end of its release cycle, in an attempt to identify any
// unexpected regressions either prior to or during the prerelease testing
// period. Consequently, the program is designed to just keep running until
// interrupted with SIGINT, similarly to a fuzz tester. After you interrupt it,
// stresstest will conclude any tests it is currently running and then exit.
//
// By default it runs only one test at a time, but you can opt in to running
// multiple tests concurrently (at the expense of occupying more CPU time)
// using the -concurrent command line option.
package main
