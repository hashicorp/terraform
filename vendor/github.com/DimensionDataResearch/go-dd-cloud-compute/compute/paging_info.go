package compute

// PagedResult represents the common fields for all paged results from the compute API.
type PagedResult struct {
	// The current page number.
	PageNumber int `json:"pageNumber"`

	// The number of items in the current page of results.
	PageCount int `json:"pageCount"`

	// The total number of results that match the requested filter criteria (if any).
	TotalCount int `json:"totalCount"`

	// The maximum number of results per page.
	PageSize int `json:"pageSize"`
}

// NextPage creates a PagingInfo for the next page of results.
func (page *PagedResult) NextPage() *PagingInfo {
	return &PagingInfo{
		PageNumber: page.PageNumber + 1,
		PageSize:   page.PageSize,
	}
}

// PagingInfo contains the paging configuration for a compute API operation.
type PagingInfo struct {
	PageNumber int
	PageSize   int
}

func (pagingInfo *PagingInfo) ensureValidPageSize() {
	if pagingInfo.PageSize < 5 {
		pagingInfo.PageSize = 5
	}
}

// First configures the PagingInfo for the first page of results.
func (pagingInfo *PagingInfo) First() {
	pagingInfo.ensureValidPageSize()

	pagingInfo.PageNumber = 1
}

// Next configures the PagingInfo for the next page of results.
func (pagingInfo *PagingInfo) Next() {
	pagingInfo.ensureValidPageSize()

	pagingInfo.PageNumber++
}
