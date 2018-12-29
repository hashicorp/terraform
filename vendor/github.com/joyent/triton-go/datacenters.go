package triton

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"

	"github.com/hashicorp/errwrap"
)

type DataCentersClient struct {
	*Client
}

// DataCenters returns a c used for accessing functions pertaining
// to Datacenter functionality in the Triton API.
func (c *Client) Datacenters() *DataCentersClient {
	return &DataCentersClient{c}
}

type DataCenter struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type ListDataCentersInput struct{}

func (client *DataCentersClient) ListDataCenters(*ListDataCentersInput) ([]*DataCenter, error) {
	respReader, err := client.executeRequest(http.MethodGet, "/my/datacenters", nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing ListDatacenters request: {{err}}", err)
	}

	var intermediate map[string]string
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&intermediate); err != nil {
		return nil, errwrap.Wrapf("Error decoding ListDatacenters response: {{err}}", err)
	}

	keys := make([]string, len(intermediate))
	i := 0
	for k := range intermediate {
		keys[i] = k
		i++
	}
	sort.Strings(keys)

	result := make([]*DataCenter, len(intermediate))
	i = 0
	for _, key := range keys {
		result[i] = &DataCenter{
			Name: key,
			URL:  intermediate[key],
		}
		i++
	}

	return result, nil
}

type GetDataCenterInput struct {
	Name string
}

func (client *DataCentersClient) GetDataCenter(input *GetDataCenterInput) (*DataCenter, error) {
	resp, err := client.executeRequestRaw(http.MethodGet, fmt.Sprintf("/my/datacenters/%s", input.Name), nil)
	if err != nil {
		return nil, errwrap.Wrapf("Error executing GetDatacenter request: {{err}}", err)
	}

	if resp.StatusCode != http.StatusFound {
		return nil, fmt.Errorf("Error executing GetDatacenter request: expected status code 302, got %s",
			resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return nil, errors.New("Error decoding GetDatacenter response: no Location header")
	}

	return &DataCenter{
		Name: input.Name,
		URL:  location,
	}, nil
}
