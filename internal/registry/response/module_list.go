package response

// ModuleList is the response structure for a pageable list of modules.
type ModuleList struct {
	Meta    PaginationMeta `json:"meta"`
	Modules []*Module      `json:"modules"`
}
