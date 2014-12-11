package remote

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/terraform"
)

var (
	// ErrConflict is used to indicate the upload was rejected
	// due to a conflict on the state
	ErrConflict = fmt.Errorf("Conflicting state file")

	// ErrServerNewer is used to indicate the serial number of
	// the state is newer on the server side
	ErrServerNewer = fmt.Errorf("Server-side Serial is newer")

	// ErrRequireAuth is used if the remote server requires
	// authentication and none is provided
	ErrRequireAuth = fmt.Errorf("Remote server requires authentication")

	// ErrInvalidAuth is used if we provide authentication which
	// is not valid
	ErrInvalidAuth = fmt.Errorf("Invalid authentication")

	// ErrRemoteInternal is used if we get an internal error
	// from the remote server
	ErrRemoteInternal = fmt.Errorf("Remote server reporting internal error")
)

type RemoteClient interface {
	GetState() (*RemoteStatePayload, error)
	PutState(state []byte, force bool) error
	DeleteState() error
}

// RemoteStatePayload is used to return the remote state
// along with associated meta data when we do a remote fetch.
type RemoteStatePayload struct {
	MD5   []byte
	State []byte
}

// NewClientByState is used to construct a client from
// our remote state.
func NewClientByState(remote *terraform.RemoteState) (RemoteClient, error) {
	return NewClientByType(remote.Type, remote.Config)
}

// NewClientByType is used to construct a RemoteClient
// based on the configured type.
func NewClientByType(ctype string, conf map[string]string) (RemoteClient, error) {
	ctype = strings.ToLower(ctype)
	switch ctype {
	case "atlas":
		return NewAtlasRemoteClient(conf)
	case "consul":
		return NewConsulRemoteClient(conf)
	case "http":
		return NewHTTPRemoteClient(conf)
	default:
		return nil, fmt.Errorf("Unknown remote client type '%s'", ctype)
	}
}
