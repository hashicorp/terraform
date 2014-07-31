package openstack

import (
	"log"

	"bytes"
	"encoding/json"
	"errors"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
	"net/http"
)

type server struct {
	Server serverContainer `json:"server"`
}

type serverContainer struct {
	Name      string `json:"name"`
	ImageRef  string `json:"imageRef"`
	FlavorRef string `json:"flavorRef"`
}

func resource_openstack_compute_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {

	p := meta.(*ResourceProvider)
	client := p.client

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	serverContainer := serverContainer{rs.Attributes["name"], rs.Attributes["imageRef"], rs.Attributes["flavorRef"]}
	server := server{serverContainer}

	body, err := json.Marshal(server)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", client.Config.ComputeEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header = http.Header{
		"Accept":       {"application/json"},
		"Content-Type": {"application/json"},
		"X-Auth-Token": {client.Token},
	}

	httpClient := &http.Client{}
	res, err := httpClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != 202 {
		return nil, errors.New("Creation failed: " + res.Status)
	}

	rs.Attributes["id"] = "1234"

	return rs, nil
}

func resource_openstack_compute_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {

	log.Printf("[INFO] update")

	return s, nil
}

func resource_openstack_compute_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {

	log.Printf("[INFO] destroy")

	return nil
}

func resource_openstack_compute_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {

	log.Printf("[INFO] refresh")

	return s, nil
}

func resource_openstack_compute_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	log.Printf("[INFO] diff")

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"imageRef":  diff.AttrTypeCreate,
			"flavorRef": diff.AttrTypeCreate,
			"name":      diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{
			"id",
		},

		ComputedAttrsUpdate: []string{},
	}

	return b.Diff(s, c)
}
