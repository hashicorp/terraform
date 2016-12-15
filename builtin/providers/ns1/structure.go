package ns1

import (
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"gopkg.in/ns1/ns1-go.v2/rest/model/dns"
	"gopkg.in/ns1/ns1-go.v2/rest/model/filter"
	"gopkg.in/ns1/ns1-go.v2/rest/model/monitor"
)

// Helper methods copied from AWS provider

type setMap map[string]interface{}

func (s setMap) Set(key string, value interface{}) {
	s[key] = value
}

func (s setMap) MapList() []map[string]interface{} {
	return []map[string]interface{}{s.Map()}
}

func (s setMap) Map() map[string]interface{} {
	return map[string]interface{}(s)
}

func expandNS1Answers(d *schema.ResourceData) ([]*dns.Answer, error) {
	var err error

	list := d.Get("answer").([]interface{})
	result := make([]*dns.Answer, 0, len(list))

	for _, a := range list {
		ans := a.(map[string]interface{})

		vs := ans["rdata"].([]interface{})
		rdata := make([]string, len(vs))
		for i, raw := range vs {
			rdata[i] = raw.(string)
		}

		answer := &dns.Answer{Rdata: rdata}

		// k := fmt.Sprintf("answer.%d.meta", i)
		// answer.Meta = expandNS1Metadata(d, k)

		if ans["region"] != "" {
			answer.RegionName = ans["region"].(string)
		}

		result = append(result, answer)
	}

	return result, err
}

func expandNS1Filters(d *schema.ResourceData) ([]*filter.Filter, error) {
	list := d.Get("filter").([]interface{})
	result := make([]*filter.Filter, 0, len(list))

	for _, f := range list {
		fil := f.(map[string]interface{})

		filter := &filter.Filter{
			Type:     fil["type"].(string),
			Disabled: fil["disabled"].(bool),
			Config:   make(map[string]interface{}),
		}

		vs := fil["config"].(map[string]interface{})
		for k, v := range vs {
			switch filter.Type {
			case "select_first_n", "ipv4_prefix_shuffle":
				val, err := strconv.Atoi(v.(string))
				if err != nil {
					return nil, err
				}
				filter.Config[k] = val
			case "sticky", "sticky_region", "weighted_sticky",
				"geofence_country", "geofence_regional",
				"netfence_asn", "netfence_prefix":
				filter.Config[k] = strconv.FormatBool(v.(bool))
			default:
				filter.Config[k] = v.(string)

			}
		}

		result = append(result, filter)
	}

	return result, nil
}

func expandNS1JobRules(d *schema.ResourceData) []*monitor.Rule {
	list := d.Get("rules").([]interface{})
	result := make([]*monitor.Rule, 0, len(list))

	for _, r := range list {
		rMap := r.(map[string]interface{})

		rule := &monitor.Rule{
			Key:        rMap["key"].(string),
			Value:      rMap["value"].(string),
			Comparison: rMap["comparison"].(string),
		}

		result = append(result, rule)
	}

	return result
}

func flattenNS1Answers(list []*dns.Answer) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))

	for _, a := range list {
		r := make(map[string]interface{})
		r["rdata"] = a.Rdata

		if a.RegionName != "" {
			r["region"] = a.RegionName
		}

		// log.Printf("[DEBUG] flattenAnswers: %#v \n", a.Meta)
		// r["meta"] = flattenNS1Metadata(a.Meta)

		result = append(result, r)
	}

	return result
}

func flattenNS1Filters(list []*filter.Filter) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))

	for _, f := range list {
		r := make(map[string]interface{})
		r["type"] = f.Type
		r["disabled"] = f.Disabled

		if f.Config != nil {
			r["config"] = f.Config
		}

		result = append(result, r)
	}

	return result
}

func flattenNS1JobRules(list []*monitor.Rule) []map[string]interface{} {
	result := make([]map[string]interface{}, len(list))

	for i, rule := range list {
		r := make(map[string]interface{})
		r["key"] = rule.Key
		r["value"] = rule.Value
		r["comparison"] = rule.Comparison

		result[i] = r
	}

	return result
}
