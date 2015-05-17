package google

import (
	"bytes"
	"fmt"

	"google.golang.org/api/autoscaler/v1beta2"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/replicapool/v1beta2"
	"github.com/hashicorp/terraform/helper/resource"
)

// OperationWaitType is an enum specifying what type of operation
// we're waiting on.
type OperationWaitType byte

const (
	OperationWaitInvalid OperationWaitType = iota
	OperationWaitGlobal
	OperationWaitRegion
	OperationWaitZone
)

type OperationWaiter struct {
	Service *compute.Service
	Op      *compute.Operation
	Project string
	Region  string
	Zone    string
	Type    OperationWaitType
}

func (w *OperationWaiter) RefreshFunc() resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		var op *compute.Operation
		var err error

		switch w.Type {
		case OperationWaitGlobal:
			op, err = w.Service.GlobalOperations.Get(
				w.Project, w.Op.Name).Do()
		case OperationWaitRegion:
			op, err = w.Service.RegionOperations.Get(
				w.Project, w.Region, w.Op.Name).Do()
		case OperationWaitZone:
			op, err = w.Service.ZoneOperations.Get(
				w.Project, w.Zone, w.Op.Name).Do()
		default:
			return nil, "bad-type", fmt.Errorf(
				"Invalid wait type: %#v", w.Type)
		}

		if err != nil {
			return nil, "", err
		}

		return op, op.Status, nil
	}
}

func (w *OperationWaiter) Conf() *resource.StateChangeConf {
	return &resource.StateChangeConf{
		Pending: []string{"PENDING", "RUNNING"},
		Target:  "DONE",
		Refresh: w.RefreshFunc(),
	}
}

// OperationError wraps compute.OperationError and implements the
// error interface so it can be returned.
type OperationError compute.OperationError

func (e OperationError) Error() string {
	var buf bytes.Buffer

	for _, err := range e.Errors {
		buf.WriteString(err.Message + "\n")
	}

	return buf.String()
}

// Replicapool Operations
type ReplicaPoolOperationWaiter struct {
	Service *replicapool.Service
	Op      *replicapool.Operation
	Project string
	Region  string
	Zone    string
}

func (w *ReplicaPoolOperationWaiter) RefreshFunc() resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		var op *replicapool.Operation
		var err error

		op, err = w.Service.ZoneOperations.Get(
			w.Project, w.Zone, w.Op.Name).Do()

		if err != nil {
			return nil, "", err
		}

		return op, op.Status, nil
	}
}

func (w *ReplicaPoolOperationWaiter) Conf() *resource.StateChangeConf {
	return &resource.StateChangeConf{
		Pending: []string{"PENDING", "RUNNING"},
		Target:  "DONE",
		Refresh: w.RefreshFunc(),
	}
}

// ReplicaPoolOperationError wraps replicapool.OperationError and implements the
// error interface so it can be returned.
type ReplicaPoolOperationError replicapool.OperationError

func (e ReplicaPoolOperationError) Error() string {
	var buf bytes.Buffer

	for _, err := range e.Errors {
		buf.WriteString(err.Message + "\n")
	}

	return buf.String()
}

// Autoscaler Operations
type AutoscalerOperationWaiter struct {
	Service *autoscaler.Service
	Op      *autoscaler.Operation
	Project string
	Zone    string
}

func (w *AutoscalerOperationWaiter) RefreshFunc() resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		var op *autoscaler.Operation
		var err error

		op, err = w.Service.ZoneOperations.Get(
			w.Project, w.Zone, w.Op.Name).Do()

		if err != nil {
			return nil, "", err
		}

		return op, op.Status, nil
	}
}

func (w *AutoscalerOperationWaiter) Conf() *resource.StateChangeConf {
	return &resource.StateChangeConf{
		Pending: []string{"PENDING", "RUNNING"},
		Target:  "DONE",
		Refresh: w.RefreshFunc(),
	}
}

// AutoscalerOperationError wraps autoscaler.OperationError and implements the
// error interface so it can be returned.
type AutoscalerOperationError autoscaler.OperationError

func (e AutoscalerOperationError) Error() string {
	var buf bytes.Buffer

	for _, err := range e.Errors {
		buf.WriteString(err.Message + "\n")
	}

	return buf.String()
}
