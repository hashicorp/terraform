package circonus

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/terraform/helper/schema"
)

// The _Check type is the backing store of the `circonus_check` resource.

type _Check struct {
	api.CheckBundle
}

type _CheckType string

const (
	_CheckTypeJSON _CheckType = "json"
)

func _NewCheck() _Check {
	return _Check{
		CheckBundle: *api.NewCheckBundle(),
	}
}

func (c *_Check) Create(ctxt *providerContext) error {
	cb, err := ctxt.client.CreateCheckBundle(&c.CheckBundle)
	if err != nil {
		return err
	}

	c.CID = cb.CID

	return nil
}

func (c *_Check) Update(ctxt *providerContext) error {
	panic("not implemented")
}

func (c *_Check) ParseSchema(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)

	if name, ok := d.GetOk(checkNameAttr); ok {
		c.DisplayName = name.(string)
	}

	if status, ok := schemaGetBoolOK(d, _CheckActiveAttr); ok {
		statusString := checkStatusActive
		if !status {
			statusString = checkStatusDisabled
		}

		c.Status = statusString
	}

	if collectorsList, ok := schemaGetSetAsListOk(d, _CheckCollectorAttr); ok {
		c.Brokers = collectorsList.CollectKey(_CheckCollectorIDAttr)
	}

	if streamList, ok := schemaGetSetAsListOk(d, _CheckStreamAttr); ok {
		c.Metrics = make([]api.CheckBundleMetric, 0, len(streamList))

		for _, metricListRaw := range streamList {
			metricAttrs := _NewInterfaceMap(metricListRaw)

			m := _NewMetric()
			m.Name = metricAttrs.GetString(_MetricNameAttr)
			m.SetTags(metricAttrs.GetTags(ctxt, _MetricTagsAttr, _Tag{}))
			m.Type = metricAttrs.GetString(_MetricTypeAttr)
			m.Units = metricAttrs.GetStringPtr(_MetricUnitAttr)

			c.Metrics = append(c.Metrics, m.CheckBundleMetric)
		}
	}

	if l, ok := schemaGetSetAsListOk(d, _CheckJSONAttr); ok {
		if err := c.parseJSONCheck(l); err != nil {
			return err
		}
	}

	// var checkType CheckType
	// if v, ok := d.GetOk(checkTypeAttr); ok {
	// 	c.Type = v.(string)
	// 	checkType = CheckType(c.Type)
	// }

	// if bundleConfigRaw, ok := d.GetOk(checkConfigAttr); ok {
	// 	bundleConfigList := bundleConfigRaw.([]interface{})
	// 	checkConfig := makeCheckBundleConfig(checkType)
	// 	switch configLen := len(bundleConfigList); {
	// 	case configLen > 1:
	// 		return fmt.Errorf("config doesn't match schema: count %d", configLen)
	// 	case configLen == 1:
	// 		configMap := bundleConfigList[0].(map[string]interface{})
	// 		if v, ok := configMap[checkConfigHTTPHeadersAttr]; ok {
	// 			headerMap := v.(map[string]interface{})
	// 			for hK, hV := range headerMap {
	// 				hKey := config.HeaderPrefix + config.Key(hK)
	// 				checkConfig[hKey] = hV.(string)
	// 			}
	// 		}

	// 		if v, ok := configMap[checkConfigHTTPVersionAttr]; ok {
	// 			checkConfig[checkConfigHTTPVersionAttr] = v.(string)
	// 		}

	// 		if v, ok := configMap[checkConfigPortAttr]; ok {
	// 			checkConfig[checkConfigPortAttr] = v.(string)
	// 		}

	// 		if v, ok := configMap[checkConfigReadLimitAttr]; ok {
	// 			checkConfig[checkConfigReadLimitAttr] = fmt.Sprintf("%d", v.(int))
	// 		}

	// 		if v, ok := configMap[checkConfigRedirectsAttr]; ok {
	// 			checkConfig[checkConfigRedirectsAttr] = fmt.Sprintf("%d", v.(int))
	// 		}

	// 		if v, ok := configMap[checkConfigURLAttr]; ok {
	// 			checkConfig[checkConfigURLAttr] = v.(string)
	// 		}
	// 	case configLen == 0:
	// 		// Default config values, if any
	// 	}

	// 	c.Config = checkConfig
	// }

	if v, ok := d.GetOk(checkMetricLimitAttr); ok {
		c.MetricLimit = v.(int)
	}

	// if bundleMetricsRaw, ok := d.GetOk(checkMetricAttr); ok {
	// 	bundleMetricsList := bundleMetricsRaw.([]interface{})
	// 	c.Metrics = make([]api.CheckBundleMetric, 0, len(bundleMetricsList))

	// 	if len(bundleMetricsList) == 0 {
	// 		return fmt.Errorf("at least one metric must be specified per check bundle")
	// 	}

	// 	for _, metricRaw := range bundleMetricsList {
	// 		metricMap := metricRaw.(map[string]interface{})

	// 		var metricName string
	// 		if v, ok := metricMap[checkMetricNameAttr]; ok {
	// 			metricName = v.(string)
	// 		}

	// 		var metricStatus string = metricStatusActive
	// 		if v, ok := metricMap[checkMetricActiveAttr]; ok {
	// 			if !v.(bool) {
	// 				metricStatus = metricStatusAvailable
	// 			}
	// 		}

	// 		var metricTags _Tags
	// 		if tagsRaw, ok := metricMap[checkMetricTagsAttr]; ok {
	// 			tags := tagsRaw.(map[string]interface{})

	// 			metricTags = make(_Tags, len(tags))
	// 			for k, v := range tags {
	// 				metricTags[_TagCategory(k)] = _TagValue(v.(string))
	// 			}
	// 		}
	// 		metricTags = injectTag(ctxt, metricTags, ctxt.defaultTag)

	// 		var metricType string
	// 		if v, ok := metricMap[checkMetricTypeAttr]; ok {
	// 			metricType = v.(string)
	// 		}

	// 		var metricUnits string
	// 		if v, ok := metricMap[checkMetricUnitsAttr]; ok {
	// 			metricUnits = v.(string)
	// 		}

	// 		c.Metrics = append(c.Metrics, api.CheckBundleMetric{
	// 			Name:   metricName,
	// 			Status: metricStatus,
	// 			Tags:   tagsToAPI(metricTags),
	// 			Type:   metricType,
	// 			Units:  metricUnits,
	// 		})
	// 	}
	// }

	c.Notes = schemaGetStringPtr(d, checkNotesAttr)

	if v, ok := d.GetOk(checkPeriodAttr); ok {
		d, _ := time.ParseDuration(v.(string))
		c.Period = uint(d.Seconds())
	}

	var checkTags _Tags
	if tagsRaw, ok := d.GetOk(checkTagsAttr); ok {
		tags := tagsRaw.(map[string]interface{})

		checkTags = make(_Tags, len(tags))
		for k, v := range tags {
			checkTags[_TagCategory(k)] = _TagValue(v.(string))
		}
	}
	checkTags = injectTag(ctxt, checkTags, ctxt.defaultTag)
	c.Tags = tagsToAPI(checkTags)

	if v, ok := d.GetOk(checkTargetAttr); ok {
		c.Target = v.(string)
	}

	if v, ok := d.GetOk(checkTimeoutAttr); ok {
		c.Timeout = v.(float32)
	}

	if err := c.Validate(); err != nil {
		return err
	}

	return nil
}

func (c *_Check) Validate() error {
	if c.Timeout > float32(c.Period) {
		return fmt.Errorf("Timeout (%f) can not exceed period (%d)", c.Timeout, c.Period)
	}

	return nil
}

func (c *_Check) parseJSONCheck(l _InterfaceList) error {
	c.Type = string(_CheckTypeJSON)

	for _, mapRaw := range l {
		jsonAttrs := mapRaw.(map[string]interface{})

		if mapRaw, ok := jsonAttrs[string(_CheckJSONHTTPHeadersAttr)]; ok {
			headerMap := mapRaw.(map[string]interface{})

			for k, v := range headerMap {
				h := config.HeaderPrefix + config.Key(k)
				c.Config[h] = v.(string)
			}
		}

		if v, ok := jsonAttrs[string(_CheckJSONHTTPVersionAttr)]; ok {
			c.Config[config.HTTPVersion] = v.(string)
		}

		if v, ok := jsonAttrs[string(_CheckJSONURLAttr)]; ok {
			c.Config[config.URL] = v.(string)

			u, _ := url.Parse(v.(string))
			host := strings.SplitN(u.Host, ":", 2)
			c.Target = host[0]
		}
	}

	return nil
}
