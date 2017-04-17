package rabbithole

//
// GET /api/overview
//

type QueueTotals struct {
	Messages        int         `json:"messages"`
	MessagesDetails RateDetails `json:"messages_details"`

	MessagesReady        int         `json:"messages_ready"`
	MessagesReadyDetails RateDetails `json:"messages_ready_details"`

	MessagesUnacknowledged        int         `json:"messages_unacknowledged"`
	MessagesUnacknowledgedDetails RateDetails `json:"messages_unacknowledged_details"`
}

type ObjectTotals struct {
	Consumers   int `json:"consumers"`
	Queues      int `json:"queues"`
	Exchanges   int `json:"exchanges"`
	Connections int `json:"connections"`
	Channels    int `json:"channels"`
}

type Listener struct {
	Node      string `json:"node"`
	Protocol  string `json:"protocol"`
	IpAddress string `json:"ip_address"`
	Port      Port   `json:"port"`
}

type Overview struct {
	ManagementVersion string          `json:"management_version"`
	StatisticsLevel   string          `json:"statistics_level"`
	RabbitMQVersion   string          `json:"rabbitmq_version"`
	ErlangVersion     string          `json:"erlang_version"`
	FullErlangVersion string          `json:"erlang_full_version"`
	ExchangeTypes     []ExchangeType  `json:"exchange_types"`
	MessageStats      MessageStats    `json:"message_stats"`
	QueueTotals       QueueTotals     `json:"queue_totals"`
	ObjectTotals      ObjectTotals    `json:"object_totals"`
	Node              string          `json:"node"`
	StatisticsDBNode  string          `json:"statistics_db_node"`
	Listeners         []Listener      `json:"listeners"`
	Contexts          []BrokerContext `json:"contexts"`
}

func (c *Client) Overview() (rec *Overview, err error) {
	req, err := newGETRequest(c, "overview")
	if err != nil {
		return nil, err
	}

	if err = executeAndParseRequest(c, req, &rec); err != nil {
		return nil, err
	}

	return rec, nil
}

//
// GET /api/whoami
//

type WhoamiInfo struct {
	Name        string `json:"name"`
	Tags        string `json:"tags"`
	AuthBackend string `json:"auth_backend"`
}

func (c *Client) Whoami() (rec *WhoamiInfo, err error) {
	req, err := newGETRequest(c, "whoami")
	if err != nil {
		return nil, err
	}

	if err = executeAndParseRequest(c, req, &rec); err != nil {
		return nil, err
	}

	return rec, nil
}
