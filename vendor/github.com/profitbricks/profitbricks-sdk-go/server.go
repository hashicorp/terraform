package profitbricks

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type Server struct {
	Id         string                     `json:"id,omitempty"`
	Type_      string                     `json:"type,omitempty"`
	Href       string                     `json:"href,omitempty"`
	Metadata   *DatacenterElementMetadata `json:"metadata,omitempty"`
	Properties ServerProperties           `json:"properties,omitempty"`
	Entities   *ServerEntities            `json:"entities,omitempty"`
	Response   string                     `json:"Response,omitempty"`
	Headers    *http.Header               `json:"headers,omitempty"`
	StatusCode int                        `json:"headers,omitempty"`
}

type ServerProperties struct {
	Name             string             `json:"name,omitempty"`
	Cores            int                `json:"cores,omitempty"`
	Ram              int                `json:"ram,omitempty"`
	AvailabilityZone string             `json:"availabilityZone,omitempty"`
	VmState          string             `json:"vmState,omitempty"`
	BootCdrom        *ResourceReference `json:"bootCdrom,omitempty"`
	BootVolume       *ResourceReference `json:"bootVolume,omitempty"`
	CpuFamily        string             `json:"cpuFamily,omitempty"`
}

type ServerEntities struct {
	Cdroms  *Cdroms  `json:"cdroms,omitempty"`
	Volumes *Volumes `json:"volumes,omitempty"`
	Nics    *Nics    `json:"nics,omitempty"`
}

type Servers struct {
	Id         string       `json:"id,omitempty"`
	Type_      string       `json:"type,omitempty"`
	Href       string       `json:"href,omitempty"`
	Items      []Server     `json:"items,omitempty"`
	Response   string       `json:"Response,omitempty"`
	Headers    *http.Header `json:"headers,omitempty"`
	StatusCode int          `json:"headers,omitempty"`
}

type ResourceReference struct {
	Id    string `json:"id,omitempty"`
	Type_ string `json:"type,omitempty"`
	Href  string `json:"href,omitempty"`
}

type CreateServerRequest struct {
	ServerProperties `json:"properties"`
}

// ListServers returns a server struct collection
func ListServers(dcid string) Servers {
	path := server_col_path(dcid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	resp := do(req)
	return toServers(resp)
}

// CreateServer creates a server from a jason []byte and returns a Instance struct
func CreateServer(dcid string, server Server) Server {
	obj, _ := json.Marshal(server)
	path := server_col_path(dcid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(obj))
	req.Header.Add("Content-Type", FullHeader)
	return toServer(do(req))
}

// GetServer pulls data for the server where id = srvid returns a Instance struct
func GetServer(dcid, srvid string) Server {
	path := server_path(dcid, srvid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return toServer(do(req))
}

// PatchServer partial update of server properties passed in as jason []byte
// Returns Instance struct
func PatchServer(dcid string, srvid string, props ServerProperties) Server {
	jason, _ := json.Marshal(props)
	path := server_path(dcid, srvid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(jason))
	req.Header.Add("Content-Type", PatchHeader)
	return toServer(do(req))
}

// DeleteServer deletes the server where id=srvid and returns Resp struct
func DeleteServer(dcid, srvid string) Resp {
	path := server_path(dcid, srvid)
	return is_delete(path)
}

func ListAttachedCdroms(dcid, srvid string) Images {
	path := server_cdrom_col_path(dcid, srvid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return toImages(do(req))
}

func AttachCdrom(dcid string, srvid string, cdid string) Image {
	jason := []byte(`{"id":"` + cdid + `"}`)
	path := server_cdrom_col_path(dcid, srvid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jason))
	req.Header.Add("Content-Type", FullHeader)
	return toImage(do(req))
}

func GetAttachedCdrom(dcid, srvid, cdid string) Volume {
	path := server_cdrom_path(dcid, srvid, cdid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return toVolume(do(req))
}

func DetachCdrom(dcid, srvid, cdid string) Resp {
	path := server_cdrom_path(dcid, srvid, cdid)
	return is_delete(path)
}

func ListAttachedVolumes(dcid, srvid string) Volumes {
	path := server_volume_col_path(dcid, srvid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	resp := do(req)
	return toVolumes(resp)
}

func AttachVolume(dcid string, srvid string, volid string) Volume {
	jason := []byte(`{"id":"` + volid + `"}`)
	path := server_volume_col_path(dcid, srvid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jason))
	req.Header.Add("Content-Type", FullHeader)
	return toVolume(do(req))
}

func GetAttachedVolume(dcid, srvid, volid string) Volume {
	path := server_volume_path(dcid, srvid, volid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	resp := do(req)
	return toVolume(resp)
}

func DetachVolume(dcid, srvid, volid string) Resp {
	path := server_volume_path(dcid, srvid, volid)
	return is_delete(path)
}

// StartServer starts a server
func StartServer(dcid, srvid string) Resp {
	return server_command(dcid, srvid, "start")
}

// StopServer stops a server
func StopServer(dcid, srvid string) Resp {
	return server_command(dcid, srvid, "stop")
}

// RebootServer reboots a server
func RebootServer(dcid, srvid string) Resp {
	return server_command(dcid, srvid, "reboot")
}

// server_command is a generic function for running server commands
func server_command(dcid, srvid, cmd string) Resp {
	jason := `
		{}
		`
	path := server_command_path(dcid, srvid, cmd)
	return is_command(path, jason)
}

func toServer(resp Resp) Server {
	var server Server
	json.Unmarshal(resp.Body, &server)
	server.Response = string(resp.Body)
	server.Headers = &resp.Headers
	server.StatusCode = resp.StatusCode
	return server
}

func toServers(resp Resp) Servers {
	var col Servers
	json.Unmarshal(resp.Body, &col)
	col.Response = string(resp.Body)
	col.Headers = &resp.Headers
	col.StatusCode = resp.StatusCode
	return col
}
