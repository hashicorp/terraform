package contentful

// Sys model
type Sys struct {
	ID               string       `json:"id,omitempty"`
	Type             string       `json:"type,omitempty"`
	LinkType         string       `json:"linkType,omitempty"`
	CreatedAt        string       `json:"createdAt,omitempty"`
	UpdatedAt        string       `json:"updatedAt,omitempty"`
	UpdatedBy        *Sys         `json:"updatedBy,omitempty"`
	Version          int          `json:"version,omitempty"`
	Revision         int          `json:"revision,omitempty"`
	ContentType      *ContentType `json:"contentType,omitempty"`
	Space            *Space       `json:"space,omitempty"`
	FirstPublishedAt string       `json:"firstPublishedAt,omitempty"`
	PublishedCounter int          `json:"publishedCounter,omitempty"`
	PublishedAt      string       `json:"publishedAt,omitempty"`
	PublishedBy      *Sys         `json:"publishedBy,omitempty"`
	PublishedVersion int          `json:"publishedVersion,omitempty"`
}
