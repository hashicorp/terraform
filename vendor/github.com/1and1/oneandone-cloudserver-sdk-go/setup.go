package oneandone

// The base url for 1&1 Cloud Server REST API.
var BaseUrl = "https://cloudpanel-api.1and1.com/v1"

// Authentication token
var Token string

// SetBaseUrl is intended to set the REST base url. BaseUrl is declared in setup.go
func SetBaseUrl(newbaseurl string) string {
	BaseUrl = newbaseurl
	return BaseUrl
}

// SetToken is used to set authentication Token for the REST service. Token is declared in setup.go
func SetToken(newtoken string) string {
	Token = newtoken
	return Token
}
