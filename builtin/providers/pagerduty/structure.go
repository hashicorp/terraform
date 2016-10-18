package pagerduty

import pagerduty "github.com/PagerDuty/go-pagerduty"

// Expands an array of escalation rules into []pagerduty.EscalationRules
func expandRules(list []interface{}) []pagerduty.EscalationRule {
	result := make([]pagerduty.EscalationRule, 0, len(list))

	for _, r := range list {
		rule := r.(map[string]interface{})

		escalationRule := &pagerduty.EscalationRule{
			Delay: uint(rule["escalation_delay_in_minutes"].(int)),
		}

		for _, t := range rule["target"].([]interface{}) {
			target := t.(map[string]interface{})
			escalationRule.Targets = append(
				escalationRule.Targets,
				pagerduty.APIObject{
					ID:   target["id"].(string),
					Type: target["type"].(string),
				},
			)
		}

		result = append(result, *escalationRule)

	}

	return result
}

// Flattens an array of []pagerduty.EscalationRule into a map[string]interface{}
func flattenRules(list []pagerduty.EscalationRule) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))

	for _, i := range list {
		r := make(map[string]interface{})
		r["id"] = i.ID
		r["escalation_delay_in_minutes"] = i.Delay

		if len(i.Targets) > 0 {
			targets := make([]map[string]interface{}, 0, len(i.Targets))
			for _, t := range i.Targets {
				targets = append(targets, map[string]interface{}{
					"id":   t.ID,
					"type": t.Type,
				})
			}
			r["target"] = targets
		}

		result = append(result, r)
	}

	return result
}

// Expands an array of schedules into []pagerduty.Schedule
func expandLayers(list []interface{}) []pagerduty.ScheduleLayer {
	result := make([]pagerduty.ScheduleLayer, 0, len(list))

	for _, l := range list {
		layer := l.(map[string]interface{})

		scheduleLayer := &pagerduty.ScheduleLayer{
			Name:                      layer["name"].(string),
			Start:                     layer["start"].(string),
			End:                       layer["end"].(string),
			RotationVirtualStart:      layer["rotation_virtual_start"].(string),
			RotationTurnLengthSeconds: uint(layer["rotation_turn_length_seconds"].(int)),
		}

		if layer["id"] != "" {
			scheduleLayer.ID = layer["id"].(string)
		}

		for _, u := range layer["users"].([]interface{}) {
			scheduleLayer.Users = append(
				scheduleLayer.Users,
				pagerduty.UserReference{
					User: pagerduty.APIObject{
						ID:   u.(string),
						Type: "user_reference",
					},
				},
			)
		}

		for _, r := range layer["restriction"].([]interface{}) {
			restriction := r.(map[string]interface{})
			scheduleLayer.Restrictions = append(
				scheduleLayer.Restrictions,
				pagerduty.Restriction{
					Type:            restriction["type"].(string),
					StartTimeOfDay:  restriction["start_time_of_day"].(string),
					DurationSeconds: uint(restriction["duration_seconds"].(int)),
				},
			)
		}

		result = append(result, *scheduleLayer)
	}

	return result
}

// Flattens an array of []pagerduty.ScheduleLayer into a map[string]interface{}
func flattenLayers(list []pagerduty.ScheduleLayer) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))

	for _, i := range list {
		r := make(map[string]interface{})
		r["id"] = i.ID
		r["name"] = i.Name
		r["end"] = i.End
		r["start"] = i.Start
		r["rotation_virtual_start"] = i.RotationVirtualStart
		r["rotation_turn_length_seconds"] = i.RotationTurnLengthSeconds

		if len(i.Users) > 0 {
			users := make([]string, 0, len(i.Users))
			for _, u := range i.Users {
				users = append(users, u.User.ID)
			}
			r["users"] = users
		}

		if len(i.Restrictions) > 0 {
			restrictions := make([]map[string]interface{}, 0, len(i.Restrictions))
			for _, r := range i.Restrictions {
				restrictions = append(restrictions, map[string]interface{}{
					"duration_seconds":  r.DurationSeconds,
					"start_time_of_day": r.StartTimeOfDay,
					"type":              r.Type,
				})
			}
			r["restriction"] = restrictions
		}

		result = append(result, r)
	}

	// Reverse the final result and return it
	resultReversed := make([]map[string]interface{}, 0, len(result))

	for i := len(result) - 1; i >= 0; i-- {
		resultReversed = append(resultReversed, result[i])
	}

	return resultReversed
}

// Flattens an array of []pagerduty.User into a map[string]interface{}
func flattenOnCalls(list []pagerduty.OnCall) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))

	for _, i := range list {
		r := make(map[string]interface{})
		r["escalation_level"] = i.EscalationLevel
		r["start"] = i.Start
		r["end"] = i.End

		user := make(map[string]interface{}, 1)
		user["id"] = i.User.ID
		user["type"] = i.User.Type
		user["name"] = i.User.Summary
		user["summary"] = i.User.Summary

		schedule := make(map[string]interface{}, 1)
		schedule["id"] = i.Schedule.ID
		schedule["type"] = i.Schedule.Type
		schedule["summary"] = i.Schedule.Summary

		policy := make(map[string]interface{}, 1)
		policy["id"] = i.EscalationPolicy.ID
		policy["type"] = i.EscalationPolicy.Type
		policy["summary"] = i.EscalationPolicy.Summary

		r["user"] = user
		r["schedule"] = schedule
		r["escalation_policy"] = policy

		result = append(result, r)
	}

	return result
}

// Takes the result of flatmap.Expand for an array of strings
// and returns a []string
func expandStringList(configured []interface{}) []string {
	vs := make([]string, 0, len(configured))
	for _, v := range configured {
		vs = append(vs, string(v.(string)))
	}
	return vs
}
