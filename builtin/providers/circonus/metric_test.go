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
	ar := newMapReader(nil, m)

	csum := metricChecksum(ar)
	if csum != 4250221491 {
		t.Fatalf("Checksum mismatch")
	}
}
