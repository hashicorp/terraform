package chef

import (
	"fmt"
	"time"
)

// SandboxService is the chef-client Sandbox service used as the entrypoint and caller for Sandbox methods
type SandboxService struct {
	client *Client
}

// SandboxRequest is the desired chef-api structure for a Post body
type SandboxRequest struct {
	Checksums map[string]interface{} `json:"checksums"`
}

// SandboxPostResponse is the struct returned from the chef-server for Post Requests to /sandbox
type SandboxPostResponse struct {
	ID        string `json:"sandbox_id"`
	Uri       string `json:"uri"`
	Checksums map[string]SandboxItem
}

// A SandbooxItem is embeddedinto  the response from the chef-server and the actual sandbox It is the Url and state for a specific Item.
type SandboxItem struct {
	Url    string `json:"url"`
	Upload bool   `json:"needs_upload"`
}

// Sandbox Is the structure of an actul sandbox that has been created and returned by the final PUT to the sandbox ID
type Sandbox struct {
	ID           string    `json:"guid"`
	Name         string    `json:"name"`
	CreationTime time.Time `json:"create_time"`
	Completed    bool      `json:"is_completed"`
	Checksums    []string
}

// Post creates a new sandbox on the chef-server. Deviates from the Chef-server api in that it takes a []string of sums for the sandbox instead of the IMO rediculous hash of nulls that the API wants. We convert it to the right structure under the hood for the chef-server api.
// http://docs.getchef.com/api_chef_server.html#id38
func (s SandboxService) Post(sums []string) (data SandboxPostResponse, err error) {
	smap := make(map[string]interface{})
	for _, hashstr := range sums {
		smap[hashstr] = nil
	}
	sboxReq := SandboxRequest{Checksums: smap}

	body, err := JSONReader(sboxReq)
	if err != nil {
		return
	}

	err = s.client.magicRequestDecoder("POST", "/sandboxes", body, &data)
	return
}

// Put is used to commit a sandbox ID to the chef server. To singal that the sandox you have Posted is now uploaded.
func (s SandboxService) Put(id string) (box Sandbox, err error) {
	answer := make(map[string]bool)
	answer["is_completed"] = true
	body, err := JSONReader(answer)

	if id == "" {
		return box, fmt.Errorf("must supply sandbox id to PUT request.")
	}

	err = s.client.magicRequestDecoder("PUT", "/sandboxes/"+id, body, &box)
	return
}
