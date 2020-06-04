package awsbase

import (
	"errors"
	"fmt"
)

// CannotAssumeRoleError occurs when AssumeRole cannot complete.
type CannotAssumeRoleError struct {
	Config *Config
	Err    error
}

func (e CannotAssumeRoleError) Error() string {
	if e.Config == nil {
		return fmt.Sprintf("cannot assume role: %s", e.Err)
	}

	return fmt.Sprintf(`IAM Role (%s) cannot be assumed.

There are a number of possible causes of this - the most common are:
  * The credentials used in order to assume the role are invalid
  * The credentials do not have appropriate permission to assume the role
  * The role ARN is not valid

Error: %s
`, e.Config.AssumeRoleARN, e.Err)
}

func (e CannotAssumeRoleError) Unwrap() error {
	return e.Err
}

// IsCannotAssumeRoleError returns true if the error contains the CannotAssumeRoleError type.
func IsCannotAssumeRoleError(err error) bool {
	var e CannotAssumeRoleError
	return errors.As(err, &e)
}

func (c *Config) NewCannotAssumeRoleError(err error) CannotAssumeRoleError {
	return CannotAssumeRoleError{Config: c, Err: err}
}

// NoValidCredentialSourcesError occurs when all credential lookup methods have been exhausted without results.
type NoValidCredentialSourcesError struct {
	Config *Config
	Err    error
}

func (e NoValidCredentialSourcesError) Error() string {
	if e.Config == nil {
		return fmt.Sprintf("no valid credential sources found: %s", e.Err)
	}

	return fmt.Sprintf(`no valid credential sources for %s found.

Please see %s
for more information about providing credentials.

Error: %s
`, e.Config.CallerName, e.Config.CallerDocumentationURL, e.Err)
}

func (e NoValidCredentialSourcesError) Unwrap() error {
	return e.Err
}

// IsNoValidCredentialSourcesError returns true if the error contains the NoValidCredentialSourcesError type.
func IsNoValidCredentialSourcesError(err error) bool {
	var e NoValidCredentialSourcesError
	return errors.As(err, &e)
}

func (c *Config) NewNoValidCredentialSourcesError(err error) NoValidCredentialSourcesError {
	return NoValidCredentialSourcesError{Config: c, Err: err}
}
