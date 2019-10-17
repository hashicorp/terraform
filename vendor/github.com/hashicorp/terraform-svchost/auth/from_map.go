package auth

import (
	"github.com/zclconf/go-cty/cty"
)

// HostCredentialsFromMap converts a map of key-value pairs from a credentials
// definition provided by the user (e.g. in a config file, or via a credentials
// helper) into a HostCredentials object if possible, or returns nil if
// no credentials could be extracted from the map.
//
// This function ignores map keys it is unfamiliar with, to allow for future
// expansion of the credentials map format for new credential types.
func HostCredentialsFromMap(m map[string]interface{}) HostCredentials {
	if m == nil {
		return nil
	}
	if token, ok := m["token"].(string); ok {
		return HostCredentialsToken(token)
	}
	return nil
}

// HostCredentialsFromObject converts a cty.Value of an object type into a
// HostCredentials object if possible, or returns nil if no credentials could
// be extracted from the map.
//
// This function ignores object attributes it is unfamiliar with, to allow for
// future expansion of the credentials object structure for new credential types.
//
// If the given value is not of an object type, this function will panic.
func HostCredentialsFromObject(obj cty.Value) HostCredentials {
	if !obj.Type().HasAttribute("token") {
		return nil
	}

	tokenV := obj.GetAttr("token")
	if tokenV.IsNull() || !tokenV.IsKnown() {
		return nil
	}
	if !cty.String.Equals(tokenV.Type()) {
		// Weird, but maybe some future Terraform version accepts an object
		// here for some reason, so we'll be resilient.
		return nil
	}

	return HostCredentialsToken(tokenV.AsString())
}
