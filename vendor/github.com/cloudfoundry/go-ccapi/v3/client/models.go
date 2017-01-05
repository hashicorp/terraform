package client

type GetResourcesResponse struct {
	Pagination Pagination    `json:"pagination,omitempty"`
	Resources  []interface{} `json:"resources,omitempty"`
}

type Pagination struct {
	TotalResults *int    `json:"total_results"`
	First        *Link   `json:"first"`
	Last         *Link   `json:"last"`
	Next         *string `json:"next"`
	Previous     *string `json:"previous"`
}

type Link struct {
	Href string `json:"href"`
}
