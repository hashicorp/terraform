package nsone

type Answer struct {
	Region string                 `json:"region,omitempty"`
	Answer []string               `json:"answer,omitempty"`
	Meta   map[string]interface{} `json:"meta,omitempty"`
}

type Filter struct {
	Filter   string                 `json:"filter"`
	Disabled bool                   `json:"disabled,omitempty"`
	Config   map[string]interface{} `json:"config"`
}

type Record struct {
	Id              string            `json:"id,omitempty"`
	Zone            string            `json:"zone,omitempty"`
	Domain          string            `json:"domain,omitempty"`
	Type            string            `json:"type,omitempty"`
	Link            string            `json:"link,omitempty"`
	Meta            map[string]string `json:"meta,omitempty"`
	Answers         []Answer          `json:"answers"`
	Filters         []Filter          `json:"filters,omitempty"`
	Ttl             int               `json:"ttl,omitempty"`
	UseClientSubnet bool              `json:"use_client_subnet"`
	Regions         map[string]Region `json:"regions,omitempty"`
}

type Region struct {
	Meta RegionMeta `json:"meta"`
}

type RegionMeta struct {
	GeoRegion []string `json:"georegion,omitempty"`
	Country   []string `json:"country,omitempty"`
	USState   []string `json:"us_state,omitempty"`
	Up        bool     `json:"up,omitempty"`
}

type MetaFeed struct {
	Feed string `json:"feed"`
}

type MetaStatic string

func NewRecord(zone string, domain string, t string) *Record {
	return &Record{
		Zone:            zone,
		Domain:          domain,
		Type:            t,
		UseClientSubnet: true,
		Answers:         make([]Answer, 0),
	}
}

func NewAnswer() Answer {
	return Answer{
		Meta: make(map[string]interface{}),
	}
}

func NewMetaFeed(feed_id string) MetaFeed {
	return MetaFeed{
		Feed: feed_id,
	}
}

func (r *Record) LinkTo(to string) {
	r.Meta = nil
	r.Answers = make([]Answer, 0)
	r.Link = to
}
