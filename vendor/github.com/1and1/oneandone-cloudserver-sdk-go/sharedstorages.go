package oneandone

import (
	"net/http"
)

type SharedStorage struct {
	Identity
	descField
	Size           int                   `json:"size"`
	MinSizeAllowed int                   `json:"minimum_size_allowed"`
	SizeUsed       string                `json:"size_used,omitempty"`
	State          string                `json:"state,omitempty"`
	CloudPanelId   string                `json:"cloudpanel_id,omitempty"`
	SiteId         string                `json:"site_id,omitempty"`
	CifsPath       string                `json:"cifs_path,omitempty"`
	NfsPath        string                `json:"nfs_path,omitempty"`
	CreationDate   string                `json:"creation_date,omitempty"`
	Servers        []SharedStorageServer `json:"servers,omitempty"`
	Datacenter     *Datacenter           `json:"datacenter,omitempty"`
	ApiPtr
}

type SharedStorageServer struct {
	Id     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Rights string `json:"rights,omitempty"`
}

type SharedStorageRequest struct {
	DatacenterId string `json:"datacenter_id,omitempty"`
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
	Size         *int   `json:"size"`
}

type SharedStorageAccess struct {
	State               string `json:"state,omitempty"`
	KerberosContentFile string `json:"kerberos_content_file,omitempty"`
	UserDomain          string `json:"user_domain,omitempty"`
	SiteId              string `json:"site_id,omitempty"`
	NeedsPasswordReset  int    `json:"needs_password_reset"`
}

// GET /shared_storages
func (api *API) ListSharedStorages(args ...interface{}) ([]SharedStorage, error) {
	url, err := processQueryParams(createUrl(api, sharedStoragePathSegment), args...)
	if err != nil {
		return nil, err
	}
	result := []SharedStorage{}
	err = api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	for index, _ := range result {
		result[index].api = api
	}
	return result, nil
}

// POST /shared_storages
func (api *API) CreateSharedStorage(request *SharedStorageRequest) (string, *SharedStorage, error) {
	result := new(SharedStorage)
	url := createUrl(api, sharedStoragePathSegment)
	err := api.Client.Post(url, request, &result, http.StatusAccepted)
	if err != nil {
		return "", nil, err
	}
	result.api = api
	return result.Id, result, nil
}

// GET /shared_storages/{id}
func (api *API) GetSharedStorage(ss_id string) (*SharedStorage, error) {
	result := new(SharedStorage)
	url := createUrl(api, sharedStoragePathSegment, ss_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// DELETE /shared_storages/{id}
func (api *API) DeleteSharedStorage(ss_id string) (*SharedStorage, error) {
	result := new(SharedStorage)
	url := createUrl(api, sharedStoragePathSegment, ss_id)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// PUT /shared_storages/{id}
func (api *API) UpdateSharedStorage(ss_id string, request *SharedStorageRequest) (*SharedStorage, error) {
	result := new(SharedStorage)
	url := createUrl(api, sharedStoragePathSegment, ss_id)
	err := api.Client.Put(url, &request, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /shared_storages/{id}/servers
func (api *API) ListSharedStorageServers(st_id string) ([]SharedStorageServer, error) {
	result := []SharedStorageServer{}
	url := createUrl(api, sharedStoragePathSegment, st_id, "servers")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// POST /shared_storages/{id}/servers
func (api *API) AddSharedStorageServers(st_id string, servers []SharedStorageServer) (*SharedStorage, error) {
	result := new(SharedStorage)
	req := struct {
		Servers []SharedStorageServer `json:"servers"`
	}{servers}
	url := createUrl(api, sharedStoragePathSegment, st_id, "servers")
	err := api.Client.Post(url, &req, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /shared_storages/{id}/servers/{id}
func (api *API) GetSharedStorageServer(st_id string, ser_id string) (*SharedStorageServer, error) {
	result := new(SharedStorageServer)
	url := createUrl(api, sharedStoragePathSegment, st_id, "servers", ser_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DELETE /shared_storages/{id}/servers/{id}
func (api *API) DeleteSharedStorageServer(st_id string, ser_id string) (*SharedStorage, error) {
	result := new(SharedStorage)
	url := createUrl(api, sharedStoragePathSegment, st_id, "servers", ser_id)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /shared_storages/access
func (api *API) GetSharedStorageCredentials() ([]SharedStorageAccess, error) {
	result := []SharedStorageAccess{}
	url := createUrl(api, sharedStoragePathSegment, "access")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// PUT /shared_storages/access
func (api *API) UpdateSharedStorageCredentials(new_pass string) ([]SharedStorageAccess, error) {
	result := []SharedStorageAccess{}
	req := struct {
		Password string `json:"password"`
	}{new_pass}
	url := createUrl(api, sharedStoragePathSegment, "access")
	err := api.Client.Put(url, &req, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (ss *SharedStorage) GetState() (string, error) {
	in, err := ss.api.GetSharedStorage(ss.Id)
	if in == nil {
		return "", err
	}
	return in.State, err
}
