package common

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

const (
	// LocationDefaultEndpoint is the default API endpoint of Location services
	locationDefaultEndpoint = "https://location.aliyuncs.com"
	locationAPIVersion      = "2015-06-12"
	HTTP_PROTOCOL           = "http"
	HTTPS_PROTOCOL          = "https"
)

var (
	endpoints = make(map[Region]map[string]string)
)

//init endpoints from file
func init() {

}

func NewLocationClient(accessKeyId, accessKeySecret string) *Client {
	endpoint := os.Getenv("LOCATION_ENDPOINT")
	if endpoint == "" {
		endpoint = locationDefaultEndpoint
	}

	client := &Client{}
	client.Init(endpoint, locationAPIVersion, accessKeyId, accessKeySecret)
	return client
}

func (client *Client) DescribeEndpoint(args *DescribeEndpointArgs) (*DescribeEndpointResponse, error) {
	response := &DescribeEndpointResponse{}
	err := client.Invoke("DescribeEndpoint", args, response)
	if err != nil {
		return nil, err
	}
	return response, err
}

func getProductRegionEndpoint(region Region, serviceCode string) string {
	if sp, ok := endpoints[region]; ok {
		if endpoint, ok := sp[serviceCode]; ok {
			return endpoint
		}
	}

	return ""
}

func setProductRegionEndpoint(region Region, serviceCode string, endpoint string) {
	endpoints[region] = map[string]string{
		serviceCode: endpoint,
	}
}

func (client *Client) DescribeOpenAPIEndpoint(region Region, serviceCode string) string {
	if endpoint := getProductRegionEndpoint(region, serviceCode); endpoint != "" {
		return endpoint
	}

	defaultProtocols := HTTP_PROTOCOL

	args := &DescribeEndpointArgs{
		Id:          region,
		ServiceCode: serviceCode,
		Type:        "openAPI",
	}

	endpoint, err := client.DescribeEndpoint(args)
	if err != nil || endpoint.Endpoint == "" {
		return ""
	}

	for _, protocol := range endpoint.Protocols.Protocols {
		if strings.ToLower(protocol) == HTTPS_PROTOCOL {
			defaultProtocols = HTTPS_PROTOCOL
			break
		}
	}

	ep := fmt.Sprintf("%s://%s", defaultProtocols, endpoint.Endpoint)

	setProductRegionEndpoint(region, serviceCode, ep)
	return ep
}

func loadEndpointFromFile(region Region, serviceCode string) string {
	data, err := ioutil.ReadFile("./endpoints.xml")
	if err != nil {
		return ""
	}

	var endpoints Endpoints
	err = xml.Unmarshal(data, &endpoints)
	if err != nil {
		return ""
	}

	for _, endpoint := range endpoints.Endpoint {
		if endpoint.RegionIds.RegionId == string(region) {
			for _, product := range endpoint.Products.Product {
				if strings.ToLower(product.ProductName) == serviceCode {
					return fmt.Sprintf("%s://%s", HTTPS_PROTOCOL, product.DomainName)
				}
			}
		}
	}

	return ""
}
