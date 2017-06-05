package librato

import (
	"fmt"
	"net/http"
)

// MetricsService handles communication with the Librato API methods related to
// metrics.
type MetricsService struct {
	client *Client
}

// Metric represents a Librato Metric.
type Metric struct {
	Name        *string           `json:"name"`
	Description *string           `json:"description,omitempty"`
	Type        *string           `json:"type"`
	Period      *uint             `json:"period,omitempty"`
	DisplayName *string           `json:"display_name,omitempty"`
	Composite   *string           `json:"composite,omitempty"`
	Attributes  *MetricAttributes `json:"attributes,omitempty"`
}

// MetricAttributes are named attributes as key:value pairs.
type MetricAttributes struct {
	Color *string `json:"color"`
	// These are interface{} because sometimes the Librato API
	// returns strings, and sometimes it returns integers
	DisplayMax        interface{} `json:"display_max"`
	DisplayMin        interface{} `json:"display_min"`
	DisplayUnitsLong  string      `json:"display_units_long"`
	DisplayUnitsShort string      `json:"display_units_short"`
	DisplayStacked    bool        `json:"display_stacked"`
	CreatedByUA       string      `json:"created_by_ua,omitempty"`
	GapDetection      bool        `json:"gap_detection,omitempty"`
	Aggregate         bool        `json:"aggregate,omitempty"`
}

// ListMetricsOptions are used to coordinate paging of metrics.
type ListMetricsOptions struct {
	*PaginationMeta
	Name string `url:"name,omitempty"`
}

// AdvancePage advances to the specified page in result set, while retaining
// the filtering options.
func (l *ListMetricsOptions) AdvancePage(next *PaginationMeta) ListMetricsOptions {
	return ListMetricsOptions{
		PaginationMeta: next,
		Name:           l.Name,
	}
}

// ListMetricsResponse represents the response of a List call against the metrics service.
type ListMetricsResponse struct {
	ThisPage *PaginationResponseMeta
	NextPage *PaginationMeta
}

// List metrics using the provided options.
//
// Librato API docs: https://www.librato.com/docs/api/#list-a-subset-of-metrics
func (m *MetricsService) List(opts *ListMetricsOptions) ([]Metric, *ListMetricsResponse, error) {
	u, err := urlWithOptions("metrics", opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := m.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var metricsResponse struct {
		Query   PaginationResponseMeta
		Metrics []Metric
	}

	_, err = m.client.Do(req, &metricsResponse)
	if err != nil {
		return nil, nil, err
	}

	return metricsResponse.Metrics,
		&ListMetricsResponse{
			ThisPage: &metricsResponse.Query,
			NextPage: metricsResponse.Query.nextPage(opts.PaginationMeta),
		},
		nil
}

// Get a metric by name
//
// Librato API docs: https://www.librato.com/docs/api/#retrieve-a-metric-by-name
func (m *MetricsService) Get(name string) (*Metric, *http.Response, error) {
	u := fmt.Sprintf("metrics/%s", name)
	req, err := m.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	metric := new(Metric)
	resp, err := m.client.Do(req, metric)
	if err != nil {
		return nil, resp, err
	}

	return metric, resp, err
}

// MeasurementSubmission represents the payload to submit/create a metric.
type MeasurementSubmission struct {
	MeasureTime *uint               `json:"measure_time,omitempty"`
	Source      *string             `json:"source,omitempty"`
	Gauges      []*GaugeMeasurement `json:"gauges,omitempty"`
	Counters    []*Measurement      `json:"counters,omitempty"`
}

// Measurement represents a Librato Measurement.
type Measurement struct {
	Name        string   `json:"name"`
	Value       *float64 `json:"value,omitempty"`
	MeasureTime *uint    `json:"measure_time,omitempty"`
	Source      *string  `json:"source,omitempty"`
}

// GaugeMeasurement represents a Librato measurement gauge.
type GaugeMeasurement struct {
	*Measurement
	Count      *uint    `json:"count,omitempty"`
	Sum        *float64 `json:"sum,omitempty"`
	Max        *float64 `json:"max,omitempty"`
	Min        *float64 `json:"min,omitempty"`
	SumSquares *float64 `json:"sum_squares,omitempty"`
}

// Create metrics
//
// Librato API docs: https://www.librato.com/docs/api/#create-a-measurement
func (m *MetricsService) Create(measurements *MeasurementSubmission) (*http.Response, error) {
	req, err := m.client.NewRequest("POST", "/metrics", measurements)
	if err != nil {
		return nil, err
	}

	return m.client.Do(req, nil)
}

// Update a metric.
//
// Librato API docs: https://www.librato.com/docs/api/#update-a-metric-by-name
func (m *MetricsService) Update(metric *Metric) (*http.Response, error) {
	u := fmt.Sprintf("metrics/%s", *metric.Name)

	req, err := m.client.NewRequest("PUT", u, metric)
	if err != nil {
		return nil, err
	}

	return m.client.Do(req, nil)
}

// Delete a metric.
//
// Librato API docs: https://www.librato.com/docs/api/#delete-a-metric-by-name
func (m *MetricsService) Delete(name string) (*http.Response, error) {
	u := fmt.Sprintf("metrics/%s", name)
	req, err := m.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	return m.client.Do(req, nil)
}
