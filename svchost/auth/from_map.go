package auth

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
