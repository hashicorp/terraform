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
func waitForASGCapacity(
	d *schema.ResourceData,
	autoscaleId string,
	meta interface{},
	satisfiedFunc capacitySatisfiedFunc) error {
	wait, err := time.ParseDuration(d.Get("wait_for_capacity_timeout").(string))
	if err != nil {
		return err
	}

	if wait == 0 {
		log.Printf("[DEBUG] Capacity timeout set to 0, skipping capacity waiting.")
		return nil
	}

	log.Printf("[DEBUG] Waiting on %s for capacity...", autoscaleId)

	return resource.Retry(wait, func() *resource.RetryError {
		g, err := getAwsAutoscalingGroup(autoscaleId, meta.(*AWSClient).autoscalingconn)
		if err != nil {
			return resource.NonRetryableError(err)
		}
		if g == nil {
			log.Printf("[INFO] Autoscaling Group %q not found", autoscaleId)
			//TODO JRN: Need to figure out what to do here!
			// 	There could be a few reasons for this I suppose, the ASG actually not existing which seems like it should be an error
			//	Or AWS's eventual consistency not returning the ASG information yet, which should probably be a retryable error, not a fast "fail"
			//  I don't see how the calling code actually handles this ID not being sent, it seems like it proceeds on as "normal"
			// d.SetId("")
			return nil
		}

		significantLbs := getSignificantLbsNames(d)
		elbis, err := getELBInstanceStates(significantLbs, meta)
		albis, err := getTargetGroupInstanceStates(g, meta)

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
			for _, lbName := range significantLbs {
				states, lbOk := elbis[lbName]
				if !lbOk {
					inAllLbs = false
					break
				}
				state, ok := states[*i.InstanceId]
				if !ok || !strings.EqualFold(state, "InService") {
					inAllLbs = false
				}
			}
			for _, states := range albis {
				state, ok := states[*i.InstanceId]
				if !ok || !strings.EqualFold(state, "healthy") {
					inAllLbs = false
				}
			}
			if inAllLbs {
				haveELB++
			}
		}

		satisfied, reason := satisfiedFunc(d, haveASG, haveELB)

		log.Printf("[DEBUG] %q Capacity: %d ASG, %d ELB/ALB, satisfied: %t, reason: %q",
			autoscaleId, haveASG, haveELB, satisfied, reason)

		if satisfied {
			return nil
		}

		return resource.RetryableError(
			fmt.Errorf("%q: Waiting up to %s: %s", autoscaleId, wait, reason))
	})
}

// Function to support extracting the LB names that we actually care about.
// For the aws_autoscaling_group that can come in the form of a set via the `load_balancers` attribute
// for the aws_atuoscaling_attachment that can be in the form of a single LB name via the `elb` attribute
func getSignificantLbsNames(d *schema.ResourceData) []string {
	if s, ok := d.GetOk("load_balancers"); ok && s.(*schema.Set).Len() > 0 {
		ret := []string{}
		for _, v := range s.(*schema.Set).List() {
			ret = append(ret, v.(string))
		}
		return ret
	}
	if v, ok := d.GetOk("elb"); ok {
		return []string{v.(string)}
	}
	return []string{}
}

type capacitySatisfiedFunc func(*schema.ResourceData, int, int) (bool, string)

// capacitySatisfiedCreate treats all targets as minimums
func capacitySatisfiedCreate(d *schema.ResourceData, haveASG, haveELB int) (bool, string) {
	minASG := d.Get("min_size").(int)
	if wantASG := d.Get("desired_capacity").(int); wantASG > 0 {
		minASG = wantASG
	}
	if haveASG < minASG {
		return false, fmt.Sprintf(
			"Need at least %d healthy instances in ASG, have %d", minASG, haveASG)
	}
	minELB := d.Get("min_elb_capacity").(int)
	if wantELB := d.Get("wait_for_elb_capacity").(int); wantELB > 0 {
		minELB = wantELB
	}
	if haveELB < minELB {
		return false, fmt.Sprintf(
			"Need at least %d healthy instances in ELB, have %d", minELB, haveELB)
	}
	return true, ""
}

// capacitySatisfiedUpdate only cares about specific targets
func capacitySatisfiedUpdate(d *schema.ResourceData, haveASG, haveELB int) (bool, string) {
	if wantASG := d.Get("desired_capacity").(int); wantASG > 0 {
		if haveASG != wantASG {
			return false, fmt.Sprintf(
				"Need exactly %d healthy instances in ASG, have %d", wantASG, haveASG)
		}
	}
	if wantELB := d.Get("wait_for_elb_capacity").(int); wantELB > 0 {
		if haveELB != wantELB {
			return false, fmt.Sprintf(
				"Need exactly %d healthy instances in ELB, have %d", wantELB, haveELB)
		}
	}
	return true, ""
}

func capacitySatisfiedAttach(d *schema.ResourceData, haveASG, haveELB int) (bool, string) {
	if wantELB := d.Get("wait_for_elb_capacity").(int); wantELB > 0 && haveELB < wantELB {
		return false, fmt.Sprintf(
			"Need exactly %d healthy instances in ELB, have %d", wantELB, haveELB)
	}
	return true, ""
}
