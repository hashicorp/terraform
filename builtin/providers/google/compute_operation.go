package google

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"google.golang.org/api/compute/v1"
)

// OperationWaitType is an enum specifying what type of operation
// we're waiting on.
type ComputeOperationWaitType byte

const (
	ComputeOperationWaitInvalid ComputeOperationWaitType = iota
	ComputeOperationWaitGlobal
	ComputeOperationWaitRegion
	ComputeOperationWaitZone
)

type ComputeOperationWaiter struct {
	Service *compute.Service
	Op      *compute.Operation
	Project string
	Region  string
	Type    ComputeOperationWaitType
	Zone    string
}

func (w *ComputeOperationWaiter) RefreshFunc() resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		var op *compute.Operation
		var err error

		switch w.Type {
		case ComputeOperationWaitGlobal:
			op, err = w.Service.GlobalOperations.Get(
				w.Project, w.Op.Name).Do()
		case ComputeOperationWaitRegion:
			op, err = w.Service.RegionOperations.Get(
				w.Project, w.Region, w.Op.Name).Do()
		case ComputeOperationWaitZone:
			op, err = w.Service.ZoneOperations.Get(
				w.Project, w.Zone, w.Op.Name).Do()
		default:
			return nil, "bad-type", fmt.Errorf(
				"Invalid wait type: %#v", w.Type)
		}

		if err != nil {
			return nil, "", err
		}

		log.Printf("[DEBUG] Got %q when asking for operation %q", op.Status, w.Op.Name)

		return op, op.Status, nil
	}
}

func (w *ComputeOperationWaiter) Conf() *resource.StateChangeConf {
	return &resource.StateChangeConf{
		Pending: []string{"PENDING", "RUNNING"},
		Target:  []string{"DONE"},
		Refresh: w.RefreshFunc(),
	}
}

// ComputeOperationError wraps compute.OperationError and implements the
// error interface so it can be returned.
type ComputeOperationError compute.OperationError

func (e ComputeOperationError) Error() string {
	var buf bytes.Buffer

	for _, err := range e.Errors {
		buf.WriteString(err.Message + "\n")
	}

	return buf.String()
}

func computeOperationWaitGlobal(config *Config, op *compute.Operation, project string, activity string) error {
	return computeOperationWaitGlobalTime(config, op, project, activity, 4)
}

func computeOperationWaitGlobalTime(config *Config, op *compute.Operation, project string, activity string, timeoutMin int) error {
	w := &ComputeOperationWaiter{
		Service: config.clientCompute,
		Op:      op,
		Project: project,
		Type:    ComputeOperationWaitGlobal,
	}

	state := w.Conf()
	state.Delay = 10 * time.Second
	state.Timeout = time.Duration(timeoutMin) * time.Minute
	state.MinTimeout = 2 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for %s: %s", activity, err)
	}

	op = opRaw.(*compute.Operation)
	if op.Error != nil {
		return ComputeOperationError(*op.Error)
	}

	return nil
}

func computeOperationWaitRegion(config *Config, op *compute.Operation, project string, region, activity string) error {
	w := &ComputeOperationWaiter{
		Service: config.clientCompute,
		Op:      op,
		Project: project,
		Type:    ComputeOperationWaitRegion,
		Region:  region,
	}

	state := w.Conf()
	state.Delay = 10 * time.Second
	state.Timeout = 4 * time.Minute
	state.MinTimeout = 2 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for %s: %s", activity, err)
	}

	op = opRaw.(*compute.Operation)
	if op.Error != nil {
		return ComputeOperationError(*op.Error)
	}

	return nil
}

func computeOperationWaitZone(config *Config, op *compute.Operation, project string, zone, activity string) error {
	return computeOperationWaitZoneTime(config, op, project, zone, 4, activity)
}

func computeOperationWaitZoneTime(config *Config, op *compute.Operation, project string, zone string, minutes int, activity string) error {
	w := &ComputeOperationWaiter{
		Service: config.clientCompute,
		Op:      op,
		Project: project,
		Zone:    zone,
		Type:    ComputeOperationWaitZone,
	}
	state := w.Conf()
	state.Delay = 10 * time.Second
	state.Timeout = time.Duration(minutes) * time.Minute
	state.MinTimeout = 2 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for %s: %s", activity, err)
	}
	op = opRaw.(*compute.Operation)
	if op.Error != nil {
		// Return the error
		return ComputeOperationError(*op.Error)
	}
	return nil
}
