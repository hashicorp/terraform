package scaleway

import (
	"fmt"
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

func validateVolumeType(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if value != "l_ssd" {
		errors = append(errors, fmt.Errorf("%q must be l_ssd", k))
	}
	return
}

func validateVolumeSize(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	if value < 1 || value > 150 {
		errors = append(errors, fmt.Errorf("%q be more than 1 and less than 150", k))
	}
	return
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

	return resource.Retry(20*time.Minute, func() *resource.RetryError {
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

func waitForServerState(scaleway *api.ScalewayAPI, serverID, targetState string) error {
	return resource.Retry(20*time.Minute, func() *resource.RetryError {
		s, err := scaleway.GetServer(serverID)

		if err != nil {
			return resource.NonRetryableError(err)
		}

		if s.State != targetState {
			return resource.RetryableError(fmt.Errorf("Waiting for server to enter %q state", targetState))
		}

		return nil
	})
}
