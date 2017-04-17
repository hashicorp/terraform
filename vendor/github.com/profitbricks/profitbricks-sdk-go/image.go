package profitbricks

import (
	"encoding/json"
	"net/http"
)

type Image struct {
	Id         string                     `json:"id,omitempty"`
	Type       string                     `json:"type,omitempty"`
	Href       string                     `json:"href,omitempty"`
	Metadata   *DatacenterElementMetadata `json:"metadata,omitempty"`
	Properties ImageProperties            `json:"properties,omitempty"`
	Response   string                     `json:"Response,omitempty"`
	Headers    *http.Header               `json:"headers,omitempty"`
	StatusCode int                        `json:"headers,omitempty"`
}

type ImageProperties struct {
	Name                string       `json:"name,omitempty"`
	Description         string       `json:"description,omitempty"`
	Location            string       `json:"location,omitempty"`
	Size                int          `json:"size,omitempty"`
	CpuHotPlug          bool         `json:"cpuHotPlug,omitempty"`
	CpuHotUnplug        bool         `json:"cpuHotUnplug,omitempty"`
	RamHotPlug          bool         `json:"ramHotPlug,omitempty"`
	RamHotUnplug        bool         `json:"ramHotUnplug,omitempty"`
	NicHotPlug          bool         `json:"nicHotPlug,omitempty"`
	NicHotUnplug        bool         `json:"nicHotUnplug,omitempty"`
	DiscVirtioHotPlug   bool         `json:"discVirtioHotPlug,omitempty"`
	DiscVirtioHotUnplug bool         `json:"discVirtioHotUnplug,omitempty"`
	DiscScsiHotPlug     bool         `json:"discScsiHotPlug,omitempty"`
	DiscScsiHotUnplug   bool         `json:"discScsiHotUnplug,omitempty"`
	LicenceType         string       `json:"licenceType,omitempty"`
	ImageType           string       `json:"imageType,omitempty"`
	Public              bool         `json:"public,omitempty"`
	Response            string       `json:"Response,omitempty"`
	Headers             *http.Header `json:"headers,omitempty"`
	StatusCode          int          `json:"headers,omitempty"`
}

type Images struct {
	Id         string       `json:"id,omitempty"`
	Type_      string       `json:"type,omitempty"`
	Href       string       `json:"href,omitempty"`
	Items      []Image      `json:"items,omitempty"`
	Response   string       `json:"Response,omitempty"`
	Headers    *http.Header `json:"headers,omitempty"`
	StatusCode int          `json:"headers,omitempty"`
}

type Cdroms struct {
	Id         string       `json:"id,omitempty"`
	Type_      string       `json:"type,omitempty"`
	Href       string       `json:"href,omitempty"`
	Items      []Image      `json:"items,omitempty"`
	Response   string       `json:"Response,omitempty"`
	Headers    *http.Header `json:"headers,omitempty"`
	StatusCode int          `json:"headers,omitempty"`
}

// ListImages returns an Collection struct
func ListImages() Images {
	path := image_col_path()
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	resp := do(req)
	return toImages(resp)
}

// GetImage returns an Instance struct where id ==imageid
func GetImage(imageid string) Image {
	path := image_path(imageid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	resp := do(req)
	return toImage(resp)
}

func toImage(resp Resp) Image {
	var image Image
	json.Unmarshal(resp.Body, &image)
	image.Response = string(resp.Body)
	image.Headers = &resp.Headers
	image.StatusCode = resp.StatusCode
	return image
}

func toImages(resp Resp) Images {
	var col Images
	json.Unmarshal(resp.Body, &col)
	col.Response = string(resp.Body)
	col.Headers = &resp.Headers
	col.StatusCode = resp.StatusCode
	return col
}
