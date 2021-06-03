package response

// Redirect causes the frontend to perform a window redirect.
type Redirect struct {
	URL string `json:"url"`
}
