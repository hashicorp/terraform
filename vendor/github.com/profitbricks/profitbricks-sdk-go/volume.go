package profitbricks

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type Volume struct {
	Id         string                     `json:"id,omitempty"`
	Type_      string                     `json:"type,omitempty"`
	Href       string                     `json:"href,omitempty"`
	Metadata   *DatacenterElementMetadata `json:"metadata,omitempty"`
	Properties VolumeProperties           `json:"properties,omitempty"`
	Response   string                     `json:"Response,omitempty"`
	Headers    *http.Header               `json:"headers,omitempty"`
	StatusCode int                        `json:"headers,omitempty"`
}

type VolumeProperties struct {
	Name                string   `json:"name,omitempty"`
	Type                string   `json:"type,omitempty"`
	Size                int      `json:"size,omitempty"`
	AvailabilityZone    string   `json:"availabilityZone,omitempty"`
	Image               string   `json:"image,omitempty"`
	ImagePassword       string   `json:"imagePassword,omitempty"`
	SshKeys             []string `json:"sshKeys,omitempty"`
	Bus                 string   `json:"bus,omitempty"`
	LicenceType         string   `json:"licenceType,omitempty"`
	CpuHotPlug          bool     `json:"cpuHotPlug,omitempty"`
	CpuHotUnplug        bool     `json:"cpuHotUnplug,omitempty"`
	RamHotPlug          bool     `json:"ramHotPlug,omitempty"`
	RamHotUnplug        bool     `json:"ramHotUnplug,omitempty"`
	NicHotPlug          bool     `json:"nicHotPlug,omitempty"`
	NicHotUnplug        bool     `json:"nicHotUnplug,omitempty"`
	DiscVirtioHotPlug   bool     `json:"discVirtioHotPlug,omitempty"`
	DiscVirtioHotUnplug bool     `json:"discVirtioHotUnplug,omitempty"`
	DiscScsiHotPlug     bool     `json:"discScsiHotPlug,omitempty"`
	DiscScsiHotUnplug   bool     `json:"discScsiHotUnplug,omitempty"`
	DeviceNumber        int64    `json:"deviceNumber,omitempty"`
}

type Volumes struct {
	Id         string       `json:"id,omitempty"`
	Type_      string       `json:"type,omitempty"`
	Href       string       `json:"href,omitempty"`
	Items      []Volume     `json:"items,omitempty"`
	Response   string       `json:"Response,omitempty"`
	Headers    *http.Header `json:"headers,omitempty"`
	StatusCode int          `json:"headers,omitempty"`
}

type CreateVolumeRequest struct {
	VolumeProperties `json:"properties"`
}

// ListVolumes returns a Collection struct for volumes in the Datacenter
func ListVolumes(dcid string) Volumes {
	path := volume_col_path(dcid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	resp := do(req)
	return toVolumes(resp)
}

func GetVolume(dcid string, volumeId string) Volume {
	path := volume_path(dcid, volumeId)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	resp := do(req)
	return toVolume(resp)
}

func PatchVolume(dcid string, volid string, request VolumeProperties) Volume {
	obj, _ := json.Marshal(request)
	path := volume_path(dcid, volid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(obj))
	req.Header.Add("Content-Type", PatchHeader)
	return toVolume(do(req))
}

func CreateVolume(dcid string, request Volume) Volume {
	obj, _ := json.Marshal(request)
	path := volume_col_path(dcid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(obj))
	req.Header.Add("Content-Type", FullHeader)
	return toVolume(do(req))
}

func DeleteVolume(dcid, volid string) Resp {
	path := volume_path(dcid, volid)
	return is_delete(path)
}

func CreateSnapshot(dcid string, volid string, name string) Snapshot {
	var path = volume_path(dcid, volid)
	path = path + "/create-snapshot"
	url := mk_url(path)
	body := json.RawMessage("name=" + name)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Add("Content-Type", CommandHeader)
	return toSnapshot(do(req))
}

func RestoreSnapshot(dcid string, volid string, snapshotId string) Resp {
	var path = volume_path(dcid, volid)
	path = path + "/restore-snapshot"

	return is_command(path, "snapshotId="+snapshotId)
}

func toVolume(resp Resp) Volume {
	var server Volume
	json.Unmarshal(resp.Body, &server)
	server.Response = string(resp.Body)
	server.Headers = &resp.Headers
	server.StatusCode = resp.StatusCode
	return server
}

func toVolumes(resp Resp) Volumes {
	var col Volumes
	json.Unmarshal(resp.Body, &col)
	col.Response = string(resp.Body)
	col.Headers = &resp.Headers
	col.StatusCode = resp.StatusCode
	return col
}
