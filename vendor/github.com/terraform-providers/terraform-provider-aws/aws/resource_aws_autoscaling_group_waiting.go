package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
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

	log.Printf("[DEBUG] Waiting on %s for capacity...", d.Id())

	err = resource.Retry(wait, func() *resource.RetryError {
		g, err := getAwsAutoscalingGroup(d.Id(), meta.(*AWSClient).autoscalingconn)
		if err != nil {
			return resource.NonRetryableError(err)
		}
		if g == nil {
			log.Printf("[WARN] Autoscaling Group (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		elbis, err := getELBInstanceStates(g, meta)
		if err != nil {
			return resource.NonRetryableError(err)
		}
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
			for _, states := range elbis {
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
			d.Id(), haveASG, haveELB, satisfied, reason)

		if satisfied {
			return nil
		}

		return resource.RetryableError(
			fmt.Errorf("%q: Waiting up to %s: %s", d.Id(), wait, reason))
	})

	if err == nil {
		return nil
	}

	recentStatus := ""

	conn := meta.(*AWSClient).autoscalingconn
	resp, aErr := conn.DescribeScalingActivities(&autoscaling.DescribeScalingActivitiesInput{
		AutoScalingGroupName: aws.String(d.Id()),
		MaxRecords:           aws.Int64(1),
	})
	if aErr == nil {
		if len(resp.Activities) > 0 {
			recentStatus = (resp.Activities[0].String())
		} else {
			recentStatus = "(0 activities found)"
		}
	} else {
		recentStatus = fmt.Sprintf("(Failed to describe scaling activities: %s)", aErr)
	}

	return fmt.Errorf("%s. Most recent activity: %s", err, recentStatus)
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
