package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

// waitForASGCapacityTimeout gathers the current numbers of healthy instances
// in the ASG and its attached ELBs and yields these numbers to a
// capacitySatifiedFunction. Loops for up to wait_for_capacity_timeout until
// the capacitySatisfiedFunc returns true.
//
// See "Waiting for Capacity" in docs for more discussion of the feature.
func waitForASGCapacity(d *schema.ResourceData, meta interface{}) error {
	wait, err := time.ParseDuration(d.Get("wait_for_capacity_timeout").(string))
	if err != nil {
		return err
	}

	if wait == 0 {
		log.Printf("[DEBUG] Capacity timeout set to 0, skipping capacity waiting.")
		return nil
	}

	log.Printf("[DEBUG] Waiting on %s for capacity...", d.Id())

	return resource.Retry(wait, func() *resource.RetryError {
		g, err := getAwsAutoscalingGroup(d.Id(), meta.(*AWSClient).autoscalingconn)
		if err != nil {
			return resource.NonRetryableError(err)
		}
		if g == nil {
			log.Printf("[INFO] Autoscaling Group %q not found", d.Id())
			d.SetId("")
			return nil
		}
		lbis, err := getLBInstanceStates(g, meta)
		if err != nil {
			return resource.NonRetryableError(err)
		}

		haveASG := 0
		haveELB := 0

		for _, i := range g.Instances {
			if i.HealthStatus == nil || i.InstanceId == nil || i.LifecycleState == nil {
				continue
			}

			if !strings.EqualFold(*i.HealthStatus, "Healthy") {
				continue
			}

			if !strings.EqualFold(*i.LifecycleState, "InService") {
				continue
			}

			haveASG++

			inAllLbs := true
			for _, states := range lbis {
				state, ok := states[*i.InstanceId]
				if !ok || !strings.EqualFold(state, "InService") {
					inAllLbs = false
				}
			}
			if inAllLbs {
				haveELB++
			}
		}

		satisfied, reason := checkCapacitySatisfied(d, haveASG, haveELB)

		log.Printf("[DEBUG] %q Capacity: %d ASG, %d ELB, satisfied: %t, reason: %q",
			d.Id(), haveASG, haveELB, satisfied, reason)

		if satisfied {
			return nil
		}

		return resource.RetryableError(
			fmt.Errorf("%q: Waiting up to %s: %s", d.Id(), wait, reason))
	})
}

// checkCapacitySatisfied determines if required ASG and ELB targets are met.
func checkCapacitySatisfied(d *schema.ResourceData, haveASG, haveELB int) (bool, string) {
	if desiredASG, ok := d.GetOk("desired_capacity"); ok && desiredASG.(int) != haveASG {
		return false, fmt.Sprintf(
			"Need exactly %d healthy instances in ASG, have %d", desiredASG.(int), haveASG)
	}

	if minASG, ok := d.GetOk("min_size"); ok && minASG.(int) > haveASG {
		return false, fmt.Sprintf(
			"Need at least %d healthy instances in ASG, have %d", minASG.(int), haveASG)
	}

	if maxASG, ok := d.GetOk("max_size"); ok && maxASG.(int) < haveASG {
		return false, fmt.Sprintf(
			"Need at most %d healthy instances in ASG, have %d", maxASG.(int), haveASG)
	}

	if desiredELB, ok := d.GetOk("wait_for_elb_capacity"); ok && desiredELB.(int) != haveELB {
		return false, fmt.Sprintf(
			"Need exactly %d healthy instances in ELB, have %d", desiredELB.(int), haveELB)
	}

	if minELB, ok := d.GetOk("min_elb_capacity"); ok && minELB.(int) > haveELB {
		return false, fmt.Sprintf(
			"Need at least %d healthy instances in ELB, have %d", minELB.(int), haveELB)
	}

	return true, ""
}
