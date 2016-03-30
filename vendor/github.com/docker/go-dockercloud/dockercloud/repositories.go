package dockercloud

import "encoding/json"

func ListRepositories() (RepositoryListResponse, error) {
	url := "repo/" + repoSubsystemVersion + "/repository/"
	request := "GET"
	//Empty Body Request
	body := []byte(`{}`)
	var response RepositoryListResponse
	var finalResponse RepositoryListResponse

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
			var nextResponse RepositoryListResponse
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

func GetRepository(name string) (Repository, error) {

	url := ""
	if string(name[0]) == "/" {
		url = name[5:]
	} else {
		url = "repo/" + repoSubsystemVersion + "/repository/" + name + "/"
	}

	request := "GET"
	//Empty Body Request
	body := []byte(`{}`)
	var response Repository

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

func CreateRepository(createRequest RepositoryCreateRequest) (Repository, error) {

	url := "repo/" + repoSubsystemVersion + "/repository/"
	request := "POST"
	var response Repository

	newRepository, err := json.Marshal(createRequest)
	if err != nil {
		return response, err
	}

	data, err := DockerCloudCall(url, request, newRepository)
	if err != nil {
		return response, err
	}

	err = json.Unmarshal(data, &response)
	if err != nil {
		return response, err
	}

	return response, nil
}

func (self *Repository) Update(createRequest RepositoryCreateRequest) error {

	url := "repo/" + repoSubsystemVersion + "/repository/" + self.Name + "/"
	request := "PATCH"

	updatedRepository, err := json.Marshal(createRequest)
	if err != nil {
		return err
	}

	_, err = DockerCloudCall(url, request, updatedRepository)
	if err != nil {
		return err
	}

	return nil
}

func (self *Repository) Remove() error {
	url := "repo/" + repoSubsystemVersion + "/repository/" + self.Name + "/"
	request := "DELETE"
	//Empty Body Request
	body := []byte(`{}`)

	_, err := DockerCloudCall(url, request, body)
	if err != nil {
		return err
	}

	return nil
}
