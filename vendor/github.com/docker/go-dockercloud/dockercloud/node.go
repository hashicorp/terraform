package dockercloud

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

func ListNodes() (NodeListResponse, error) {

	url := "infra/" + infraSubsytemVersion + "/node/"
	request := "GET"

	//Empty Body Request
	body := []byte(`{}`)
	var response NodeListResponse
	var finalResponse NodeListResponse

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
			var nextResponse NodeListResponse
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

func GetNode(uuid string) (Node, error) {

	url := ""
	if string(uuid[0]) == "/" {
		url = uuid[5:]
	} else {
		url = "infra/" + infraSubsytemVersion + "/node/" + uuid + "/"
	}
	request := "GET"
	body := []byte(`{}`)
	var response Node

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

func (self *Node) Update(createRequest Node) error {

	url := "infra/" + infraSubsytemVersion + "/node/" + self.Uuid + "/"
	request := "PATCH"

	updatedNode, err := json.Marshal(createRequest)
	if err != nil {
		return err
	}

	_, errr := DockerCloudCall(url, request, updatedNode)
	if err != nil {
		return errr
	}

	return nil
}

func (self *Node) Upgrade() error {

	url := "infra/" + infraSubsytemVersion + "/node/" + self.Uuid + "/docker-upgrade/"
	request := "POST"
	//Empty Body Request
	body := []byte(`{}`)

	_, err := DockerCloudCall(url, request, body)
	if err != nil {
		return err
	}

	return nil
}

func (self *Node) Terminate() error {

	url := "infra/" + infraSubsytemVersion + "/node/" + self.Uuid + "/"
	request := "DELETE"
	//Empty Body Request
	body := []byte(`{}`)

	_, err := DockerCloudCall(url, request, body)
	if err != nil {
		return err
	}

	return nil
}

func (self *Node) Events(c chan NodeEvent) {
	endpoint := "infra/" + infraSubsytemVersion + "/node/" + self.Uuid + "/events/?user=" + User + "&token=" + ApiKey
	url := StreamUrl + endpoint

	header := http.Header{}
	header.Add("User-Agent", customUserAgent)

	var Dialer websocket.Dialer
	ws, _, err := Dialer.Dial(url, header)
	if err != nil {
		log.Println(err)
	}

	var msg NodeEvent
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
