package dockercloud

import "encoding/json"

func ListProviders() (ProviderListResponse, error) {

	url := "infra/" + infraSubsytemVersion + "/provider/"
	request := "GET"
	//Empty Body Request
	body := []byte(`{}`)
	var response ProviderListResponse
	var finalResponse ProviderListResponse

	data, err := DockerCloudCall(url, request, body)
	if err != nil {
		return response, err
	}

	err = json.Unmarshal(data, &response)
	if err != nil {
		return response, err
	}

	finalResponse = response

Loop:
	for {
		if response.Meta.Next != "" {
			var nextResponse ProviderListResponse
			data, err := DockerCloudCall(response.Meta.Next[5:], request, body)
			if err != nil {
				return nextResponse, err
			}
			err = json.Unmarshal(data, &nextResponse)
			if err != nil {
				return nextResponse, err
			}
			finalResponse.Objects = append(finalResponse.Objects, nextResponse.Objects...)
			response = nextResponse

		} else {
			break Loop
		}
	}

	return finalResponse, nil
}

func GetProvider(name string) (Provider, error) {

	url := ""
	if string(name[0]) == "/" {
		url = name[5:]
	} else {
		url = "infra/" + infraSubsytemVersion + "/provider/" + name + "/"
	}

	request := "GET"
	//Empty Body Request
	body := []byte(`{}`)
	var response Provider

	data, err := DockerCloudCall(url, request, body)
	if err != nil {
		return response, err
	}

	err = json.Unmarshal(data, &response)
	if err != nil {
		return response, err
	}

	return response, nil
}
