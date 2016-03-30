package dockercloud

import "encoding/json"

func ListRegions() (RegionListResponse, error) {

	url := "infra/" + infraSubsytemVersion + "/region/"
	request := "GET"
	//Empty Body Request
	body := []byte(`{}`)
	var response RegionListResponse
	var finalResponse RegionListResponse

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
			var nextResponse RegionListResponse
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

func GetRegion(id string) (Region, error) {

	url := ""
	if string(id[0]) == "/" {
		url = id[5:]
	} else {
		url = "infra/" + infraSubsytemVersion + "/region/" + id + "/"
	}

	request := "GET"
	//Empty Body Request
	body := []byte(`{}`)
	var response Region

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
