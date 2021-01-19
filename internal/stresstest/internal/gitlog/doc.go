// Package gitlog has some helpers for translating stressrun.Log values and
// their contents into git objects/repositories, as a convenient way to
// export the inputs and results of a configuration series, to allow for more
// detailed inspection of a failure case.
//
// The goal is to produce a minimally-functional git repository that contains
// a series of commits representing each of the log steps, where each one
// should be structured in a way that a user could then run
// "stresstest terraform apply" to reproduce the same behavior the test harness
// would, where "stresstest terraform" is just a thin wrapper around the normal
// "terraform" CLI executable which makes the "stressful" provider available
// to use.
package gitlog
