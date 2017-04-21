package profitbricks

// Endpoint is the base url for REST requests.
var Endpoint = "https://api.profitbricks.com/cloudapi/v3"

//  Username for authentication .
var Username string

// Password for authentication .
var Passwd string

// SetEndpoint is used to set the REST Endpoint. Endpoint is declared in config.go
func SetEndpoint(newendpoint string) string {
	Endpoint = newendpoint
	return Endpoint
}

// SetAuth is used to set Username and Passwd. Username and Passwd are declared in config.go

func SetAuth(u, p string) {
	Username = u
	Passwd = p
}

func SetUserAgent(userAgent string) {
	AgentHeader = userAgent
}
