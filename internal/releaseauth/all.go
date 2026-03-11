// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package releaseauth

// Authenticator is a generic interface for interacting with types that authenticate
// an archive.
type Authenticator interface {
	Authenticate() error
}

// All is a meta Authenticator that wraps other Authenticators and ensures they all
// return without failure.
type All struct {
	Authenticator
	authenticators []Authenticator
}

var _ Authenticator = All{}

// AllAuthenticators creates a meta Authenticator that ensures all the
// given Authenticators return without failure.
func AllAuthenticators(authenticators ...Authenticator) All {
	return All{
		authenticators: authenticators,
	}
}

// Authenticate returns the first archive authentication failure from
// the list of Authenticators given.
func (a All) Authenticate() error {
	for _, auth := range a.authenticators {
		if err := auth.Authenticate(); err != nil {
			return err
		}
	}
	return nil
}
