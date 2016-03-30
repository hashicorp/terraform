package dockercloud

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

func ListContainers() (CListResponse, error) {

	url := "app/" + appSubsystemVersion + "/container/"
	request := "GET"
	//Empty Body Request
	body := []byte(`{}`)
	var response CListResponse
	var finalResponse CListResponse

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
			var nextResponse CListResponse
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

func GetContainer(uuid string) (Container, error) {

	url := ""
	if string(uuid[0]) == "/" {
		url = uuid[5:]
	} else {
		url = "app/" + appSubsystemVersion + "/container/" + uuid + "/"
	}

	request := "GET"
	//Empty Body Request
	body := []byte(`{}`)
	var response Container

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

func (self *Container) Logs(c chan Logs) {

	endpoint := "app/" + appSubsystemVersion + "/container/" + self.Uuid + "/logs/?user=" + User + "&token=" + ApiKey
	url := StreamUrl + endpoint

	header := http.Header{}
	header.Add("User-Agent", customUserAgent)

	var Dialer websocket.Dialer
	ws, _, err := Dialer.Dial(url, header)
	if err != nil {
		log.Println(err)
	}

	var msg Logs
	for {
		if err = ws.ReadJSON(&msg); err != nil {
			if err != nil && err.Error() != "EOF" {
				log.Println(err)
			} else {
				break
			}
		}
		c <- msg
	}
}

func (self *Container) Exec(command string, c chan Exec) {
	go self.Run(command, c)
Loop:
	for {
		select {
		case s := <-c:
			if s.Output != "EOF" {
				fmt.Printf("%s", s.Output)
			} else {
				break Loop
			}
		}
	}
}

func (self *Container) Run(command string, c chan Exec) {

	endpoint := "app/" + appSubsystemVersion + "/container/" + self.Uuid + "/exec/?user=" + User + "&token=" + ApiKey + "&command=" + url.QueryEscape(command)
	url := StreamUrl + endpoint

	header := http.Header{}
	header.Add("User-Agent", customUserAgent)

	var Dialer websocket.Dialer
	ws, _, err := Dialer.Dial(url, header)
	if err != nil {
		log.Println(err)
	}

	var msg Exec
	for {
		if err = ws.ReadJSON(&msg); err != nil {
			if err != nil && err.Error() != "EOF" {
				log.Println(err)
			} else {
				break
			}
		}
		c <- msg
	}
}

func (self *Container) Start() error {

	url := "app/" + appSubsystemVersion + "/container/" + self.Uuid + "/start/"
	request := "POST"
	//Empty Body Request
	body := []byte(`{}`)
	var response Container

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

func (self *Container) Stop() error {

	url := "app/" + appSubsystemVersion + "/container/" + self.Uuid + "/stop/"
	request := "POST"
	//Empty Body Request
	body := []byte(`{}`)

	_, err := DockerCloudCall(url, request, body)
	if err != nil {
		return err
	}

	return nil
}

func (self *Container) Redeploy(reuse_volume ReuseVolumesOption) error {

	url := ""
	if reuse_volume.Reuse != true {
		url = "app/" + appSubsystemVersion + "/container/" + self.Uuid + "/redeploy/?reuse_volumes=false"
	} else {
		url = "app/" + appSubsystemVersion + "/container/" + self.Uuid + "/redeploy/"
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

func (self *Container) Terminate() error {

	url := "app/" + appSubsystemVersion + "/container/" + self.Uuid + "/"
	request := "DELETE"
	//Empty Body Request
	body := []byte(`{}`)

	_, err := DockerCloudCall(url, request, body)
	if err != nil {
		return err
	}

	return nil
}
