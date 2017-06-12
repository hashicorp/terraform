package runscope

import (
	"fmt"
	"time"
)


// Integration represents an integration with a third-party. See https://www.runscope.com/docs/api/integrations
type Integration struct {
	ID              string `json:"id"`
	UUID            string `json:"uuid"`
	IntegrationType string `json:"type"`
	Description     string `json:"description,omitempty"`
}

// People represents a person belonging to a team. See https://www.runscope.com/docs/api/teams
type People struct {
	ID		    string    `json:"id"`
	UUID		string    `json:"uuid"`
	Name		string    `json:"name"`
	Email		string	  `json:"email"`
	CreatedAt	time.Time `json:"created_at"`
	LastLoginAt	time.Time `json:"last_login_at"`
	GroupName	string    `json:"group_name"`
}

// ListIntegrations list all configured integrations for your team. See https://www.runscope.com/docs/api/integrations
func (client *Client) ListIntegrations(teamID string) ([]*Integration, error) {
	resource, error := client.readResource("integration", teamID,
		fmt.Sprintf("/teams/%s/integrations", teamID))
	if error != nil {
		return nil, error
	}

	integration, error := getIntegrationFromResponse(resource.Data)
	if error != nil {
		return nil, error
	}

	return integration, nil
}

// ListPeople list all the people on your team. See https://www.runscope.com/docs/api/teams
func (client *Client) ListPeople(teamID string) ([]*People, error) {
	resource, error := client.readResource("people", teamID,
		fmt.Sprintf("/teams/%s/people", teamID))
	if error != nil {
		return nil, error
	}

	people, error := getPeopleFromResponse(resource.Data)
	if error != nil {
		return nil, error
	}

	return people, nil
}

func choose(items []*Integration,test func(*Integration) bool) (result []*Integration) {
	for _, item := range items {
		if test(item) {
			result = append(result, item)
		}
	}

	return
}

func getIntegrationFromResponse(response interface{}) ([]*Integration, error) {
	var integrations []*Integration
	err := decode(&integrations, response)
	return integrations, err
}

func getPeopleFromResponse(response interface{}) ([]*People, error) {
	var people []*People
	err := decode(&people, response)
	return people, err
}
