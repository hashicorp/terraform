package rabbithole

import (
	"encoding/json"
	"net/http"
	"net/url"
)

// Federation definition: additional arguments
// added to the entities (queues, exchanges or both)
// that match a policy.
type FederationDefinition struct {
	Uri            string `json:"uri"`
	Expires        int    `json:"expires"`
	MessageTTL     int32  `json:"message-ttl"`
	MaxHops        int    `json:"max-hops"`
	PrefetchCount  int    `json:"prefetch-count"`
	ReconnectDelay int    `json:"reconnect-delay"`
	AckMode        string `json:"ack-mode"`
	TrustUserId    bool   `json:"trust-user-id"`
}

// Represents a configured Federation upstream.
type FederationUpstream struct {
	Definition FederationDefinition `json:"value"`
}

//
// PUT /api/parameters/federation-upstream/{vhost}/{upstream}
//

// Updates a federation upstream
func (c *Client) PutFederationUpstream(vhost string, upstreamName string, fDef FederationDefinition) (res *http.Response, err error) {
	fedUp := FederationUpstream{
		Definition: fDef,
	}
	body, err := json.Marshal(fedUp)
	if err != nil {
		return nil, err
	}

	req, err := newRequestWithBody(c, "PUT", "parameters/federation-upstream/"+url.QueryEscape(vhost)+"/"+url.QueryEscape(upstreamName), body)
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
// DELETE /api/parameters/federation-upstream/{vhost}/{name}
//

// Deletes a federation upstream.
func (c *Client) DeleteFederationUpstream(vhost, upstreamName string) (res *http.Response, err error) {
	req, err := newRequestWithBody(c, "DELETE", "parameters/federation-upstream/"+url.QueryEscape(vhost)+"/"+url.QueryEscape(upstreamName), nil)
	if err != nil {
		return nil, err
	}

	res, err = executeRequest(c, req)
	if err != nil {
		return nil, err
	}

	return res, nil
}
