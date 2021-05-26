package response

import (
	"net/url"
	"strconv"
)

// PaginationMeta is a structure included in responses for pagination.
type PaginationMeta struct {
	Limit         int    `json:"limit"`
	CurrentOffset int    `json:"current_offset"`
	NextOffset    *int   `json:"next_offset,omitempty"`
	PrevOffset    *int   `json:"prev_offset,omitempty"`
	NextURL       string `json:"next_url,omitempty"`
	PrevURL       string `json:"prev_url,omitempty"`
}

// NewPaginationMeta populates pagination meta data from result parameters
func NewPaginationMeta(offset, limit int, hasMore bool, currentURL string) PaginationMeta {
	pm := PaginationMeta{
		Limit:         limit,
		CurrentOffset: offset,
	}

	// Calculate next/prev offsets, leave nil if not valid pages
	nextOffset := offset + limit
	if hasMore {
		pm.NextOffset = &nextOffset
	}

	prevOffset := offset - limit
	if prevOffset < 0 {
		prevOffset = 0
	}
	if prevOffset < offset {
		pm.PrevOffset = &prevOffset
	}

	// If URL format provided, populate URLs. Intentionally swallow URL errors for now, API should
	// catch missing URLs if we call with bad URL arg (and we care about them being present).
	if currentURL != "" && pm.NextOffset != nil {
		pm.NextURL, _ = setQueryParam(currentURL, "offset", *pm.NextOffset, 0)
	}
	if currentURL != "" && pm.PrevOffset != nil {
		pm.PrevURL, _ = setQueryParam(currentURL, "offset", *pm.PrevOffset, 0)
	}

	return pm
}

func setQueryParam(baseURL, key string, val, defaultVal int) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	if val == defaultVal {
		// elide param if it's the default value
		q.Del(key)
	} else {
		q.Set(key, strconv.Itoa(val))
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
