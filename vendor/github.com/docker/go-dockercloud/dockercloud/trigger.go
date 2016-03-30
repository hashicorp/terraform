package dockercloud

import "encoding/json"

func (self *Service) ListTriggers() (TriggerListResponse, error) {
	url := "app/" + appSubsystemVersion + "/service/" + self.Uuid + "/trigger/"
	request := "GET"
	//Empty Body Request
	body := []byte(`{}`)
	var response TriggerListResponse
	var finalResponse TriggerListResponse

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
			var nextResponse TriggerListResponse
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

func (self *Service) GetTrigger(trigger_uuid string) (Trigger, error) {

	url := ""
	if string(trigger_uuid[0]) == "/" {
		url = trigger_uuid[5:]
	} else {
		url = "app/" + appSubsystemVersion + "/service/" + self.Uuid + "/trigger/" + trigger_uuid + "/"
	}

	request := "GET"
	body := []byte(`{}`)
	var response Trigger

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

func (self *Service) CreateTrigger(createRequest TriggerCreateRequest) (Trigger, error) {

	url := "app/" + appSubsystemVersion + "/service/" + self.Uuid + "/trigger/"
	request := "POST"
	var response Trigger

	newTrigger, err := json.Marshal(createRequest)
	if err != nil {
		return response, err
	}

	data, err := DockerCloudCall(url, request, newTrigger)
	if err != nil {
		return response, err
	}

	err = json.Unmarshal(data, &response)
	if err != nil {
		return response, err
	}

	return response, nil
}

func (self *Service) DeleteTrigger(trigger_uuid string) error {
	url := ""
	if string(trigger_uuid[0]) == "/" {
		url = trigger_uuid[8:]
	} else {
		url = "app/" + appSubsystemVersion + "/service/" + self.Uuid + "/trigger/" + trigger_uuid + "/"
	}

	request := "DELETE"
	body := []byte(`{}`)
	var response Trigger

	data, err := DockerCloudCall(url, request, body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &response)
	if err != nil {
		return err
	}

	return nil
}

func (self *Service) CallTrigger(trigger_uuid string) (Trigger, error) {
	url := ""
	if string(trigger_uuid[0]) == "/" {
		url = trigger_uuid[8:]
	} else {
		url = "app/" + appSubsystemVersion + "/service/" + self.Uuid + "/trigger/" + trigger_uuid + "/call/"
	}

	request := "POST"
	body := []byte(`{}`)
	var response Trigger

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
