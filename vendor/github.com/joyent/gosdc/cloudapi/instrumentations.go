package cloudapi

import (
	"net/http"

	"github.com/joyent/gocommon/client"
	"github.com/joyent/gocommon/errors"
)

// Analytics represents the available analytics
type Analytics struct {
	Modules         map[string]interface{} // Namespace to organize metrics
	Fields          map[string]interface{} // Fields represent metadata by which data points can be filtered or decomposed
	Types           map[string]interface{} // Types are used with both metrics and fields for two purposes: to hint to clients at how to best label values, and to distinguish between numeric and discrete quantities.
	Metrics         map[string]interface{} // Metrics describe quantities which can be measured by the system
	Transformations map[string]interface{} // Transformations are post-processing functions that can be applied to data when it's retrieved.
}

// Instrumentation specify which metric to collect, how frequently to aggregate data (e.g., every second, every hour, etc.)
// how much data to keep (e.g., 10 minutes' worth, 6 months' worth, etc.) and other configuration options
type Instrumentation struct {
	Module          string   `json:"module"`
	Stat            string   `json:"stat"`
	Predicate       string   `json:"predicate"`
	Decomposition   []string `json:"decomposition"`
	ValueDimension  int      `json:"value-dimenstion"`
	ValueArity      string   `json:"value-arity"`
	RetentionTime   int      `json:"retention-time"`
	Granularity     int      `json:"granularitiy"`
	IdleMax         int      `json:"idle-max"`
	Transformations []string `json:"transformations"`
	PersistData     bool     `json:"persist-data"`
	Crtime          int      `json:"crtime"`
	ValueScope      string   `json:"value-scope"`
	Id              string   `json:"id"`
	Uris            []Uri    `json:"uris"`
}

// Uri represents a Universal Resource Identifier
type Uri struct {
	Uri  string // Resource identifier
	Name string // URI name
}

// InstrumentationValue represents the data associated to an instrumentation for a point in time
type InstrumentationValue struct {
	Value           interface{}
	Transformations map[string]interface{}
	StartTime       int
	Duration        int
}

// HeatmapOpts represent the option that can be specified
// when retrieving an instrumentation.'s heatmap
type HeatmapOpts struct {
	Height       int      `json:"height"`        // Height of the image in pixels
	Width        int      `json:"width"`         // Width of the image in pixels
	Ymin         int      `json:"ymin"`          // Y-Axis value for the bottom of the image (default: 0)
	Ymax         int      `json:"ymax"`          // Y-Axis value for the top of the image (default: auto)
	Nbuckets     int      `json:"nbuckets"`      // Number of buckets in the vertical dimension
	Selected     []string `json:"selected"`      // Array of field values to highlight, isolate or exclude
	Isolate      bool     `json:"isolate"`       // If true, only draw selected values
	Exclude      bool     `json:"exclude"`       // If true, don't draw selected values at all
	Hues         []string `json:"hues"`          // Array of colors for highlighting selected field values
	DecomposeAll bool     `json:"decompose_all"` // Highlight all field values
	X            int      `json:"x"`
	Y            int      `json:"y"`
}

// Heatmap represents an instrumentation's heatmap
type Heatmap struct {
	BucketTime int                    `json:"bucket_time"` // Time corresponding to the bucket (Unix seconds)
	BucketYmin int                    `json:"bucket_ymin"` // Minimum y-axis value for the bucket
	BucketYmax int                    `json:"bucket_ymax"` // Maximum y-axis value for the bucket
	Present    map[string]interface{} `json:"present"`     // If the instrumentation defines a discrete decomposition, this property's value is an object whose keys are values of that field and whose values are the number of data points in that bucket for that key
	Total      int                    `json:"total"`       // The total number of data points in the bucket
}

// CreateInstrumentationOpts represent the option that can be specified
// when creating a new instrumentation.
type CreateInstrumentationOpts struct {
	Clone         int    `json:"clone"`     // An existing instrumentation ID to be cloned
	Module        string `json:"module"`    // Analytics module
	Stat          string `json:"stat"`      // Analytics stat
	Predicate     string `json:"predicate"` // Instrumentation predicate, must be JSON string
	Decomposition string `json:"decomposition"`
	Granularity   int    `json:"granularity"`    // Number of seconds between data points (default is 1)
	RetentionTime int    `json:"retention-time"` // How long to keep this instrumentation data for
	PersistData   bool   `json:"persist-data"`   // Whether or not to store this for historical analysis
	IdleMax       int    `json:"idle-max"`       // Number of seconds after which if the instrumentation or its data has not been accessed via the API the service may delete the instrumentation and its data
}

// DescribeAnalytics retrieves the "schema" for instrumentations that can be created.
// See API docs: http://apidocs.joyent.com/cloudapi/#DescribeAnalytics
func (c *Client) DescribeAnalytics() (*Analytics, error) {
	var resp Analytics
	req := request{
		method: client.GET,
		url:    apiAnalytics,
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get analytics")
	}
	return &resp, nil
}

// ListInstrumentations retrieves all currently created instrumentations.
// See API docs: http://apidocs.joyent.com/cloudapi/#ListInstrumentations
func (c *Client) ListInstrumentations() ([]Instrumentation, error) {
	var resp []Instrumentation
	req := request{
		method: client.GET,
		url:    makeURL(apiAnalytics, apiInstrumentations),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get instrumentations")
	}
	return resp, nil
}

// GetInstrumentation retrieves the configuration for the specified instrumentation.
// See API docs: http://apidocs.joyent.com/cloudapi/#GetInstrumentation
func (c *Client) GetInstrumentation(instrumentationID string) (*Instrumentation, error) {
	var resp Instrumentation
	req := request{
		method: client.GET,
		url:    makeURL(apiAnalytics, apiInstrumentations, instrumentationID),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get instrumentation with id %s", instrumentationID)
	}
	return &resp, nil
}

// GetInstrumentationValue retrieves the data associated to an instrumentation
// for a point in time.
// See API docs: http://apidocs.joyent.com/cloudapi/#GetInstrumentationValue
func (c *Client) GetInstrumentationValue(instrumentationID string) (*InstrumentationValue, error) {
	var resp InstrumentationValue
	req := request{
		method: client.GET,
		url:    makeURL(apiAnalytics, apiInstrumentations, instrumentationID, apiInstrumentationsValue, apiInstrumentationsRaw),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get value for instrumentation with id %s", instrumentationID)
	}
	return &resp, nil
}

// GetInstrumentationHeatmap retrieves the specified instrumentation's heatmap.
// See API docs: http://apidocs.joyent.com/cloudapi/#GetInstrumentationHeatmap
func (c *Client) GetInstrumentationHeatmap(instrumentationID string) (*Heatmap, error) {
	var resp Heatmap
	req := request{
		method: client.GET,
		url:    makeURL(apiAnalytics, apiInstrumentations, instrumentationID, apiInstrumentationsValue, apiInstrumentationsHeatmap, apiInstrumentationsImage),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get heatmap image for instrumentation with id %s", instrumentationID)
	}
	return &resp, nil
}

// GetInstrumentationHeatmapDetails allows you to retrieve the bucket details
// for a heatmap.
// See API docs: http://apidocs.joyent.com/cloudapi/#GetInstrumentationHeatmapDetails
func (c *Client) GetInstrumentationHeatmapDetails(instrumentationID string) (*Heatmap, error) {
	var resp Heatmap
	req := request{
		method: client.GET,
		url:    makeURL(apiAnalytics, apiInstrumentations, instrumentationID, apiInstrumentationsValue, apiInstrumentationsHeatmap, apiInstrumentationsDetails),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get heatmap details for instrumentation with id %s", instrumentationID)
	}
	return &resp, nil
}

// CreateInstrumentation Creates an instrumentation. You can clone an existing
// instrumentation by passing in the parameter clone, which should be a numeric id
// of an existing instrumentation.
// See API docs: http://apidocs.joyent.com/cloudapi/#CreateInstrumentation
func (c *Client) CreateInstrumentation(opts CreateInstrumentationOpts) (*Instrumentation, error) {
	var resp Instrumentation
	req := request{
		method:         client.POST,
		url:            makeURL(apiAnalytics, apiInstrumentations),
		reqValue:       opts,
		resp:           &resp,
		expectedStatus: http.StatusCreated,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to create instrumentation")
	}
	return &resp, nil
}

// DeleteInstrumentation destroys an instrumentation.
// See API docs: http://apidocs.joyent.com/cloudapi/#DeleteInstrumentation
func (c *Client) DeleteInstrumentation(instrumentationID string) error {
	req := request{
		method:         client.DELETE,
		url:            makeURL(apiAnalytics, apiInstrumentations, instrumentationID),
		expectedStatus: http.StatusNoContent,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to delete instrumentation with id %s", instrumentationID)
	}
	return nil
}
