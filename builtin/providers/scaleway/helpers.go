package scaleway

import (
	"log"
	"time"

	"github.com/scaleway/scaleway-cli/pkg/api"
)

// Bool returns a pointer to of the bool value passed in.
func Bool(val bool) *bool {
	return &val
}

// String returns a pointer to of the string value passed in.
func String(val string) *string {
	return &val
}

// NOTE copied from github.com/scaleway/scaleway-cli/pkg/api/helpers.go
// the helpers.go file pulls in quite a lot dependencies, and they're just convenience wrappers anyway

func deleteServerSafe(s *api.ScalewayAPI, serverID string) error {
	server, err := s.GetServer(serverID)
	if err != nil {
		return err
	}

	if server.State != "stopped" {
		if err := s.PostServerAction(serverID, "poweroff"); err != nil {
			return err
		}
		if err := waitForServerState(s, serverID, "stopped"); err != nil {
			return err
		}
	}

	if err := s.DeleteServer(serverID); err != nil {
		return err
	}
	if rootVolume, ok := server.Volumes["0"]; ok {
		if err := s.DeleteVolume(rootVolume.Identifier); err != nil {
			return err
		}
	}

	return nil
}

func waitForServerState(s *api.ScalewayAPI, serverID string, targetState string) error {
	var server *api.ScalewayServer
	var err error

	var currentState string

	for {
		server, err = s.GetServer(serverID)
		if err != nil {
			return err
		}
		if currentState != server.State {
			log.Printf("[DEBUG] Server changed state to %q\n", server.State)
			currentState = server.State
		}
		if server.State == targetState {
			break
		}
		time.Sleep(1 * time.Second)
	}

	return nil
}
