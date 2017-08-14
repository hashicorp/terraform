package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas_v2/listeners"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas_v2/loadbalancers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas_v2/monitors"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas_v2/pools"
)

func waitForLBV2Listener(networkingClient *gophercloud.ServiceClient, id string, target string, pending []string, timeout time.Duration) error {
	log.Printf("[DEBUG] Waiting for listener %s to become %s.", id, target)

	stateConf := &resource.StateChangeConf{
		Target:     []string{target},
		Pending:    pending,
		Refresh:    resourceLBV2ListenerRefreshFunc(networkingClient, id),
		Timeout:    timeout,
		Delay:      5 * time.Second,
		MinTimeout: 1 * time.Second,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		if _, ok := err.(gophercloud.ErrDefault404); ok {
			switch target {
			case "DELETED":
				return nil
			default:
				return fmt.Errorf("Error: listener %s not found: %s", id, err)
			}
		}
		return fmt.Errorf("Error waiting for listener %s to become %s: %s", id, target, err)
	}

	return nil
}

func resourceLBV2ListenerRefreshFunc(networkingClient *gophercloud.ServiceClient, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		listener, err := listeners.Get(networkingClient, id).Extract()
		if err != nil {
			return nil, "", err
		}

		// The listener resource has no Status attribute, so a successful Get is the best we can do
		return listener, "ACTIVE", nil
	}
}

func waitForLBV2LoadBalancer(networkingClient *gophercloud.ServiceClient, id string, target string, pending []string, timeout time.Duration) error {
	log.Printf("[DEBUG] Waiting for loadbalancer %s to become %s.", id, target)

	stateConf := &resource.StateChangeConf{
		Target:     []string{target},
		Pending:    pending,
		Refresh:    resourceLBV2LoadBalancerRefreshFunc(networkingClient, id),
		Timeout:    timeout,
		Delay:      5 * time.Second,
		MinTimeout: 1 * time.Second,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		if _, ok := err.(gophercloud.ErrDefault404); ok {
			switch target {
			case "DELETED":
				return nil
			default:
				return fmt.Errorf("Error: loadbalancer %s not found: %s", id, err)
			}
		}
		return fmt.Errorf("Error waiting for loadbalancer %s to become %s: %s", id, target, err)
	}

	return nil
}

func resourceLBV2LoadBalancerRefreshFunc(networkingClient *gophercloud.ServiceClient, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		lb, err := loadbalancers.Get(networkingClient, id).Extract()
		if err != nil {
			return nil, "", err
		}

		return lb, lb.ProvisioningStatus, nil
	}
}

func waitForLBV2Member(networkingClient *gophercloud.ServiceClient, poolID, memberID string, target string, pending []string, timeout time.Duration) error {
	log.Printf("[DEBUG] Waiting for member %s to become %s.", memberID, target)

	stateConf := &resource.StateChangeConf{
		Target:     []string{target},
		Pending:    pending,
		Refresh:    resourceLBV2MemberRefreshFunc(networkingClient, poolID, memberID),
		Timeout:    timeout,
		Delay:      5 * time.Second,
		MinTimeout: 1 * time.Second,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		if _, ok := err.(gophercloud.ErrDefault404); ok {
			switch target {
			case "DELETED":
				return nil
			default:
				return fmt.Errorf("Error: member %s not found: %s", memberID, err)
			}
		}
		return fmt.Errorf("Error waiting for member %s to become %s: %s", memberID, target, err)
	}

	return nil
}

func resourceLBV2MemberRefreshFunc(networkingClient *gophercloud.ServiceClient, poolID, memberID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		member, err := pools.GetMember(networkingClient, poolID, memberID).Extract()
		if err != nil {
			return nil, "", err
		}

		// The member resource has no Status attribute, so a successful Get is the best we can do
		return member, "ACTIVE", nil
	}
}

func waitForLBV2Monitor(networkingClient *gophercloud.ServiceClient, id string, target string, pending []string, timeout time.Duration) error {
	log.Printf("[DEBUG] Waiting for monitor %s to become %s.", id, target)

	stateConf := &resource.StateChangeConf{
		Target:     []string{target},
		Pending:    pending,
		Refresh:    resourceLBV2MonitorRefreshFunc(networkingClient, id),
		Timeout:    timeout,
		Delay:      5 * time.Second,
		MinTimeout: 1 * time.Second,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		if _, ok := err.(gophercloud.ErrDefault404); ok {
			switch target {
			case "DELETED":
				return nil
			default:
				return fmt.Errorf("Error: monitor %s not found: %s", id, err)
			}
		}
		return fmt.Errorf("Error waiting for monitor %s to become %s: %s", id, target, err)
	}

	return nil
}

func resourceLBV2MonitorRefreshFunc(networkingClient *gophercloud.ServiceClient, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		monitor, err := monitors.Get(networkingClient, id).Extract()
		if err != nil {
			return nil, "", err
		}

		// The monitor resource has no Status attribute, so a successful Get is the best we can do
		return monitor, "ACTIVE", nil
	}
}

func waitForLBV2Pool(networkingClient *gophercloud.ServiceClient, id string, target string, pending []string, timeout time.Duration) error {
	log.Printf("[DEBUG] Waiting for pool %s to become %s.", id, target)

	stateConf := &resource.StateChangeConf{
		Target:     []string{target},
		Pending:    pending,
		Refresh:    resourceLBV2PoolRefreshFunc(networkingClient, id),
		Timeout:    timeout,
		Delay:      5 * time.Second,
		MinTimeout: 1 * time.Second,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		if _, ok := err.(gophercloud.ErrDefault404); ok {
			switch target {
			case "DELETED":
				return nil
			default:
				return fmt.Errorf("Error: pool %s not found: %s", id, err)
			}
		}
		return fmt.Errorf("Error waiting for pool %s to become %s: %s", id, target, err)
	}

	return nil
}

func resourceLBV2PoolRefreshFunc(networkingClient *gophercloud.ServiceClient, poolID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		pool, err := pools.Get(networkingClient, poolID).Extract()
		if err != nil {
			return nil, "", err
		}

		// The pool resource has no Status attribute, so a successful Get is the best we can do
		return pool, "ACTIVE", nil
	}
}

func waitForLBV2viaPool(networkingClient *gophercloud.ServiceClient, id string, target string, timeout time.Duration) error {
	pool, err := pools.Get(networkingClient, id).Extract()
	if err != nil {
		return err
	}

	if pool.Loadbalancers != nil {
		// we know each pool has an LB
		lbID := pool.Loadbalancers[0].ID
		return waitForLBV2LoadBalancer(networkingClient, lbID, target, nil, timeout)
	}

	// got a pool but no LB - this is wrong
	return fmt.Errorf("No Load Balancer on pool %s", id)
}
