package noaa

import (
	"sort"

	"github.com/cloudfoundry/sonde-go/events"
)

// SortContainerMetrics sorts a slice of containerMetrics by InstanceIndex.
//
// The input slice is sorted; the return value is simply a pointer to the same slice.
func SortContainerMetrics(messages []*events.ContainerMetric) []*events.ContainerMetric {
	sort.Sort(containerMetricSlice(messages))
	return messages
}

type containerMetricSlice []*events.ContainerMetric

func (lms containerMetricSlice) Len() int {
	return len(lms)
}

func (lms containerMetricSlice) Less(i, j int) bool {
	return (*(lms[i])).GetInstanceIndex() < (*(lms[j])).GetInstanceIndex()
}

func (lms containerMetricSlice) Swap(i, j int) {
	lms[i], lms[j] = lms[j], lms[i]
}
