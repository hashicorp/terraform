package nsone

type DataFeed struct {
	SourceId string                 `json:"-"`
	Id       string                 `json:"id,omitempty"`
	Name     string                 `json:"name"`
	Config   map[string]string      `json:"config,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

func NewDataFeed(source_id string) *DataFeed {
	return &DataFeed{SourceId: source_id}
}
