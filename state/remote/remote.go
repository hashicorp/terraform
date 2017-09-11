package remote

import (
	"fmt"

	"github.com/hashicorp/terraform/state"
)

// Client is the interface that must be implemented for a remote state
// driver. It supports dumb put/get/delete, and the higher level structs
// handle persisting the state properly here.
type Client interface {
	Get() (*Payload, error)
	Put([]byte) error
	Delete() error
}

// ClientLocker is an optional interface that allows a remote state
// backend to enable state lock/unlock.
type ClientLocker interface {
	Client
	state.Locker
}

// Payload is the return value from the remote state storage.
type Payload struct {
	MD5  []byte
	Data []byte
}

// Factory is the factory function to create a remote client.
type Factory func(map[string]string) (Client, error)

// NewClient returns a new Client with the given type and configuration.
// The client is looked up in the BuiltinClients variable.
func NewClient(t string, conf map[string]string) (Client, error) {
	f, ok := BuiltinClients[t]
	if !ok {
		return nil, fmt.Errorf("unknown remote client type: %s", t)
	}

	return f(conf)
}

// BuiltinClients is the list of built-in clients that can be used with
// NewClient.
var BuiltinClients = map[string]Factory{
	"artifactory": artifactoryFactory,
	"etcd":        etcdFactory,
	"http":        httpFactory,
	"local":       fileFactory,
	"manta":       mantaFactory,
}
