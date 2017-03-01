package circonus

import "testing"

func Test_MetricChecksum(t *testing.T) {
	unit := "qty"
	m := interfaceMap{
		string(metricActiveAttr): true,
		string(metricNameAttr):   "asdf",
		string(metricTagsAttr):   tagsToState(apiToTags([]string{"foo", "bar"})),
		string(metricTypeAttr):   "json",
		string(metricUnitAttr):   &unit,
	}

	csum := metricChecksum(m)
	if csum != 4250221491 {
		t.Fatalf("Checksum mismatch")
	}
}
