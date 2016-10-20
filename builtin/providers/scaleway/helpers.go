package scaleway

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
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

// deleteRunningServer terminates the server and waits until it is removed.
func deleteRunningServer(scaleway *api.ScalewayAPI, server *api.ScalewayServer) error {
	err := scaleway.PostServerAction(server.Identifier, "terminate")

	if err != nil {
		if serr, ok := err.(api.ScalewayAPIError); ok {
			if serr.StatusCode == 404 {
				return nil
			}
		}

		return err
	}

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := scaleway.GetServer(server.Identifier)

		if err == nil {
			return resource.RetryableError(fmt.Errorf("Waiting for server %q to be deleted", server.Identifier))
		}

		if serr, ok := err.(api.ScalewayAPIError); ok {
			if serr.StatusCode == 404 {
				return nil
			}
		}

		return resource.RetryableError(err)
	})
}

// deleteStoppedServer needs to cleanup attached root volumes. this is not done
// automatically by Scaleway
func deleteStoppedServer(scaleway *api.ScalewayAPI, server *api.ScalewayServer) error {
	if err := scaleway.DeleteServer(server.Identifier); err != nil {
		return err
	}

	if rootVolume, ok := server.Volumes["0"]; ok {
		if err := scaleway.DeleteVolume(rootVolume.Identifier); err != nil {
			return err
		}
	}
	return nil
}

// NOTE copied from github.com/scaleway/scaleway-cli/pkg/api/helpers.go
// the helpers.go file pulls in quite a lot dependencies, and they're just convenience wrappers anyway

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
