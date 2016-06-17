package compute

import (
	"fmt"
	"log"
	"time"
)

// WaitForDeploy waits for a resource's pending deployment operation to complete.
func (client *Client) WaitForDeploy(resourceType string, id string, timeout time.Duration) (resource Resource, err error) {
	return client.waitForPendingOperation(resourceType, id, "Deploy", ResourceStatusPendingAdd, timeout)
}

// WaitForEdit waits for a resource's pending edit operation to complete.
func (client *Client) WaitForEdit(resourceType string, id string, timeout time.Duration) (resource Resource, err error) {
	return client.WaitForChange(resourceType, id, "Edit", timeout)
}

// WaitForChange waits for a resource's pending change operation to complete.
func (client *Client) WaitForChange(resourceType string, id string, actionDescription string, timeout time.Duration) (resource Resource, err error) {
	return client.waitForPendingOperation(resourceType, id, actionDescription, ResourceStatusPendingChange, timeout)
}

// WaitForDelete waits for a resource's pending deletion to complete.
func (client *Client) WaitForDelete(resourceType string, id string, timeout time.Duration) error {
	_, err := client.waitForPendingOperation(resourceType, id, "Delete", ResourceStatusPendingDelete, timeout)

	return err
}

// waitForPendingOperation waits for a resource's pending operation to complete (i.e. for its status to become ResourceStatusNormal or the resource to disappear if expectedStatus is ResourceStatusPendingDelete).
func (client *Client) waitForPendingOperation(resourceType string, id string, actionDescription string, expectedStatus string, timeout time.Duration) (resource Resource, err error) {
	return client.waitForResourceStatus(resourceType, id, actionDescription, expectedStatus, ResourceStatusNormal, timeout)
}

// waitForResourceStatus polls a resource for its status (which is expected to initially be expectedStatus) until it becomes expectedStatus.
// getResource is a function that, given the resource Id, will retrieve the resource.
// timeout is the length of time before the wait times out.
func (client *Client) waitForResourceStatus(resourceType string, id string, actionDescription string, expectedStatus string, targetStatus string, timeout time.Duration) (resource Resource, err error) {
	waitTimeout := time.NewTimer(timeout)
	defer waitTimeout.Stop()

	pollTicker := time.NewTicker(5 * time.Second)
	defer pollTicker.Stop()

	resourceDescription, err := GetResourceDescription(resourceType)
	if err != nil {
		return nil, err
	}

	for {
		select {
		case <-waitTimeout.C:
			return nil, fmt.Errorf("Timed out after waiting %d seconds for %s of %s '%s' to complete", timeout/time.Second, actionDescription, resourceDescription, id)

		case <-pollTicker.C:
			log.Printf("Polling status for %s '%s'...", resourceDescription, id)
			resource, err := client.GetResource(id, resourceType)
			if err != nil {
				return nil, err
			}
			if err != nil {
				return nil, err
			}

			if resource.IsDeleted() {
				if expectedStatus == ResourceStatusPendingDelete {
					log.Printf("%s '%s' has been successfully deleted.", resourceDescription, id)

					return nil, nil
				}

				return nil, fmt.Errorf("No %s was found with Id '%s'", resourceDescription, id)
			}

			switch resource.GetState() {
			case ResourceStatusNormal:
				log.Printf("%s of %s '%s' has successfully completed.", actionDescription, resourceDescription, id)

				return resource, nil

			case ResourceStatusPendingAdd:
				log.Printf("%s of %s '%s' is still in progress...", actionDescription, resourceDescription, id)

				continue
			case ResourceStatusPendingChange:
				log.Printf("%s of %s '%s' is still in progress...", actionDescription, resourceDescription, id)

				continue
			case ResourceStatusPendingDelete:
				log.Printf("%s of %s '%s' is still in progress...", actionDescription, resourceDescription, id)

				continue
			default:
				log.Printf("Unexpected status for %s '%s' ('%s').", resourceDescription, id, resource.GetState())

				return nil, fmt.Errorf("%s failed for %s '%s' ('%s'): encountered unexpected state '%s'", actionDescription, resourceDescription, id, resource.GetName(), resource.GetState())
			}
		}
	}
}
