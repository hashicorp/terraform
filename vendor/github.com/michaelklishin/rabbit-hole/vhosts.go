package rabbithole

import (
	"encoding/json"
	"net/http"
	"net/url"
)

//
// GET /api/vhosts
//

// Example response:

// [
//   {
//     "message_stats": {
//       "publish": 78,
//       "publish_details": {
//         "rate": 0
//       }
//     },
//     "messages": 0,
//     "messages_details": {
//       "rate": 0
//     },
//     "messages_ready": 0,
//     "messages_ready_details": {
//       "rate": 0
//     },
//     "messages_unacknowledged": 0,
//     "messages_unacknowledged_details": {
//       "rate": 0
//     },
//     "recv_oct": 16653,
//     "recv_oct_details": {
//       "rate": 0
//     },
//     "send_oct": 40495,
//     "send_oct_details": {
//       "rate": 0
//     },
//     "name": "\/",
//     "tracing": false
//   },
//   {
//     "name": "29dd51888b834698a8b5bc3e7f8623aa1c9671f5",
//     "tracing": false
//   }
// ]

type VhostInfo struct {
	// Virtual host name
	Name string `json:"name"`
	// True if tracing is enabled for this virtual host
	Tracing bool `json:"tracing"`

	// Total number of messages in queues of this virtual host
	Messages        int         `json:"messages"`
	MessagesDetails RateDetails `json:"messages_details"`

	// Total number of messages ready to be delivered in queues of this virtual host
	MessagesReady        int         `json:"messages_ready"`
	MessagesReadyDetails RateDetails `json:"messages_ready_details"`

	// Total number of messages pending acknowledgement from consumers in this virtual host
	MessagesUnacknowledged        int         `json:"messages_unacknowledged"`
	MessagesUnacknowledgedDetails RateDetails `json:"messages_unacknowledged_details"`

	// Octets received
	RecvOct uint64 `json:"recv_oct"`
	// Octets sent
	SendOct        uint64      `json:"send_oct"`
	RecvCount      uint64      `json:"recv_cnt"`
	SendCount      uint64      `json:"send_cnt"`
	SendPending    uint64      `json:"send_pend"`
	RecvOctDetails RateDetails `json:"recv_oct_details"`
	SendOctDetails RateDetails `json:"send_oct_details"`
}

// Returns a list of virtual hosts.
func (c *Client) ListVhosts() (rec []VhostInfo, err error) {
	req, err := newGETRequest(c, "vhosts")
	if err != nil {
		return []VhostInfo{}, err
	}

	if err = executeAndParseRequest(c, req, &rec); err != nil {
		return []VhostInfo{}, err
	}

	return rec, nil
}

//
// GET /api/vhosts/{name}
//

// Returns information about a specific virtual host.
func (c *Client) GetVhost(vhostname string) (rec *VhostInfo, err error) {
	req, err := newGETRequest(c, "vhosts/"+url.QueryEscape(vhostname))
	if err != nil {
		return nil, err
	}

	if err = executeAndParseRequest(c, req, &rec); err != nil {
		return nil, err
	}

	return rec, nil
}

//
// PUT /api/vhosts/{name}
//

// Settings used to create or modify virtual hosts.
type VhostSettings struct {
	// True if tracing should be enabled.
	Tracing bool `json:"tracing"`
}

// Creates or updates a virtual host.
func (c *Client) PutVhost(vhostname string, settings VhostSettings) (res *http.Response, err error) {
	body, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}

	req, err := newRequestWithBody(c, "PUT", "vhosts/"+url.QueryEscape(vhostname), body)
	if err != nil {
		return nil, err
	}

	res, err = executeRequest(c, req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

//
// DELETE /api/vhosts/{name}
//

// Deletes a virtual host.
func (c *Client) DeleteVhost(vhostname string) (res *http.Response, err error) {
	req, err := newRequestWithBody(c, "DELETE", "vhosts/"+url.QueryEscape(vhostname), nil)
	if err != nil {
		return nil, err
	}

	res, err = executeRequest(c, req)
	if err != nil {
		return nil, err
	}

	return res, nil
}
