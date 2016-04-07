package aws

import "testing"

import "github.com/hashicorp/terraform/helper/schema"

func TestCapacitySatisfied(t *testing.T) {
	cases := map[string]struct {
		Data            map[string]interface{}
		HaveASG         int
		HaveELB         int
		ExpectSatisfied bool
		ExpectReason    string
	}{
		"min_size, have less": {
			Data: map[string]interface{}{
				"min_size": 5,
			},
			HaveASG:         2,
			ExpectSatisfied: false,
			ExpectReason:    "Need at least 5 healthy instances in ASG, have 2",
		},
		"min_size, have less, launch suspended": {
			Data: map[string]interface{}{
				"min_size":            5,
				"suspended_processes": schema.NewSet(schema.HashString, []interface{}{"Launch"}),
			},
			HaveASG:         2,
			ExpectSatisfied: true,
		},
		"min_size, got it": {
			Data: map[string]interface{}{
				"min_size": 5,
			},
			HaveASG:         5,
			ExpectSatisfied: true,
		},
		"min_size, have more": {
			Data: map[string]interface{}{
				"min_size": 5,
			},
			HaveASG:         10,
			ExpectSatisfied: true,
		},
		"desired_capacity, have less": {
			Data: map[string]interface{}{
				"desired_capacity": 5,
			},
			HaveASG:         2,
			ExpectSatisfied: false,
			ExpectReason:    "Need exactly 5 healthy instances in ASG, have 2",
		},
		"desired_capacity, have less, launch suspended": {
			Data: map[string]interface{}{
				"desired_capacity":    5,
				"suspended_processes": schema.NewSet(schema.HashString, []interface{}{"Launch"}),
			},
			HaveASG:         2,
			ExpectSatisfied: true,
		},
		"desired_capacity, have less, terminate suspended": {
			Data: map[string]interface{}{
				"desired_capacity":    5,
				"suspended_processes": schema.NewSet(schema.HashString, []interface{}{"Terminate"}),
			},
			HaveASG:         2,
			ExpectSatisfied: false,
			ExpectReason:    "Need exactly 5 healthy instances in ASG, have 2",
		},
		"desired_capacity overrides min_size": {
			Data: map[string]interface{}{
				"min_size":         2,
				"desired_capacity": 5,
			},
			HaveASG:         2,
			ExpectSatisfied: false,
			ExpectReason:    "Need exactly 5 healthy instances in ASG, have 2",
		},
		"desired_capacity, got it": {
			Data: map[string]interface{}{
				"desired_capacity": 5,
			},
			HaveASG:         5,
			ExpectSatisfied: true,
		},
		"max_size, have less": {
			Data: map[string]interface{}{
				"max_size": 5,
			},
			HaveASG:         2,
			ExpectSatisfied: true,
		},
		"max_size, have more": {
			Data: map[string]interface{}{
				"max_size": 5,
			},
			HaveASG:         10,
			ExpectSatisfied: false,
			ExpectReason:    "Need at most 5 healthy instances in ASG, have 10",
		},
		"max_size, have more, terminate suspended": {
			Data: map[string]interface{}{
				"max_size":            5,
				"suspended_processes": schema.NewSet(schema.HashString, []interface{}{"Terminate"}),
			},
			HaveASG:         10,
			ExpectSatisfied: true,
		},
		"max_size, got it": {
			Data: map[string]interface{}{
				"max_size": 5,
			},
			HaveASG:         5,
			ExpectSatisfied: true,
		},
		"min_size, max_size, between": {
			Data: map[string]interface{}{
				"min_size": 1,
				"max_size": 5,
			},
			HaveASG:         2,
			ExpectSatisfied: true,
		},
		"min_elb_capacity, have less": {
			Data: map[string]interface{}{
				"min_elb_capacity": 5,
			},
			HaveELB:         2,
			ExpectSatisfied: false,
			ExpectReason:    "Need at least 5 healthy instances in ELB, have 2",
		},
		"min_elb_capacity, have less, launch suspended": {
			Data: map[string]interface{}{
				"min_elb_capacity":    5,
				"suspended_processes": schema.NewSet(schema.HashString, []interface{}{"Launch"}),
			},
			HaveELB:         2,
			ExpectSatisfied: true,
		},
		"min_elb_capacity, got it": {
			Data: map[string]interface{}{
				"min_elb_capacity": 5,
			},
			HaveELB:         5,
			ExpectSatisfied: true,
		},
		"min_elb_capacity, have more": {
			Data: map[string]interface{}{
				"min_elb_capacity": 5,
			},
			HaveELB:         10,
			ExpectSatisfied: true,
		},
		"wait_for_elb_capacity, have less": {
			Data: map[string]interface{}{
				"wait_for_elb_capacity": 5,
			},
			HaveELB:         2,
			ExpectSatisfied: false,
			ExpectReason:    "Need exactly 5 healthy instances in ELB, have 2",
		},
		"wait_for_elb_capacity, have less, launch suspended": {
			Data: map[string]interface{}{
				"wait_for_elb_capacity": 5,
				"suspended_processes":   schema.NewSet(schema.HashString, []interface{}{"Launch"}),
			},
			HaveELB:         2,
			ExpectSatisfied: true,
		},
		"wait_for_elb_capacity, have more, terminate suspended": {
			Data: map[string]interface{}{
				"wait_for_elb_capacity": 5,
				"suspended_processes":   schema.NewSet(schema.HashString, []interface{}{"Terminate"}),
			},
			HaveELB:         10,
			ExpectSatisfied: true,
		},
		"wait_for_elb_capacity, got it": {
			Data: map[string]interface{}{
				"wait_for_elb_capacity": 5,
			},
			HaveELB:         5,
			ExpectSatisfied: true,
		},
		"wait_for_elb_capacity overrides min_elb_capacity": {
			Data: map[string]interface{}{
				"min_elb_capacity":      2,
				"wait_for_elb_capacity": 5,
			},
			HaveELB:         2,
			ExpectSatisfied: false,
			ExpectReason:    "Need exactly 5 healthy instances in ELB, have 2",
		},
	}

	r := resourceAwsAutoscalingGroup()
	for tn, tc := range cases {
		d := r.TestResourceData()
		for k, v := range tc.Data {
			if err := d.Set(k, v); err != nil {
				t.Fatalf("err: %s", err)
			}
		}
		gotSatisfied, gotReason := checkCapacitySatisfied(d, tc.HaveASG, tc.HaveELB)

		if gotSatisfied != tc.ExpectSatisfied {
			t.Fatalf("%s: expected satisfied: %t, got: %t (reason: %s)",
				tn, tc.ExpectSatisfied, gotSatisfied, gotReason)
		}

		if gotReason != tc.ExpectReason {
			t.Fatalf("%s: expected reason: %s, got: %s",
				tn, tc.ExpectReason, gotReason)
		}
	}
}
