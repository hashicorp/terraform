package profitbricks

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type Snapshot struct {
	Id         string                    `json:"id,omitempty"`
	Type_      string                    `json:"type,omitempty"`
	Href       string                    `json:"href,omitempty"`
	Metadata   DatacenterElementMetadata `json:"metadata,omitempty"`
	Properties SnapshotProperties        `json:"properties,omitempty"`
	Response   string                    `json:"Response,omitempty"`
	Headers    *http.Header              `json:"headers,omitempty"`
	StatusCode int                       `json:"headers,omitempty"`
}

type SnapshotProperties struct {
	Name                string `json:"name,omitempty"`
	Description         string `json:"description,omitempty"`
	Location            string `json:"location,omitempty"`
	Size                int    `json:"size,omitempty"`
	CpuHotPlug          bool   `json:"cpuHotPlug,omitempty"`
	CpuHotUnplug        bool   `json:"cpuHotUnplug,omitempty"`
	RamHotPlug          bool   `json:"ramHotPlug,omitempty"`
	RamHotUnplug        bool   `json:"ramHotUnplug,omitempty"`
	NicHotPlug          bool   `json:"nicHotPlug,omitempty"`
	NicHotUnplug        bool   `json:"nicHotUnplug,omitempty"`
	DiscVirtioHotPlug   bool   `json:"discVirtioHotPlug,omitempty"`
	DiscVirtioHotUnplug bool   `json:"discVirtioHotUnplug,omitempty"`
	DiscScsiHotPlug     bool   `json:"discScsiHotPlug,omitempty"`
	DiscScsiHotUnplug   bool   `json:"discScsiHotUnplug,omitempty"`
	LicenceType         string `json:"licenceType,omitempty"`
}

type Snapshots struct {
	Id         string       `json:"id,omitempty"`
	Type_      string       `json:"type,omitempty"`
	Href       string       `json:"href,omitempty"`
	Items      []Snapshot   `json:"items,omitempty"`
	Response   string       `json:"Response,omitempty"`
	Headers    *http.Header `json:"headers,omitempty"`
	StatusCode int          `json:"headers,omitempty"`
}

func ListSnapshots() Snapshots {
	path := snapshot_col_path()
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return toSnapshots(do(req))
}

func GetSnapshot(snapshotId string) Snapshot {
	path := snapshot_col_path() + slash(snapshotId)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return toSnapshot(do(req))
}

func DeleteSnapshot(snapshotId string) Resp {
	path := snapshot_col_path() + slash(snapshotId)
	url := mk_url(path)
	req, _ := http.NewRequest("DELETE", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return do(req)
}

func UpdateSnapshot(snapshotId string, request SnapshotProperties) Snapshot {
	path := snapshot_col_path() + slash(snapshotId)
	obj, _ := json.Marshal(request)
	url := mk_url(path)
	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(obj))
	req.Header.Add("Content-Type", PatchHeader)
	return toSnapshot(do(req))
}

func toSnapshot(resp Resp) Snapshot {
	var lan Snapshot
	json.Unmarshal(resp.Body, &lan)
	lan.Response = string(resp.Body)
	lan.Headers = &resp.Headers
	lan.StatusCode = resp.StatusCode
	return lan
}
func toSnapshots(resp Resp) Snapshots {
	var col Snapshots
	json.Unmarshal(resp.Body, &col)
	col.Response = string(resp.Body)
	col.Headers = &resp.Headers
	col.StatusCode = resp.StatusCode
	return col
}
