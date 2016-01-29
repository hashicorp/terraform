package dynect

/*
This struct represents the request body that would be sent to the DynECT API
for logging in and getting a session token for future requests.
*/
type LoginBlock struct {
	Username     string `json:"user_name"`
	Password     string `json:"password"`
	CustomerName string `json:"customer_name"`
}

// Type ResponseBlock holds the "header" information returned by any call to
// the DynECT API.
//
// All response-type structs should include this as an anonymous/embedded field.
type ResponseBlock struct {
	Status   string         `json:"string"`
	JobId    int            `json:"job_id,omitempty"`
	Messages []MessageBlock `json:"msgs,omitempty"`
}

// Type MessageBlock holds the message information from the server, and is
// nested within the ResponseBlock type.
type MessageBlock struct {
	Info      string `json:"INFO"`
	Source    string `json:"SOURCE"`
	ErrorCode string `json:"ERR_CD"`
	Level     string `json:"LVL"`
}

// Type LoginResponse holds the data returned by an HTTP POST call to
// https://api.dynect.net/REST/Session/.
type LoginResponse struct {
	ResponseBlock
	Data LoginDataBlock `json:"data"`
}

// Type LoginDataBlock holds the token and API version information from an HTTP
// POST call to https://api.dynect.net/REST/Session/.
//
// It is nested within the LoginResponse struct.
type LoginDataBlock struct {
	Token   string `json:"token"`
	Version string `json:"version"`
}

// RecordRequest holds the request body for a record create/update
type RecordRequest struct {
	RData DataBlock `json:"rdata"`
	TTL   string    `json:"ttl,omitempty"`
}

// PublishZoneBlock holds the request body for a publish zone request
// https://help.dyn.com/update-zone-api/
type PublishZoneBlock struct {
	Publish bool `json:"publish"`
}
