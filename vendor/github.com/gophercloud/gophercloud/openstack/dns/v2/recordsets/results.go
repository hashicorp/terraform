package recordsets

import (
	"encoding/json"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

type commonResult struct {
	gophercloud.Result
}

// Extract interprets a GetResult, CreateResult or UpdateResult as a concrete RecordSet.
// An error is returned if the original call or the extraction failed.
func (r commonResult) Extract() (*RecordSet, error) {
	var s *RecordSet
	err := r.ExtractInto(&s)
	return s, err
}

// CreateResult is the deferred result of a Create call.
type CreateResult struct {
	commonResult
}

// GetResult is the deferred result of a Get call.
type GetResult struct {
	commonResult
}

// RecordSetPage is a single page of RecordSet results.
type RecordSetPage struct {
	pagination.LinkedPageBase
}

// UpdateResult is the deferred result of an Update call.
type UpdateResult struct {
	commonResult
}

// DeleteResult is the deferred result of an Delete call.
type DeleteResult struct {
	gophercloud.ErrResult
}

// IsEmpty returns true if the page contains no results.
func (r RecordSetPage) IsEmpty() (bool, error) {
	s, err := ExtractRecordSets(r)
	return len(s) == 0, err
}

// ExtractRecordSets extracts a slice of RecordSets from a Collection acquired from List.
func ExtractRecordSets(r pagination.Page) ([]RecordSet, error) {
	var s struct {
		RecordSets []RecordSet `json:"recordsets"`
	}
	err := (r.(RecordSetPage)).ExtractInto(&s)
	return s.RecordSets, err
}

type RecordSet struct {
	// ID is the unique ID of the recordset
	ID string `json:"id"`

	// ZoneID is the ID of the zone the recordset belongs to.
	ZoneID string `json:"zone_id"`

	// ProjectID is the ID of the project that owns the recordset.
	ProjectID string `json:"project_id"`

	// Name is the name of the recordset.
	Name string `json:"name"`

	// ZoneName is the name of the zone the recordset belongs to.
	ZoneName string `json:"zone_name"`

	// Type is the RRTYPE of the recordset.
	Type string `json:"type"`

	// Records are the DNS records of the recordset.
	Records []string `json:"records"`

	// TTL is the time to live of the recordset.
	TTL int `json:"ttl"`

	// Status is the status of the recordset.
	Status string `json:"status"`

	// Action is the current action in progress of the recordset.
	Action string `json:"action"`

	// Description is the description of the recordset.
	Description string `json:"description"`

	// Version is the revision of the recordset.
	Version int `json:version"`

	// CreatedAt is the date when the recordset was created.
	CreatedAt time.Time `json:"-"`

	// UpdatedAt is the date when the recordset was updated.
	UpdatedAt time.Time `json:"-"`

	// Links includes HTTP references to the itself,
	// useful for passing along to other APIs that might want a recordset reference.
	Links []gophercloud.Link `json:"-"`
}

func (r *RecordSet) UnmarshalJSON(b []byte) error {
	type tmp RecordSet
	var s struct {
		tmp
		CreatedAt gophercloud.JSONRFC3339MilliNoZ `json:"created_at"`
		UpdatedAt gophercloud.JSONRFC3339MilliNoZ `json:"updated_at"`
		Links     map[string]interface{}          `json:"links"`
	}
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	*r = RecordSet(s.tmp)

	r.CreatedAt = time.Time(s.CreatedAt)
	r.UpdatedAt = time.Time(s.UpdatedAt)

	if s.Links != nil {
		for rel, href := range s.Links {
			if v, ok := href.(string); ok {
				link := gophercloud.Link{
					Rel:  rel,
					Href: v,
				}
				r.Links = append(r.Links, link)
			}
		}
	}

	return err
}
