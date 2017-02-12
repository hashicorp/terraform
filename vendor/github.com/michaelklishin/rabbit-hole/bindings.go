package rabbithole

import (
	"encoding/json"
	"net/http"
	"net/url"
)

//
// GET /api/bindings
//

// Example response:
//
// [
//   {
//     "source": "",
//     "vhost": "\/",
//     "destination": "amq.gen-Dzw36tPTm_VsmILY9oTG9w",
//     "destination_type": "queue",
//     "routing_key": "amq.gen-Dzw36tPTm_VsmILY9oTG9w",
//     "arguments": {
//
//     },
//     "properties_key": "amq.gen-Dzw36tPTm_VsmILY9oTG9w"
//   }
// ]

type BindingInfo struct {
	// Binding source (exchange name)
	Source string `json:"source"`
	Vhost  string `json:"vhost"`
	// Binding destination (queue or exchange name)
	Destination string `json:"destination"`
	// Destination type, either "queue" or "exchange"
	DestinationType string                 `json:"destination_type"`
	RoutingKey      string                 `json:"routing_key"`
	Arguments       map[string]interface{} `json:"arguments"`
	PropertiesKey   string                 `json:"properties_key"`
}

// Returns all bindings
func (c *Client) ListBindings() (rec []BindingInfo, err error) {
	req, err := newGETRequest(c, "bindings/")
	if err != nil {
		return []BindingInfo{}, err
	}

	if err = executeAndParseRequest(c, req, &rec); err != nil {
		return []BindingInfo{}, err
	}

	return rec, nil
}

//
// GET /api/bindings/{vhost}
//

// Returns all bindings in a virtual host.
func (c *Client) ListBindingsIn(vhost string) (rec []BindingInfo, err error) {
	req, err := newGETRequest(c, "bindings/"+url.QueryEscape(vhost))
	if err != nil {
		return []BindingInfo{}, err
	}

	if err = executeAndParseRequest(c, req, &rec); err != nil {
		return []BindingInfo{}, err
	}

	return rec, nil
}

//
// GET /api/queues/{vhost}/{queue}/bindings
//

// Example response:
// [
//   {"source":"",
//    "vhost":"/",
//    "destination":"amq.gen-H0tnavWatL7g7uU2q5cAPA",
//    "destination_type":"queue",
//    "routing_key":"amq.gen-H0tnavWatL7g7uU2q5cAPA",
//    "arguments":{},
//    "properties_key":"amq.gen-H0tnavWatL7g7uU2q5cAPA"},
//   {"source":"temp",
//    "vhost":"/",
//    "destination":"amq.gen-H0tnavWatL7g7uU2q5cAPA",
//    "destination_type":"queue",
//    "routing_key":"",
//    "arguments":{},
//    "properties_key":"~"}
// ]

// Returns all bindings of individual queue.
func (c *Client) ListQueueBindings(vhost, queue string) (rec []BindingInfo, err error) {
	req, err := newGETRequest(c, "queues/"+url.QueryEscape(vhost)+"/"+url.QueryEscape(queue)+"/bindings")
	if err != nil {
		return []BindingInfo{}, err
	}

	if err = executeAndParseRequest(c, req, &rec); err != nil {
		return []BindingInfo{}, err
	}

	return rec, nil
}

//
// POST /api/bindings/{vhost}/e/{source}/{destination_type}/{destination}
//

// DeclareBinding updates information about a binding between a source and a target
func (c *Client) DeclareBinding(vhost string, info BindingInfo) (res *http.Response, err error) {
	info.Vhost = vhost

	if info.Arguments == nil {
		info.Arguments = make(map[string]interface{})
	}
	body, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}

	req, err := newRequestWithBody(c, "POST", "bindings/"+url.QueryEscape(vhost)+"/e/"+url.QueryEscape(info.Source)+"/"+url.QueryEscape(string(info.DestinationType[0]))+"/"+url.QueryEscape(info.Destination), body)

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
// DELETE /api/bindings/{vhost}/e/{source}/{destination_type}/{destination}/{props}
//

// DeleteBinding delets an individual binding
func (c *Client) DeleteBinding(vhost string, info BindingInfo) (res *http.Response, err error) {
	req, err := newRequestWithBody(c, "DELETE", "bindings/"+url.QueryEscape(vhost)+"/e/"+url.QueryEscape(info.Source)+"/"+url.QueryEscape(string(info.DestinationType[0]))+"/"+url.QueryEscape(info.Destination)+"/"+url.QueryEscape(info.PropertiesKey), nil)
	if err != nil {
		return nil, err
	}

	res, err = executeRequest(c, req)
	if err != nil {
		return nil, err
	}

	return res, nil
}
