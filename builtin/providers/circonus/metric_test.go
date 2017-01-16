package circonus

import "testing"

func Test_MetricChecksum(t *testing.T) {
	var unit string = "qty"
	m := _InterfaceMap{
		string(_MetricActiveAttr): true,
		string(_MetricNameAttr):   "asdf",
		string(_MetricTagsAttr):   tagsToState(apiToTags([]string{"foo", "bar"})),
		string(_MetricTypeAttr):   "json",
		string(_MetricUnitAttr):   &unit,
	}
	ar := _NewMapReader(nil, m)

	csum := _MetricChecksum(ar)
	if csum != 4250221491 {
		t.Fatalf("Checksum mismatch")
	}
}
