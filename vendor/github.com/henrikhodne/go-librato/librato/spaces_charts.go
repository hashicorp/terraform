package librato

import (
	"fmt"
	"net/http"
)

// SpaceChart represents a chart in a Librato Space.
type SpaceChart struct {
	ID           *uint              `json:"id,omitempty"`
	Name         *string            `json:"name,omitempty"`
	Type         *string            `json:"type,omitempty"`
	Min          *float64           `json:"min,omitempty"`
	Max          *float64           `json:"max,omitempty"`
	Label        *string            `json:"label,omitempty"`
	RelatedSpace *uint              `json:"related_space,omitempty"`
	Streams      []SpaceChartStream `json:"streams,omitempty"`
}

// SpaceChartStream represents a single stream in a chart in a Librato Space.
type SpaceChartStream struct {
	Metric            *string  `json:"metric,omitempty"`
	Source            *string  `json:"source,omitempty"`
	Composite         *string  `json:"composite,omitempty"`
	GroupFunction     *string  `json:"group_function,omitempty"`
	SummaryFunction   *string  `json:"summary_function,omitempty"`
	Color             *string  `json:"color,omitempty"`
	Name              *string  `json:"name,omitempty"`
	UnitsShort        *string  `json:"units_short,omitempty"`
	UnitsLong         *string  `json:"units_long,omitempty"`
	Min               *float64 `json:"min,omitempty"`
	Max               *float64 `json:"max,omitempty"`
	TransformFunction *string  `json:"transform_function,omitempty"`
	Period            *int64   `json:"period,omitempty"`
}

// CreateChart creates a chart in a given Librato Space.
//
// Librato API docs: http://dev.librato.com/v1/post/spaces/:id/charts
func (s *SpacesService) CreateChart(spaceID uint, chart *SpaceChart) (*SpaceChart, *http.Response, error) {
	u := fmt.Sprintf("spaces/%d/charts", spaceID)
	req, err := s.client.NewRequest("POST", u, chart)
	if err != nil {
		return nil, nil, err
	}

	c := new(SpaceChart)
	resp, err := s.client.Do(req, c)
	if err != nil {
		return nil, resp, err
	}

	return c, resp, err
}

// ListCharts lists all charts in a given Librato Space.
//
// Librato API docs: http://dev.librato.com/v1/get/spaces/:id/charts
func (s *SpacesService) ListCharts(spaceID uint) ([]SpaceChart, *http.Response, error) {
	u := fmt.Sprintf("spaces/%d/charts", spaceID)
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	charts := new([]SpaceChart)
	resp, err := s.client.Do(req, charts)
	if err != nil {
		return nil, resp, err
	}

	return *charts, resp, err
}

// GetChart gets a chart with a given ID in a space with a given ID.
//
// Librato API docs: http://dev.librato.com/v1/get/spaces/:id/charts
func (s *SpacesService) GetChart(spaceID, chartID uint) (*SpaceChart, *http.Response, error) {
	u := fmt.Sprintf("spaces/%d/charts/%d", spaceID, chartID)
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	c := new(SpaceChart)
	resp, err := s.client.Do(req, c)
	if err != nil {
		return nil, resp, err
	}

	return c, resp, err
}

// EditChart edits a chart.
//
// Librato API docs: http://dev.librato.com/v1/put/spaces/:id/charts/:id
func (s *SpacesService) EditChart(spaceID, chartID uint, chart *SpaceChart) (*http.Response, error) {
	u := fmt.Sprintf("spaces/%d/charts/%d", spaceID, chartID)
	req, err := s.client.NewRequest("PUT", u, chart)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// DeleteChart deletes a chart.
//
// Librato API docs: http://dev.librato.com/v1/delete/spaces/:id/charts/:id
func (s *SpacesService) DeleteChart(spaceID, chartID uint) (*http.Response, error) {
	u := fmt.Sprintf("spaces/%d/charts/%d", spaceID, chartID)
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}
