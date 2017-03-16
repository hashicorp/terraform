package circonus

// The circonusMetric type is the backing store of the `circonus_metric` resource.

import (
	"bytes"
	"fmt"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/errwrap"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

type circonusMetric struct {
	ID metricID
	api.CheckBundleMetric
}

func newMetric() circonusMetric {
	return circonusMetric{}
}

func (m *circonusMetric) Create(d *schema.ResourceData) error {
	return m.SaveState(d)
}

func (m *circonusMetric) ParseConfig(id string, d *schema.ResourceData) error {
	m.ID = metricID(id)

	if v, found := d.GetOk(metricNameAttr); found {
		m.Name = v.(string)
	}

	if v, found := d.GetOk(metricActiveAttr); found {
		m.Status = metricActiveToAPIStatus(v.(bool))
	}

	if v, found := d.GetOk(metricTagsAttr); found {
		m.Tags = derefStringList(flattenSet(v.(*schema.Set)))
	}

	if v, found := d.GetOk(metricTypeAttr); found {
		m.Type = v.(string)
	}

	if v, found := d.GetOk(metricUnitAttr); found {
		s := v.(string)
		m.Units = &s
	}

	if m.Units != nil && *m.Units == "" {
		m.Units = nil
	}

	return nil
}

func (m *circonusMetric) ParseConfigMap(id string, attrMap map[string]interface{}) error {
	m.ID = metricID(id)

	if v, found := attrMap[metricNameAttr]; found {
		m.Name = v.(string)
	}

	if v, found := attrMap[metricActiveAttr]; found {
		m.Status = metricActiveToAPIStatus(v.(bool))
	}

	if v, found := attrMap[metricTagsAttr]; found {
		m.Tags = derefStringList(flattenSet(v.(*schema.Set)))
	}

	if v, found := attrMap[metricTypeAttr]; found {
		m.Type = v.(string)
	}

	if v, found := attrMap[metricUnitAttr]; found {
		s := v.(string)
		m.Units = &s
	}

	if m.Units != nil && *m.Units == "" {
		m.Units = nil
	}

	return nil
}

func (m *circonusMetric) SaveState(d *schema.ResourceData) error {
	d.SetId(string(m.ID))

	d.Set(metricActiveAttr, metricAPIStatusToBool(m.Status))
	d.Set(metricNameAttr, m.Name)
	d.Set(metricTagsAttr, tagsToState(apiToTags(m.Tags)))
	d.Set(metricTypeAttr, m.Type)
	d.Set(metricUnitAttr, indirect(m.Units))

	return nil
}

func (m *circonusMetric) Update(d *schema.ResourceData) error {
	// NOTE: there are no "updates" to be made against an API server, so we just
	// pass through a call to SaveState.  Keep this method around for API
	// symmetry.
	return m.SaveState(d)
}

func metricAPIStatusToBool(s string) bool {
	switch s {
	case metricStatusActive:
		return true
	case metricStatusAvailable:
		return false
	default:
		// log.Printf("PROVIDER BUG: metric status %q unsupported", s)
		return false
	}
}

func metricActiveToAPIStatus(active bool) string {
	if active {
		return metricStatusActive
	}

	return metricStatusAvailable
}

func newMetricID() (string, error) {
	id, err := uuid.GenerateUUID()
	if err != nil {
		return "", errwrap.Wrapf("metric ID creation failed: {{err}}", err)
	}

	return id, nil
}

func metricChecksum(m interfaceMap) int {
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	// Order writes to the buffer using lexically sorted list for easy visual
	// reconciliation with other lists.
	if v, found := m[metricActiveAttr]; found {
		fmt.Fprintf(b, "%t", v.(bool))
	}

	if v, found := m[metricNameAttr]; found {
		fmt.Fprint(b, v.(string))
	}

	if v, found := m[metricTagsAttr]; found {
		tags := derefStringList(flattenSet(v.(*schema.Set)))
		for _, tag := range tags {
			fmt.Fprint(b, tag)
		}
	}

	if v, found := m[metricTypeAttr]; found {
		fmt.Fprint(b, v.(string))
	}

	if v, found := m[metricUnitAttr]; found {
		if v != nil {
			var s string
			switch v.(type) {
			case string:
				s = v.(string)
			case *string:
				s = *v.(*string)
			}

			if s != "" {
				fmt.Fprint(b, s)
			}
		}
	}

	s := b.String()
	return hashcode.String(s)
}
