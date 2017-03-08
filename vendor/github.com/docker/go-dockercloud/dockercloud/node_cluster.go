package dockercloud

import "encoding/json"

func ListNodeClusters() (NodeClusterListResponse, error) {

	url := ""
	if Namespace != "" {
		url = "infra/" + infraSubsytemVersion + "/" + Namespace + "/nodecluster/"
	} else {
		url = "infra/" + infraSubsytemVersion + "/nodecluster/"
	}
	request := "GET"

	//Empty Body Request
	body := []byte(`{}`)
	var response NodeClusterListResponse
	var finalResponse NodeClusterListResponse

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
			var nextResponse NodeClusterListResponse
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

func GetNodeCluster(uuid string) (NodeCluster, error) {

	url := ""
	if string(uuid[0]) == "/" {
		url = uuid[5:] + "/"
	} else {
		if Namespace != "" {
			url = "infra/" + infraSubsytemVersion + "/" + Namespace + "/nodecluster/" + uuid + "/"
		} else {
			url = "infra/" + infraSubsytemVersion + "/nodecluster/" + uuid + "/"
		}
	}

	request := "GET"
	//Empty Body Request
	body := []byte(`{}`)
	var response NodeCluster

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

func CreateNodeCluster(createRequest NodeCreateRequest) (NodeCluster, error) {

	url := ""
	if Namespace != "" {
		url = "infra/" + infraSubsytemVersion + "/" + Namespace + "/nodecluster/"
	} else {
		url = "infra/" + infraSubsytemVersion + "/nodecluster/"
	}

	request := "POST"
	var response NodeCluster

	newCluster, err := json.Marshal(createRequest)
	if err != nil {
		return response, err
	}

	data, err := DockerCloudCall(url, request, newCluster)
	if err != nil {
		return response, err
	}

	err = json.Unmarshal(data, &response)
	if err != nil {
		return response, err
	}

	return response, nil
}

func (self *NodeCluster) Deploy() error {

	url := ""
	if Namespace != "" {
		url = "infra/" + infraSubsytemVersion + "/" + Namespace + "/nodecluster/" + self.Uuid + "/deploy/"
	} else {
		url = "infra/" + infraSubsytemVersion + "/nodecluster/" + self.Uuid + "/deploy/"
	}
	request := "POST"
	//Empty Body Request
	body := []byte(`{}`)

	_, err := DockerCloudCall(url, request, body)
	if err != nil {
		return err
	}

	return nil
}

func (self *NodeCluster) Update(createRequest NodeCreateRequest) error {

	url := ""
	if Namespace != "" {
		url = "infra/" + infraSubsytemVersion + "/" + Namespace + "/nodecluster/" + self.Uuid + "/"
	} else {
		url = "infra/" + infraSubsytemVersion + "/nodecluster/" + self.Uuid + "/"
	}
	request := "PATCH"

	updatedNodeCluster, err := json.Marshal(createRequest)
	if err != nil {
		return err
	}

	_, errr := DockerCloudCall(url, request, updatedNodeCluster)
	if errr != nil {
		return errr
	}

	return nil
}

func (self *NodeCluster) Upgrade() error {

	url := ""
	if Namespace != "" {
		url = "infra/" + infraSubsytemVersion + "/" + Namespace + "/nodecluster/" + self.Uuid + "/docker-upgrade/"
	} else {
		url = "infra/" + infraSubsytemVersion + "/nodecluster/" + self.Uuid + "/docker-upgrade/"
	}
	request := "POST"
	//Empty Body Request
	body := []byte(`{}`)

	_, err := DockerCloudCall(url, request, body)
	if err != nil {
		return err
	}

	return nil
}

func (self *NodeCluster) Terminate() error {

	url := ""
	if Namespace != "" {
		url = "infra/" + infraSubsytemVersion + "/" + Namespace + "/nodecluster/" + self.Uuid + "/"
	} else {
		url = "infra/" + infraSubsytemVersion + "/nodecluster/" + self.Uuid + "/"
	}
	request := "DELETE"
	//Empty Body Request
	body := []byte(`{}`)

	_, err := DockerCloudCall(url, request, body)
	if err != nil {
		return err
	}

	return nil
}
