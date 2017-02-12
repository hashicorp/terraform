package rabbithole

import "net/url"

// Brief (very incomplete) connection information.
type BriefConnectionDetails struct {
	// Connection name
	Name string `json:"name"`
	// Client port
	PeerPort Port `json:"peer_port"`
	// Client host
	PeerHost string `json:"peer_host"`
}

type ChannelInfo struct {
	// Channel number
	Number int `json:"number"`
	// Channel name
	Name string `json:"name"`

	// basic.qos (prefetch count) value used
	PrefetchCount int `json:"prefetch_count"`
	// How many consumers does this channel have
	ConsumerCount int `json:"consumer_count"`

	// Number of unacknowledged messages on this channel
	UnacknowledgedMessageCount int `json:"messages_unacknowledged"`
	// Number of messages on this channel unconfirmed to publishers
	UnconfirmedMessageCount int `json:"messages_unconfirmed"`
	// Number of messages on this channel uncommited to message store
	UncommittedMessageCount int `json:"messages_uncommitted"`
	// Number of acks on this channel uncommited to message store
	UncommittedAckCount int `json:"acks_uncommitted"`

	// TODO(mk): custom deserializer to date/time?
	IdleSince string `json:"idle_since"`

	// True if this channel uses publisher confirms
	UsesPublisherConfirms bool `json:"confirm"`
	// True if this channel uses transactions
	Transactional bool `json:"transactional"`
	// True if this channel is blocked via channel.flow
	ClientFlowBlocked bool `json:"client_flow_blocked"`

	User  string `json:"user"`
	Vhost string `json:"vhost"`
	Node  string `json:"node"`

	ConnectionDetails BriefConnectionDetails `json:"connection_details"`
}

//
// GET /api/channels
//

// Returns information about all open channels.
func (c *Client) ListChannels() (rec []ChannelInfo, err error) {
	req, err := newGETRequest(c, "channels")
	if err != nil {
		return []ChannelInfo{}, err
	}

	if err = executeAndParseRequest(c, req, &rec); err != nil {
		return []ChannelInfo{}, err
	}

	return rec, nil
}

//
// GET /api/channels/{name}
//

// Returns channel information.
func (c *Client) GetChannel(name string) (rec *ChannelInfo, err error) {
	req, err := newGETRequest(c, "channels/"+url.QueryEscape(name))
	if err != nil {
		return nil, err
	}

	if err = executeAndParseRequest(c, req, &rec); err != nil {
		return nil, err
	}

	return rec, nil
}
