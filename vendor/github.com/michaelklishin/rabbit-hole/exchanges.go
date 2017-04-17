package rabbithole

import (
	"encoding/json"
	"net/http"
	"net/url"
)

//
// GET /api/exchanges
//

type IngressEgressStats struct {
	PublishIn        int         `json:"publish_in"`
	PublishInDetails RateDetails `json:"publish_in_details"`

	PublishOut        int         `json:"publish_out"`
	PublishOutDetails RateDetails `json:"publish_out_details"`
}

type ExchangeInfo struct {
	Name       string                 `json:"name"`
	Vhost      string                 `json:"vhost"`
	Type       string                 `json:"type"`
	Durable    bool                   `json:"durable"`
	AutoDelete bool                   `json:"auto_delete"`
	Internal   bool                   `json:"internal"`
	Arguments  map[string]interface{} `json:"arguments"`

	MessageStats IngressEgressStats `json:"message_stats"`
}

type ExchangeSettings struct {
	Type       string                 `json:"type"`
	Durable    bool                   `json:"durable"`
	AutoDelete bool                   `json:"auto_delete"`
	Arguments  map[string]interface{} `json:"arguments"`
}

func (c *Client) ListExchanges() (rec []ExchangeInfo, err error) {
	req, err := newGETRequest(c, "exchanges")
	if err != nil {
		return []ExchangeInfo{}, err
	}

	if err = executeAndParseRequest(c, req, &rec); err != nil {
		return []ExchangeInfo{}, err
	}

	return rec, nil
}

//
// GET /api/exchanges/{vhost}
//

func (c *Client) ListExchangesIn(vhost string) (rec []ExchangeInfo, err error) {
	req, err := newGETRequest(c, "exchanges/"+url.QueryEscape(vhost))
	if err != nil {
		return []ExchangeInfo{}, err
	}

	if err = executeAndParseRequest(c, req, &rec); err != nil {
		return []ExchangeInfo{}, err
	}

	return rec, nil
}

//
// GET /api/exchanges/{vhost}/{name}
//

// Example response:
//
// {
//   "incoming": [
//     {
//       "stats": {
//         "publish": 2760,
//         "publish_details": {
//           "rate": 20
//         }
//       },
//       "channel_details": {
//         "name": "127.0.0.1:46928 -> 127.0.0.1:5672 (2)",
//         "number": 2,
//         "connection_name": "127.0.0.1:46928 -> 127.0.0.1:5672",
//         "peer_port": 46928,
//         "peer_host": "127.0.0.1"
//       }
//     }
//   ],
//   "outgoing": [
//     {
//       "stats": {
//         "publish": 1280,
//         "publish_details": {
//           "rate": 20
//         }
//       },
//       "queue": {
//         "name": "amq.gen-7NhO_yRr4lDdp-8hdnvfuw",
//         "vhost": "rabbit\/hole"
//       }
//     }
//   ],
//   "message_stats": {
//     "publish_in": 2760,
//     "publish_in_details": {
//       "rate": 20
//     },
//     "publish_out": 1280,
//     "publish_out_details": {
//       "rate": 20
//     }
//   },
//   "name": "amq.fanout",
//   "vhost": "rabbit\/hole",
//   "type": "fanout",
//   "durable": true,
//   "auto_delete": false,
//   "internal": false,
//   "arguments": {
//   }
// }

type ExchangeIngressDetails struct {
	Stats          MessageStats      `json:"stats"`
	ChannelDetails PublishingChannel `json:"channel_details"`
}

type PublishingChannel struct {
	Number         int    `json:"number"`
	Name           string `json:"name"`
	ConnectionName string `json:"connection_name"`
	PeerPort       Port   `json:"peer_port"`
	PeerHost       string `json:"peer_host"`
}

type NameAndVhost struct {
	Name  string `json:"name"`
	Vhost string `json:"vhost"`
}

type ExchangeEgressDetails struct {
	Stats MessageStats `json:"stats"`
	Queue NameAndVhost `json:"queue"`
}

type DetailedExchangeInfo struct {
	Name       string                 `json:"name"`
	Vhost      string                 `json:"vhost"`
	Type       string                 `json:"type"`
	Durable    bool                   `json:"durable"`
	AutoDelete bool                   `json:"auto_delete"`
	Internal   bool                   `json:"internal"`
	Arguments  map[string]interface{} `json:"arguments"`

	Incoming []ExchangeIngressDetails `json:"incoming"`
	Outgoing []ExchangeEgressDetails  `json:"outgoing"`
}

func (c *Client) GetExchange(vhost, exchange string) (rec *DetailedExchangeInfo, err error) {
	req, err := newGETRequest(c, "exchanges/"+url.QueryEscape(vhost)+"/"+exchange)
	if err != nil {
		return nil, err
	}

	if err = executeAndParseRequest(c, req, &rec); err != nil {
		return nil, err
	}

	return rec, nil
}

//
// PUT /api/exchanges/{vhost}/{exchange}
//

func (c *Client) DeclareExchange(vhost, exchange string, info ExchangeSettings) (res *http.Response, err error) {
	if info.Arguments == nil {
		info.Arguments = make(map[string]interface{})
	}
	body, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}

	req, err := newRequestWithBody(c, "PUT", "exchanges/"+url.QueryEscape(vhost)+"/"+url.QueryEscape(exchange), body)
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
// DELETE /api/exchanges/{vhost}/{name}
//

func (c *Client) DeleteExchange(vhost, exchange string) (res *http.Response, err error) {
	req, err := newRequestWithBody(c, "DELETE", "exchanges/"+url.QueryEscape(vhost)+"/"+url.QueryEscape(exchange), nil)
	if err != nil {
		return nil, err
	}

	res, err = executeRequest(c, req)
	if err != nil {
		return nil, err
	}

	return res, nil
}
