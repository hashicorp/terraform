package nsone

// DataFeed wraps an NS1 /data/feeds resource
type DataFeed struct {
	SourceId string                 `json:"-"`
	Id       string                 `json:"id,omitempty"`
	Name     string                 `json:"name"`
	Config   map[string]string      `json:"config,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

// NewDataFeed takes a sourceID and creates a new *DataFeed
func NewDataFeed(sourceID string) *DataFeed {
	return &DataFeed{SourceId: sourceID}
}
