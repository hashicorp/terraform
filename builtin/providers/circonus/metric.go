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

func (m *circonusMetric) ParseConfig(id string, ar attrReader) error {
	m.ID = metricID(id)
	m.Name = ar.GetString(metricNameAttr)
	m.Status = metricActiveToAPIStatus(ar.GetBool(metricActiveAttr))
	m.Tags = tagsToAPI(ar.GetTags(metricTagsAttr))
	m.Type = ar.GetString(metricTypeAttr)
	m.Units = ar.GetStringPtr(metricUnitAttr)
	if m.Units != nil && *m.Units == "" {
		m.Units = nil
	}

	return nil
}

func (m *circonusMetric) SaveState(d *schema.ResourceData) error {
	stateSet(d, metricActiveAttr, metricAPIStatusToBool(m.Status))
	stateSet(d, metricNameAttr, m.Name)
	stateSet(d, metricTagsAttr, tagsToState(apiToTags(m.Tags)))
	stateSet(d, metricTypeAttr, m.Type)
	stateSet(d, metricUnitAttr, indirect(m.Units))

	d.SetId(string(m.ID))

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
		panic(fmt.Sprintf("PROVIDER BUG: metric status %q unsupported", s))
	}
}

func metricActiveToAPIStatus(active bool) string {
	switch active {
	case true:
		return metricStatusActive
	case false:
		return metricStatusAvailable
	}

	panic("suppress Go error message")
}

func newMetricID() (string, error) {
	id, err := uuid.GenerateUUID()
	if err != nil {
		return "", errwrap.Wrapf("metric ID creation failed: {{err}}", err)
	}

	return id, nil
}

func metricChecksum(ar attrReader) int {
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	// Order writes to the buffer using lexically sorted list for easy visual
	// reconciliation with other lists.
	fmt.Fprint(b, ar.GetBool(metricActiveAttr))
	fmt.Fprint(b, ar.GetString(metricNameAttr))
	tags := ar.GetTags(metricTagsAttr)
	for _, tag := range tags {
		fmt.Fprint(b, tag)
	}
	fmt.Fprint(b, ar.GetString(metricTypeAttr))
	if p := ar.GetStringPtr(metricUnitAttr); p != nil && *p != "" {
		fmt.Fprint(b, indirect(p))
	}

	s := b.String()
	return hashcode.String(s)
}
