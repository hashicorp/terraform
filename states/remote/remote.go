package remote

import (
	"fmt"

	"github.com/hashicorp/terraform/states/statemgr"
)

// Client is the interface that must be implemented for a remote state
// driver. It supports dumb put/get/delete, and the higher level structs
// handle persisting the state properly here.
type Client interface {
	Get(workspace string) (*Payload, error)
	Put(workspace string, data []byte) error
	Delete(workspace string) error
	// List worksapces.
	List() (string, error)
}

// ClientForcePusher is an optional interface that allows a remote
// state to force push by managing a flag on the client that is
// toggled on by a call to EnableForcePush.
type ClientForcePusher interface {
	Client
	EnableForcePush()
}

// ClientLocker is an optional interface that allows a remote state
// backend to enable state lock/unlock.
type ClientLocker interface {
	Client
	statemgr.Locker
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
var BuiltinClients = map[string]Factory{}
