package dockercloud

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

func ListActions() (ActionListResponse, error) {

	url := "audit/" + auditSubsystemVersion + "/action/"

	request := "GET"
	//Empty Body Request
	body := []byte(`{}`)
	var response ActionListResponse
	var finalResponse ActionListResponse
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
			var nextResponse ActionListResponse
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

func GetAction(uuid string) (Action, error) {

	url := ""
	if string(uuid[0]) == "/" {
		url = uuid[5:]
	} else {
		url = "audit/" + auditSubsystemVersion + "/action/" + uuid + "/"
	}

	request := "GET"
	//Empty Body Request
	body := []byte(`{}`)
	var response Action

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

func (self *Action) GetLogs(c chan Logs) {

	endpoint := "audit/" + auditSubsystemVersion + "/action/" + self.Uuid + "/logs/?user=" + User + "&token=" + ApiKey

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

func (self *Action) Cancel() (Action, error) {
	url := ""
	if string(self.Uuid[0]) == "/" {
		url = self.Uuid[8:]
	} else {
		url = "audit/" + auditSubsystemVersion + "/action/" + self.Uuid + "/cancel/"
	}

	request := "POST"
	//Empty Body Request
	body := []byte(`{}`)
	var response Action

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

func (self *Action) Retry() (Action, error) {
	url := ""
	if string(self.Uuid[0]) == "/" {
		url = self.Uuid[8:]
	} else {
		url = "audit/" + auditSubsystemVersion + "/action/" + self.Uuid + "/retry/"
	}

	request := "POST"
	//Empty Body Request
	body := []byte(`{}`)
	var response Action

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
