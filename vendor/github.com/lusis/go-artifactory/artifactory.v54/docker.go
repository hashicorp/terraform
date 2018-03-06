package artifactory

import "encoding/json"

// DockerImages represents the list of docker images in a docker repo
type DockerImages struct {
	Repositories []string `json:"repositories,omitempty"`
}

// DockerImageTags represents the list of tags for an image in a docker repo
type DockerImageTags struct {
	Name string   `json:"name,omitempty"`
	Tags []string `json:"tags,omitempty"`
}

// DockerImagePromotion represents the image promotion payload we send to Artifactory
type DockerImagePromotion struct {
	TargetRepo             string `json:"targetRepo"`                       // The target repository for the move or copy
	DockerRepository       string `json:"dockerRepository"`                 // The docker repository name to promote
	TargetDockerRepository string `json:"targetDockerRepository,omitempty"` // An optional docker repository name, if null, will use the same name as 'dockerRepository'
	Tag                    string `json:"tag,omitempty"`                    // An optional tag name to promote, if null - the entire docker repository will be promoted. Available from v4.10.
	TargetTag              string `json:"targetTag,omitempty"`              // An optional target tag to assign the image after promotion, if null - will use the same tag
	Copy                   bool   `json:"copy,omitempty"`                   // An optional value to set whether to copy instead of move. Default: false
}

// GetDockerRepoImages returns the docker images in the named repo
func (c *Client) GetDockerRepoImages(key string, q map[string]string) ([]string, error) {
	var dat DockerImages

	d, err := c.Get("/api/docker/"+key+"/v2/_catalog", q)
	if err != nil {
		return dat.Repositories, err
	}

	err = json.Unmarshal(d, &dat)
	if err != nil {
		return dat.Repositories, err
	}

	return dat.Repositories, nil
}

// GetDockerRepoImageTags returns the docker images in the named repo
func (c *Client) GetDockerRepoImageTags(key, image string, q map[string]string) ([]string, error) {
	var dat DockerImageTags

	d, err := c.Get("/api/docker/"+key+"/v2/"+image+"/tags/list", q)
	if err != nil {
		return dat.Tags, err
	}

	err = json.Unmarshal(d, &dat)
	if err != nil {
		return dat.Tags, err
	}

	return dat.Tags, nil
}

// PromoteDockerImage promotes a Docker image from one repository to another
func (c *Client) PromoteDockerImage(key string, p DockerImagePromotion, q map[string]string) error {
	j, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = c.Post("/api/docker/"+key+"/v2/promote", j, q)
	return err
}
