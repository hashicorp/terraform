package packet

import (
	"github.com/packethost/packngo"
)

const (
	consumerToken = "aZ9GmqHTPtxevvFq9SK3Pi2yr9YCbRzduCSXF2SNem5sjB91mDq7Th3ZwTtRqMWZ"
)

type Config struct {
	AuthToken string
}

// Client() returns a new client for accessing packet.
func (c *Config) Client() *packngo.Client {
	return packngo.NewClient(consumerToken, c.AuthToken)
}
