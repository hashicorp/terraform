package openstack

import (
	"log"

	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
	"github.com/rackspace/gophercloud"
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

	serversApi, err := gophercloud.ServersApi(client.AccessProvider, gophercloud.ApiCriteria{
		Name:      "nova",
		UrlChoice: gophercloud.PublicURL,
	})
	if err != nil {
		return nil, err
	}

	newServer, err := serversApi.CreateServer(gophercloud.NewServer{
		Name:      "12345",
		ImageRef:  rs.Attributes["imageRef"],
		FlavorRef: rs.Attributes["flavorRef"],
	})

	if err != nil {
		return nil, err
	}

	rs.Attributes["id"] = newServer.Id
	rs.Attributes["name"] = newServer.Name
	rs.Attributes["imageRef"] = newServer.ImageRef
	rs.Attributes["flavorRef"] = newServer.FlavorRef

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
			"flavorRef": diff.AttrTypeUpdate,
			"name":      diff.AttrTypeUpdate,
		},

		ComputedAttrs: []string{
			"name",
			"id",
		},

		ComputedAttrsUpdate: []string{},
	}

	return b.Diff(s, c)
}
