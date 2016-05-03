package dockercloud

import (
	"encoding/json"
	"log"
)

func ListStacks() (StackListResponse, error) {
	url := "app/" + appSubsystemVersion + "/stack/"
	request := "GET"

	//Empty Body Request
	body := []byte(`{}`)
	var response StackListResponse
	var finalResponse StackListResponse

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
			var nextResponse StackListResponse
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

func GetStack(uuid string) (Stack, error) {

	url := ""
	if string(uuid[0]) == "/" {
		url = uuid[5:]
	} else {
		url = "app/" + appSubsystemVersion + "/stack/" + uuid + "/"
	}

	request := "GET"
	//Empty Body Request
	body := []byte(`{}`)
	var response Stack

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

func (self *Stack) ExportStack() (string, error) {

	url := "app/" + appSubsystemVersion + "/stack/" + self.Uuid + "/export/"
	request := "GET"
	//Empty Body Request
	body := []byte(`{}`)

	data, err := DockerCloudCall(url, request, body)
	if err != nil {
		return "", err
	}

	s := string(data)

	return s, nil
}

func CreateStack(createRequest StackCreateRequest) (Stack, error) {
	url := "app/" + appSubsystemVersion + "/stack/"
	request := "POST"
	var response Stack

	newStack, err := json.Marshal(createRequest)
	if err != nil {
		return response, err
	}

	log.Println(string(newStack))

	data, err := DockerCloudCall(url, request, newStack)
	if err != nil {
		return response, err
	}

	err = json.Unmarshal(data, &response)
	if err != nil {
		return response, err
	}

	return response, nil
}

func (self *Stack) Update(createRequest StackCreateRequest) error {

	url := "app/" + appSubsystemVersion + "/stack/" + self.Uuid + "/"
	request := "PATCH"

	updatedStack, err := json.Marshal(createRequest)
	if err != nil {
		return err
	}

	log.Println(string(updatedStack))

	_, errr := DockerCloudCall(url, request, updatedStack)
	if errr != nil {
		return errr
	}

	return nil
}

func (self *Stack) Start() error {

	url := "app/" + appSubsystemVersion + "/stack/" + self.Uuid + "/start/"
	request := "POST"
	//Empty Body Request
	body := []byte(`{}`)

	_, err := DockerCloudCall(url, request, body)
	if err != nil {
		return err
	}

	return nil
}

func (self *Stack) Stop() error {

	url := "app/" + appSubsystemVersion + "/stack/" + self.Uuid + "/stop/"
	request := "POST"
	//Empty Body Request
	body := []byte(`{}`)

	_, err := DockerCloudCall(url, request, body)
	if err != nil {
		return err
	}

	return nil
}

func (self *Stack) Redeploy(reuse_volume ReuseVolumesOption) error {

	url := ""
	if reuse_volume.Reuse != true {
		url = "app/" + appSubsystemVersion + "/stack/" + self.Uuid + "/redeploy/?reuse_volumes=false"
	} else {
		url = "app/" + appSubsystemVersion + "/stack/" + self.Uuid + "/redeploy/"
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

func (self *Stack) Terminate() error {

	url := "app/" + appSubsystemVersion + "/stack/" + self.Uuid + "/"
	request := "DELETE"
	//Empty Body Request
	body := []byte(`{}`)

	_, err := DockerCloudCall(url, request, body)
	if err != nil {
		return err
	}

	return nil
}
