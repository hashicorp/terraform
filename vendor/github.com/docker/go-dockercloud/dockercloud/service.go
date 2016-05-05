package dockercloud

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

func ListServices() (SListResponse, error) {
	url := "app/" + appSubsystemVersion + "/service/"
	request := "GET"
	//Empty Body Request
	body := []byte(`{}`)
	var response SListResponse
	var finalResponse SListResponse

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
			var nextResponse SListResponse
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

func GetService(uuid string) (Service, error) {

	url := ""
	if string(uuid[0]) == "/" {
		url = uuid[5:]
	} else {
		url = "app/" + appSubsystemVersion + "/service/" + uuid + "/"
	}

	request := "GET"
	//Empty Body Request
	body := []byte(`{}`)
	var response Service

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

func CreateService(createRequest ServiceCreateRequest) (Service, error) {

	url := "app/" + appSubsystemVersion + "/service/"
	request := "POST"
	var response Service

	newService, err := json.Marshal(createRequest)
	if err != nil {
		return response, err
	}

	data, err := DockerCloudCall(url, request, newService)
	if err != nil {
		return response, err
	}

	err = json.Unmarshal(data, &response)
	if err != nil {
		return response, err
	}

	return response, nil
}

func (self *Service) Logs(c chan Logs) {

	endpoint := "api/app/" + appSubsystemVersion + "/service/" + self.Uuid + "/logs/"
	url := StreamUrl + endpoint

	header := http.Header{}
	header.Add("Authorization", AuthHeader)
	header.Add("User-Agent", customUserAgent)

	var Dialer websocket.Dialer
	ws, _, err := Dialer.Dial(url, header)
	if err != nil {
		log.Println(err)
	}

	var msg Logs
	for {
		if err = ws.ReadJSON(&msg); err != nil {
			log.Println(err)
			break
		}
		c <- msg
	}
}

func (self *Service) Scale() error {

	url := "app/" + appSubsystemVersion + "/service/" + self.Uuid + "/scale/"
	request := "POST"
	//Empty Body Request
	body := []byte(`{}`)
	var response Service

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

func (self *Service) Update(createRequest ServiceCreateRequest) error {
	url := "app/" + appSubsystemVersion + "/service/" + self.Uuid + "/"
	request := "PATCH"

	updatedService, err := json.Marshal(createRequest)
	if err != nil {
		return err
	}

	_, err = DockerCloudCall(url, request, updatedService)
	if err != nil {
		return err
	}

	return nil
}

func (self *Service) Start() error {
	url := "app/" + appSubsystemVersion + "/service/" + self.Uuid + "/start/"
	request := "POST"
	//Empty Body Request
	body := []byte(`{}`)
	_, err := DockerCloudCall(url, request, body)
	if err != nil {
		return err
	}

	return nil
}

func (self *Service) StopService() error {

	url := "app/" + appSubsystemVersion + "/service/" + self.Uuid + "/stop/"
	request := "POST"
	//Empty Body Request
	body := []byte(`{}`)
	_, err := DockerCloudCall(url, request, body)
	if err != nil {
		return err
	}

	return nil
}

func (self *Service) Redeploy(reuse_volume ReuseVolumesOption) error {

	url := ""
	if reuse_volume.Reuse != true {
		url = "app/" + appSubsystemVersion + "/service/" + self.Uuid + "/redeploy/?reuse_volumes=false"
	} else {
		url = "app/" + appSubsystemVersion + "/service/" + self.Uuid + "/redeploy/"
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

func (self *Service) TerminateService() error {
	url := "app/" + appSubsystemVersion + "/service/" + self.Uuid + "/"
	request := "DELETE"
	//Empty Body Request
	body := []byte(`{}`)

	_, err := DockerCloudCall(url, request, body)
	if err != nil {
		return err
	}

	return nil
}
