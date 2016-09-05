package rabbithole

import (
	"encoding/json"
	"net/http"
	"net/url"
)

// Information about backing queue (queue storage engine).
type BackingQueueStatus struct {
	Q1 int `json:"q1"`
	Q2 int `json:"q2"`
	Q3 int `json:"q3"`
	Q4 int `json:"q4"`
	// Total queue length
	Length int64 `json:"len"`
	// Number of pending acks from consumers
	PendingAcks int64 `json:"pending_acks"`
	// Number of messages held in RAM
	RAMMessageCount int64 `json:"ram_msg_count"`
	// Number of outstanding acks held in RAM
	RAMAckCount int64 `json:"ram_ack_count"`
	// Number of persistent messages in the store
	PersistentCount int64 `json:"persistent_count"`
	// Average ingress (inbound) rate, not including messages
	// that straight through to auto-acking consumers.
	AverageIngressRate float64 `json:"avg_ingress_rate"`
	// Average egress (outbound) rate, not including messages
	// that straight through to auto-acking consumers.
	AverageEgressRate float64 `json:"avg_egress_rate"`
	// rate at which unacknowledged message records enter RAM,
	// e.g. because messages are delivered requiring acknowledgement
	AverageAckIngressRate float32 `json:"avg_ack_ingress_rate"`
	// rate at which unacknowledged message records leave RAM,
	// e.g. because acks arrive or unacked messages are paged out
	AverageAckEgressRate float32 `json:"avg_ack_egress_rate"`
}

type OwnerPidDetails struct {
	Name     string `json:"name"`
	PeerPort Port   `json:"peer_port"`
	PeerHost string `json:"peer_host"`
}

type QueueInfo struct {
	// Queue name
	Name string `json:"name"`
	// Virtual host this queue belongs to
	Vhost string `json:"vhost"`
	// Is this queue durable?
	Durable bool `json:"durable"`
	// Is this queue auto-delted?
	AutoDelete bool `json:"auto_delete"`
	// Extra queue arguments
	Arguments map[string]interface{} `json:"arguments"`

	// RabbitMQ node that hosts master for this queue
	Node string `json:"node"`
	// Queue status
	Status string `json:"status"`

	// Total amount of RAM used by this queue
	Memory int64 `json:"memory"`
	// How many consumers this queue has
	Consumers int `json:"consumers"`
	// Utilisation of all the consumers
	ConsumerUtilisation float64 `json:"consumer_utilisation"`
	// If there is an exclusive consumer, its consumer tag
	ExclusiveConsumerTag string `json:"exclusive_consumer_tag"`

	// Policy applied to this queue, if any
	Policy string `json:"policy"`

	// Total bytes of messages in this queues
	MessagesBytes           int64 `json:"message_bytes"`
	MessagesBytesPersistent int64 `json:"message_bytes_persistent"`
	MessagesBytesRAM        int64 `json:"message_bytes_ram"`

	// Total number of messages in this queue
	Messages           int         `json:"messages"`
	MessagesDetails    RateDetails `json:"messages_details"`
	MessagesPersistent int         `json:"messages_persistent"`
	MessagesRAM        int         `json:"messages_ram"`

	// Number of messages ready to be delivered
	MessagesReady        int         `json:"messages_ready"`
	MessagesReadyDetails RateDetails `json:"messages_ready_details"`

	// Number of messages delivered and pending acknowledgements from consumers
	MessagesUnacknowledged        int         `json:"messages_unacknowledged"`
	MessagesUnacknowledgedDetails RateDetails `json:"messages_unacknowledged_details"`

	MessageStats MessageStats `json:"message_stats"`

	OwnerPidDetails OwnerPidDetails `json:"owner_pid_details"`

	BackingQueueStatus BackingQueueStatus `json:"backing_queue_status"`
}

type DetailedQueueInfo QueueInfo

//
// GET /api/queues
//

// [
//   {
//     "owner_pid_details": {
//       "name": "127.0.0.1:46928 -> 127.0.0.1:5672",
//       "peer_port": 46928,
//       "peer_host": "127.0.0.1"
//     },
//     "message_stats": {
//       "publish": 19830,
//       "publish_details": {
//         "rate": 5
//       }
//     },
//     "messages": 15,
//     "messages_details": {
//       "rate": 0
//     },
//     "messages_ready": 15,
//     "messages_ready_details": {
//       "rate": 0
//     },
//     "messages_unacknowledged": 0,
//     "messages_unacknowledged_details": {
//       "rate": 0
//     },
//     "policy": "",
//     "exclusive_consumer_tag": "",
//     "consumers": 0,
//     "memory": 143112,
//     "backing_queue_status": {
//       "q1": 0,
//       "q2": 0,
//       "delta": [
//         "delta",
//         "undefined",
//         0,
//         "undefined"
//       ],
//       "q3": 0,
//       "q4": 15,
//       "len": 15,
//       "pending_acks": 0,
//       "target_ram_count": "infinity",
//       "ram_msg_count": 15,
//       "ram_ack_count": 0,
//       "next_seq_id": 19830,
//       "persistent_count": 0,
//       "avg_ingress_rate": 4.9920127795527,
//       "avg_egress_rate": 4.9920127795527,
//       "avg_ack_ingress_rate": 0,
//       "avg_ack_egress_rate": 0
//     },
//     "status": "running",
//     "name": "amq.gen-QLEaT5Rn_ogbN3O8ZOQt3Q",
//     "vhost": "rabbit\/hole",
//     "durable": false,
//     "auto_delete": false,
//     "arguments": {
//       "x-message-ttl": 5000
//     },
//     "node": "rabbit@marzo"
//   }
// ]

func (c *Client) ListQueues() (rec []QueueInfo, err error) {
	req, err := newGETRequest(c, "queues")
	if err != nil {
		return []QueueInfo{}, err
	}

	if err = executeAndParseRequest(c, req, &rec); err != nil {
		return []QueueInfo{}, err
	}

	return rec, nil
}

//
// GET /api/queues/{vhost}
//

func (c *Client) ListQueuesIn(vhost string) (rec []QueueInfo, err error) {
	req, err := newGETRequest(c, "queues/"+url.QueryEscape(vhost))
	if err != nil {
		return []QueueInfo{}, err
	}

	if err = executeAndParseRequest(c, req, &rec); err != nil {
		return []QueueInfo{}, err
	}

	return rec, nil
}

//
// GET /api/queues/{vhost}/{name}
//

func (c *Client) GetQueue(vhost, queue string) (rec *DetailedQueueInfo, err error) {
	req, err := newGETRequest(c, "queues/"+url.QueryEscape(vhost)+"/"+queue)
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

type QueueSettings struct {
	Durable    bool                   `json:"durable"`
	AutoDelete bool                   `json:"auto_delete"`
	Arguments  map[string]interface{} `json:"arguments"`
}

func (c *Client) DeclareQueue(vhost, queue string, info QueueSettings) (res *http.Response, err error) {
	if info.Arguments == nil {
		info.Arguments = make(map[string]interface{})
	}
	body, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}

	req, err := newRequestWithBody(c, "PUT", "queues/"+url.QueryEscape(vhost)+"/"+url.QueryEscape(queue), body)
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
// DELETE /api/queues/{vhost}/{name}
//

func (c *Client) DeleteQueue(vhost, queue string) (res *http.Response, err error) {
	req, err := newRequestWithBody(c, "DELETE", "queues/"+url.QueryEscape(vhost)+"/"+url.QueryEscape(queue), nil)
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
// DELETE /api/queues/{vhost}/{name}/contents
//

func (c *Client) PurgeQueue(vhost, queue string) (res *http.Response, err error) {
	req, err := newRequestWithBody(c, "DELETE", "queues/"+url.QueryEscape(vhost)+"/"+url.QueryEscape(queue)+"/contents", nil)
	if err != nil {
		return nil, err
	}

	res, err = executeRequest(c, req)
	if err != nil {
		return nil, err
	}

	return res, nil
}
