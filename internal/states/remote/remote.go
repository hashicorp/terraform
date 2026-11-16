// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package remote

import (
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Client is the interface that must be implemented for a remote state
// driver. It supports dumb put/get/delete, and the higher level structs
// handle persisting the state properly here.
type Client interface {
	Get() (*Payload, tfdiags.Diagnostics)
	Put([]byte) tfdiags.Diagnostics
	Delete() tfdiags.Diagnostics
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
