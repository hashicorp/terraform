package librato

import (
	"fmt"
	"net/url"
)

// PaginationResponseMeta contains pagination metadata from Librato API
// responses.
type PaginationResponseMeta struct {
	Offset uint `json:"offset"`
	Length uint `json:"length"`
	Total  uint `json:"total"`
	Found  uint `json:"found"`
}

// Calculate the pagination metadata for the next page of the result set.
// Takes the metadata used to request the current page so that it can use the
// same sort/orderby options
func (p *PaginationResponseMeta) nextPage(originalQuery *PaginationMeta) (next *PaginationMeta) {
	nextOffset := p.Offset + p.Length

	if nextOffset >= p.Found {
		return nil
	}

	next = &PaginationMeta{}
	next.Offset = nextOffset
	next.Length = p.Length

	if originalQuery != nil {
		next.OrderBy = originalQuery.OrderBy
		next.Sort = originalQuery.Sort
	}

	return next
}

// PaginationMeta contains metadata that the Librato API requires for pagination
// http://dev.librato.com/v1/pagination
type PaginationMeta struct {
	Offset  uint   `url:"offset,omitempty"`
	Length  uint   `url:"length,omitempty"`
	OrderBy string `url:"orderby,omitempty"`
	Sort    string `url:"sort,omitempty"`
}

// EncodeValues is implemented to allow other strucs to embed PaginationMeta and
// still use github.com/google/go-querystring/query to encode the struct. It
// makes PaginationMeta implement query.Encoder.
func (m *PaginationMeta) EncodeValues(name string, values *url.Values) error {
	if m == nil {
		return nil
	}

	if m.Offset != 0 {
		values.Set("offset", fmt.Sprintf("%d", m.Offset))
	}
	if m.Length != 0 {
		values.Set("length", fmt.Sprintf("%d", m.Length))
	}
	if m.OrderBy != "" {
		values.Set("orderby", m.OrderBy)
	}
	if m.Sort != "" {
		values.Set("sort", m.Sort)
	}

	return nil
}
