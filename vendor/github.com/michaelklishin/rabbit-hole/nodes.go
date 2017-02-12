package rabbithole

import (
	"net/url"
)

// TODO: this probably should be fixed in RabbitMQ management plugin
type OsPid string

type NameDescriptionEnabled struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

type AuthMechanism NameDescriptionEnabled

type ExchangeType NameDescriptionEnabled

type NameDescriptionVersion struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type ErlangApp NameDescriptionVersion

type NodeInfo struct {
	Name      string `json:"name"`
	NodeType  string `json:"type"`
	IsRunning bool   `json:"running"`
	OsPid     OsPid  `json:"os_pid"`

	FdUsed        int  `json:"fd_used"`
	FdTotal       int  `json:"fd_total"`
	SocketsUsed   int  `json:"sockets_used"`
	SocketsTotal  int  `json:"sockets_total"`
	MemUsed       int  `json:"mem_used"`
	MemLimit      int  `json:"mem_limit"`
	MemAlarm      bool `json:"mem_alarm"`
	DiskFree      int  `json:"disk_free"`
	DiskFreeLimit int  `json:"disk_free_limit"`
	DiskFreeAlarm bool `json:"disk_free_alarm"`

	// Erlang scheduler run queue length
	RunQueueLength uint32 `json:"run_queue"`
	Processors     uint32 `json:"processors"`
	Uptime         uint64 `json:"uptime"`

	ExchangeTypes  []ExchangeType  `json:"exchange_types"`
	AuthMechanisms []AuthMechanism `json:"auth_mechanisms"`
	ErlangApps     []ErlangApp     `json:"applications"`
	Contexts       []BrokerContext `json:"contexts"`
}

//
// GET /api/nodes
//

func (c *Client) ListNodes() (rec []NodeInfo, err error) {
	req, err := newGETRequest(c, "nodes")
	if err != nil {
		return []NodeInfo{}, err
	}

	if err = executeAndParseRequest(c, req, &rec); err != nil {
		return nil, err
	}

	return rec, nil
}

//
// GET /api/nodes/{name}
//

// {
//   "partitions": [],
//   "os_pid": "39292",
//   "fd_used": 35,
//   "fd_total": 256,
//   "sockets_used": 4,
//   "sockets_total": 138,
//   "mem_used": 69964432,
//   "mem_limit": 2960660889,
//   "mem_alarm": false,
//   "disk_free_limit": 50000000,
//   "disk_free": 188362731520,
//   "disk_free_alarm": false,
//   "proc_used": 370,
//   "proc_total": 1048576,
//   "statistics_level": "fine",
//   "uptime": 98355255,
//   "run_queue": 0,
//   "processors": 8,
//   "exchange_types": [
//     {
//       "name": "topic",
//       "description": "AMQP topic exchange, as per the AMQP specification",
//       "enabled": true
//     },
//     {
//       "name": "x-consistent-hash",
//       "description": "Consistent Hashing Exchange",
//       "enabled": true
//     },
//     {
//       "name": "fanout",
//       "description": "AMQP fanout exchange, as per the AMQP specification",
//       "enabled": true
//     },
//     {
//       "name": "direct",
//       "description": "AMQP direct exchange, as per the AMQP specification",
//       "enabled": true
//     },
//     {
//       "name": "headers",
//       "description": "AMQP headers exchange, as per the AMQP specification",
//       "enabled": true
//     }
//   ],
//   "auth_mechanisms": [
//     {
//       "name": "AMQPLAIN",
//       "description": "QPid AMQPLAIN mechanism",
//       "enabled": true
//     },
//     {
//       "name": "PLAIN",
//       "description": "SASL PLAIN authentication mechanism",
//       "enabled": true
//     },
//     {
//       "name": "RABBIT-CR-DEMO",
//       "description": "RabbitMQ Demo challenge-response authentication mechanism",
//       "enabled": false
//     }
//   ],
//   "applications": [
//     {
//       "name": "amqp_client",
//       "description": "RabbitMQ AMQP Client",
//       "version": "3.2.0"
//     },
//     {
//       "name": "asn1",
//       "description": "The Erlang ASN1 compiler version 2.0.3",
//       "version": "2.0.3"
//     },
//     {
//       "name": "cowboy",
//       "description": "Small, fast, modular HTTP server.",
//       "version": "0.5.0-rmq3.2.0-git4b93c2d"
//     },
//     {
//       "name": "crypto",
//       "description": "CRYPTO version 2",
//       "version": "3.1"
//     },
//     {
//       "name": "inets",
//       "description": "INETS  CXC 138 49",
//       "version": "5.9.6"
//     },
//     {
//       "name": "kernel",
//       "description": "ERTS  CXC 138 10",
//       "version": "2.16.3"
//     },
//     {
//       "name": "mnesia",
//       "description": "MNESIA  CXC 138 12",
//       "version": "4.10"
//     },
//     {
//       "name": "mochiweb",
//       "description": "MochiMedia Web Server",
//       "version": "2.7.0-rmq3.2.0-git680dba8"
//     },
//     {
//       "name": "os_mon",
//       "description": "CPO  CXC 138 46",
//       "version": "2.2.13"
//     },
//     {
//       "name": "public_key",
//       "description": "Public key infrastructure",
//       "version": "0.20"
//     },
//     {
//       "name": "rabbit",
//       "description": "RabbitMQ",
//       "version": "3.2.0"
//     },
//     {
//       "name": "rabbitmq_consistent_hash_exchange",
//       "description": "Consistent Hash Exchange Type",
//       "version": "3.2.0"
//     },
//     {
//       "name": "rabbitmq_management",
//       "description": "RabbitMQ Management Console",
//       "version": "3.2.0"
//     },
//     {
//       "name": "rabbitmq_management_agent",
//       "description": "RabbitMQ Management Agent",
//       "version": "3.2.0"
//     },
//     {
//       "name": "rabbitmq_mqtt",
//       "description": "RabbitMQ MQTT Adapter",
//       "version": "3.2.0"
//     },
//     {
//       "name": "rabbitmq_shovel",
//       "description": "Data Shovel for RabbitMQ",
//       "version": "3.2.0"
//     },
//     {
//       "name": "rabbitmq_shovel_management",
//       "description": "Shovel Status",
//       "version": "3.2.0"
//     },
//     {
//       "name": "rabbitmq_stomp",
//       "description": "Embedded Rabbit Stomp Adapter",
//       "version": "3.2.0"
//     },
//     {
//       "name": "rabbitmq_web_dispatch",
//       "description": "RabbitMQ Web Dispatcher",
//       "version": "3.2.0"
//     },
//     {
//       "name": "rabbitmq_web_stomp",
//       "description": "Rabbit WEB-STOMP - WebSockets to Stomp adapter",
//       "version": "3.2.0"
//     },
//     {
//       "name": "sasl",
//       "description": "SASL  CXC 138 11",
//       "version": "2.3.3"
//     },
//     {
//       "name": "sockjs",
//       "description": "SockJS",
//       "version": "0.3.4-rmq3.2.0-git3132eb9"
//     },
//     {
//       "name": "ssl",
//       "description": "Erlang\/OTP SSL application",
//       "version": "5.3.1"
//     },
//     {
//       "name": "stdlib",
//       "description": "ERTS  CXC 138 10",
//       "version": "1.19.3"
//     },
//     {
//       "name": "webmachine",
//       "description": "webmachine",
//       "version": "1.10.3-rmq3.2.0-gite9359c7"
//     },
//     {
//       "name": "xmerl",
//       "description": "XML parser",
//       "version": "1.3.4"
//     }
//   ],
//   "contexts": [
//     {
//       "description": "Redirect to port 15672",
//       "path": "\/",
//       "port": 55672,
//       "ignore_in_use": true
//     },
//     {
//       "description": "RabbitMQ Management",
//       "path": "\/",
//       "port": 15672
//     }
//   ],
//   "name": "rabbit@mercurio",
//   "type": "disc",
//   "running": true
// }

func (c *Client) GetNode(name string) (rec *NodeInfo, err error) {
	req, err := newGETRequest(c, "nodes/"+url.QueryEscape(name))
	if err != nil {
		return nil, err
	}

	if err = executeAndParseRequest(c, req, &rec); err != nil {
		return nil, err
	}

	return rec, nil
}
