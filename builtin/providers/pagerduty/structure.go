package pagerduty

import pagerduty "github.com/PagerDuty/go-pagerduty"

// Expands an array of escalation rules into []pagerduty.EscalationRules
func expandEscalationRules(list []interface{}) []pagerduty.EscalationRule {
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
func flattenEscalationRules(list []pagerduty.EscalationRule) []map[string]interface{} {
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
func expandScheduleLayers(list []interface{}) []pagerduty.ScheduleLayer {
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
					StartDayOfWeek:  uint(restriction["start_day_of_week"].(int)),
					DurationSeconds: uint(restriction["duration_seconds"].(int)),
				},
			)
		}

		result = append(result, *scheduleLayer)
	}

	return result
}

// Expands an array of teams into []pagerduty.APIReference
func expandTeams(list []interface{}) []pagerduty.APIReference {
	result := make([]pagerduty.APIReference, 0, len(list))

	for _, l := range list {
		team := &pagerduty.APIReference{
			ID:   l.(string),
			Type: "team_reference",
		}

		result = append(result, *team)
	}

	return result
}

// Flattens an array of []pagerduty.ScheduleLayer into a map[string]interface{}
func flattenScheduleLayers(list []pagerduty.ScheduleLayer) []map[string]interface{} {
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
				restriction := map[string]interface{}{
					"duration_seconds":  r.DurationSeconds,
					"start_time_of_day": r.StartTimeOfDay,
					"type":              r.Type,
				}

				if r.StartDayOfWeek > 0 {
					restriction["start_day_of_week"] = r.StartDayOfWeek
				}

				restrictions = append(restrictions, restriction)
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

// Takes the result of flatmap.Expand for an array of strings
// and returns a []string
func expandStringList(configured []interface{}) []string {
	vs := make([]string, 0, len(configured))
	for _, v := range configured {
		vs = append(vs, string(v.(string)))
	}
	return vs
}

// Expands attribute slice to incident urgency rule, returns it and true if successful
func expandIncidentUrgencyRule(incidentUrgencyList interface{}) (*pagerduty.IncidentUrgencyRule, bool) {
	i := incidentUrgencyList.([]interface{})

	i, ok := incidentUrgencyList.([]interface{})
	if !ok {
		return nil, false
	}

	m, ok := i[0].(map[string]interface{})
	if !ok || len(m) == 0 {
		return nil, false
	}

	iur := pagerduty.IncidentUrgencyRule{}
	if val, ok := m["type"]; ok {
		iur.Type = val.(string)
	}
	if val, ok := m["urgency"]; ok {
		iur.Urgency = val.(string)
	}
	if val, ok := m["during_support_hours"]; ok {
		iur.DuringSupportHours = expandIncidentUrgencyType(val)
	}
	if val, ok := m["outside_support_hours"]; ok {
		iur.OutsideSupportHours = expandIncidentUrgencyType(val)
	}

	return &iur, true
}

// Expands attribute to inline model
func expandActionInlineModel(inlineModelVal interface{}) *pagerduty.InlineModel {
	inlineModel := pagerduty.InlineModel{}

	if slice, ok := inlineModelVal.([]interface{}); ok && len(slice) == 1 {
		m := slice[0].(map[string]interface{})

		if val, ok := m["type"]; ok {
			inlineModel.Type = val.(string)
		}
		if val, ok := m["name"]; ok {
			inlineModel.Name = val.(string)
		}
	}

	return &inlineModel
}

// Expands attribute into incident urgency type
func expandIncidentUrgencyType(attribute interface{}) *pagerduty.IncidentUrgencyType {
	ict := pagerduty.IncidentUrgencyType{}

	slice := attribute.([]interface{})
	if len(slice) != 1 {
		return &ict
	}

	m := slice[0].(map[string]interface{})

	if val, ok := m["type"]; ok {
		ict.Type = val.(string)
	}
	if val, ok := m["urgency"]; ok {
		ict.Urgency = val.(string)
	}

	return &ict
}

// Returns service's incident urgency rule as slice of length one and bool indicating success
func flattenIncidentUrgencyRule(service *pagerduty.Service) ([]interface{}, bool) {
	if service.IncidentUrgencyRule.Type == "" && service.IncidentUrgencyRule.Urgency == "" {
		return nil, false
	}

	m := map[string]interface{}{
		"type":    service.IncidentUrgencyRule.Type,
		"urgency": service.IncidentUrgencyRule.Urgency,
	}

	if dsh := service.IncidentUrgencyRule.DuringSupportHours; dsh != nil {
		m["during_support_hours"] = flattenIncidentUrgencyType(dsh)
	}
	if osh := service.IncidentUrgencyRule.OutsideSupportHours; osh != nil {
		m["outside_support_hours"] = flattenIncidentUrgencyType(osh)
	}

	return []interface{}{m}, true
}

func flattenIncidentUrgencyType(iut *pagerduty.IncidentUrgencyType) []interface{} {
	incidentUrgencyType := map[string]interface{}{
		"type":    iut.Type,
		"urgency": iut.Urgency,
	}
	return []interface{}{incidentUrgencyType}
}

// Expands attribute to support hours
func expandSupportHours(attribute interface{}) (sh *pagerduty.SupportHours) {
	if slice, ok := attribute.([]interface{}); ok && len(slice) >= 1 {
		m := slice[0].(map[string]interface{})
		sh = &pagerduty.SupportHours{}

		if val, ok := m["type"]; ok {
			sh.Type = val.(string)
		}
		if val, ok := m["time_zone"]; ok {
			sh.Timezone = val.(string)
		}
		if val, ok := m["start_time"]; ok {
			sh.StartTime = val.(string)
		}
		if val, ok := m["end_time"]; ok {
			sh.EndTime = val.(string)
		}
		if val, ok := m["days_of_week"]; ok {
			daysOfWeekInt := val.([]interface{})
			var daysOfWeek []uint

			for _, i := range daysOfWeekInt {
				daysOfWeek = append(daysOfWeek, uint(i.(int)))
			}

			sh.DaysOfWeek = daysOfWeek
		}
	}

	return
}

// Returns service's support hours as slice of length one
func flattenSupportHours(service *pagerduty.Service) []interface{} {
	if service.SupportHours == nil {
		return nil
	}

	m := map[string]interface{}{}

	if s := service.SupportHours; s != nil {
		m["type"] = s.Type
		m["time_zone"] = s.Timezone
		m["start_time"] = s.StartTime
		m["end_time"] = s.EndTime
		m["days_of_week"] = s.DaysOfWeek
	}

	return []interface{}{m}
}

// Expands attribute to scheduled action
func expandScheduledActions(input interface{}) (scheduledActions []pagerduty.ScheduledAction) {
	inputs := input.([]interface{})

	for _, i := range inputs {
		m := i.(map[string]interface{})
		sa := pagerduty.ScheduledAction{}

		if val, ok := m["type"]; ok {
			sa.Type = val.(string)
		}
		if val, ok := m["to_urgency"]; ok {
			sa.ToUrgency = val.(string)
		}
		if val, ok := m["at"]; ok {
			sa.At = *expandActionInlineModel(val)
		}

		scheduledActions = append(scheduledActions, sa)
	}

	return scheduledActions
}

// Returns service's scheduled actions
func flattenScheduledActions(service *pagerduty.Service) []interface{} {
	scheduledActions := []interface{}{}

	if sas := service.ScheduledActions; sas != nil {
		for _, sa := range sas {
			m := map[string]interface{}{}
			m["to_urgency"] = sa.ToUrgency
			m["type"] = sa.Type
			if at, ok := scheduledActionsAt(sa.At); ok {
				m["at"] = at
			}
			scheduledActions = append(scheduledActions, m)
		}
	}

	return scheduledActions
}

// Returns service's scheduled action's at attribute as slice of length one
func scheduledActionsAt(inlineModel pagerduty.InlineModel) ([]interface{}, bool) {
	if inlineModel.Type == "" || inlineModel.Name == "" {
		return nil, false
	}

	m := map[string]interface{}{"type": inlineModel.Type, "name": inlineModel.Name}
	return []interface{}{m}, true
}
