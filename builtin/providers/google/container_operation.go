package google

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"google.golang.org/api/container/v1"
)

type ContainerOperationWaiter struct {
	Service *container.Service
	Op      *container.Operation
	Project string
	Zone    string
}

func (w *ContainerOperationWaiter) Conf() *resource.StateChangeConf {
	return &resource.StateChangeConf{
		Pending: []string{"PENDING", "RUNNING"},
		Target:  []string{"DONE"},
		Refresh: w.RefreshFunc(),
	}
}

func (w *ContainerOperationWaiter) RefreshFunc() resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := w.Service.Projects.Zones.Operations.Get(
			w.Project, w.Zone, w.Op.Name).Do()

		if err != nil {
			return nil, "", err
		}

		log.Printf("[DEBUG] Progress of operation %q: %q", w.Op.Name, resp.Status)

		return resp, resp.Status, err
	}
}

func containerOperationWait(config *Config, op *container.Operation, project, zone, activity string, timeoutMinutes, minTimeoutSeconds int) error {
	w := &ContainerOperationWaiter{
		Service: config.clientContainer,
		Op:      op,
		Project: project,
		Zone:    zone,
	}

	state := w.Conf()
	state.Timeout = time.Duration(timeoutMinutes) * time.Minute
	state.MinTimeout = time.Duration(minTimeoutSeconds) * time.Second
	_, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for %s: %s", activity, err)
	}

	return nil
}
