package dockercloud

import "encoding/json"

func ListNodeTypes() (NodeTypeListResponse, error) {

	url := "infra/" + infraSubsytemVersion + "/nodetype/"
	request := "GET"

	//Empty Body Request
	body := []byte(`{}`)
	var response NodeTypeListResponse
	var finalResponse NodeTypeListResponse

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
			var nextResponse NodeTypeListResponse
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

	return response, nil
}

func GetNodeType(provider string, name string) (NodeType, error) {
	url := "infra/" + infraSubsytemVersion + "/nodetype/" + provider + "/" + name + "/"
	request := "GET"
	body := []byte(`{}`)
	var response NodeType

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
