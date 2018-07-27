package response

// ProviderList is the response structure for a pageable list of providers.
type ProviderList struct {
	Meta      PaginationMeta `json:"meta"`
	Providers []*Provider    `json:"providers"`
}
