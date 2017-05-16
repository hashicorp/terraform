package oneandone

import (
	"net/http"
)

type Image struct {
	idField
	ImageConfig
	MinHddSize   int         `json:"min_hdd_size"`
	Architecture *int        `json:"os_architecture"`
	CloudPanelId string      `json:"cloudpanel_id,omitempty"`
	CreationDate string      `json:"creation_date,omitempty"`
	State        string      `json:"state,omitempty"`
	OsImageType  string      `json:"os_image_type,omitempty"`
	OsFamily     string      `json:"os_family,omitempty"`
	Os           string      `json:"os,omitempty"`
	OsVersion    string      `json:"os_version,omitempty"`
	Type         string      `json:"type,omitempty"`
	Licenses     []License   `json:"licenses,omitempty"`
	Hdds         []Hdd       `json:"hdds,omitempty"`
	Datacenter   *Datacenter `json:"datacenter,omitempty"`
	ApiPtr
}

type ImageConfig struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Frequency   string `json:"frequency,omitempty"`
	ServerId    string `json:"server_id,omitempty"`
	NumImages   int    `json:"num_images"`
}

// GET /images
func (api *API) ListImages(args ...interface{}) ([]Image, error) {
	url, err := processQueryParams(createUrl(api, imagePathSegment), args...)
	if err != nil {
		return nil, err
	}
	result := []Image{}
	err = api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	for index, _ := range result {
		result[index].api = api
	}
	return result, nil
}

// POST /images
func (api *API) CreateImage(request *ImageConfig) (string, *Image, error) {
	res := new(Image)
	url := createUrl(api, imagePathSegment)
	err := api.Client.Post(url, &request, &res, http.StatusAccepted)
	if err != nil {
		return "", nil, err
	}
	res.api = api
	return res.Id, res, nil
}

// GET /images/{id}
func (api *API) GetImage(img_id string) (*Image, error) {
	result := new(Image)
	url := createUrl(api, imagePathSegment, img_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// DELETE /images/{id}
func (api *API) DeleteImage(img_id string) (*Image, error) {
	result := new(Image)
	url := createUrl(api, imagePathSegment, img_id)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// PUT /images/{id}
func (api *API) UpdateImage(img_id string, new_name string, new_desc string, new_freq string) (*Image, error) {
	result := new(Image)
	req := struct {
		Name        string `json:"name,omitempty"`
		Description string `json:"description,omitempty"`
		Frequency   string `json:"frequency,omitempty"`
	}{Name: new_name, Description: new_desc, Frequency: new_freq}
	url := createUrl(api, imagePathSegment, img_id)
	err := api.Client.Put(url, &req, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

func (im *Image) GetState() (string, error) {
	in, err := im.api.GetImage(im.Id)
	if in == nil {
		return "", err
	}
	return in.State, err
}
