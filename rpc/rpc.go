package rpc

import (
	"errors"
	"fmt"
	"net/rpc"
	"sync"

	"github.com/hashicorp/terraform/terraform"
)

// nextId is the next ID to use for names registered.
var nextId uint32 = 0
var nextLock sync.Mutex

// Register registers a Terraform thing with the RPC server and returns
// the name it is registered under.
func Register(server *rpc.Server, thing interface{}) (name string, err error) {
	nextLock.Lock()
	defer nextLock.Unlock()

	switch t := thing.(type) {
	case terraform.ResourceProvider:
		name = fmt.Sprintf("Terraform%d", nextId)
		err = server.RegisterName(name, &ResourceProviderServer{Provider: t})
	case terraform.ResourceProvisioner:
		name = fmt.Sprintf("Terraform%d", nextId)
		err = server.RegisterName(name, &ResourceProvisionerServer{Provisioner: t})
	default:
		return "", errors.New("Unknown type to register for RPC server.")
	}

	nextId += 1
	return
}
