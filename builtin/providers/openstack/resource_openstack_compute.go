package openstack

import (
	"crypto/rand"
	"time"

	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
)

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

	name := rs.Attributes["name"]
	if len(name) == 0 {
		name = randomString(16)
	}

	var osNetworks []gophercloud.NetworkConfig

	if raw := flatmap.Expand(rs.Attributes, "networks"); raw != nil {
		if entries, ok := raw.([]interface{}); ok {
			for _, entry := range entries {
				value, ok := entry.(string)
				if !ok {
					continue
				}

				osNetwork := gophercloud.NetworkConfig{value}
				osNetworks = append(osNetworks, osNetwork)
			}
		}
	}

	var securityGroup []map[string]interface{}

	if raw := flatmap.Expand(rs.Attributes, "security_groups"); raw != nil {
		if entries, ok := raw.([]interface{}); ok {
			for _, entry := range entries {
				value, ok := entry.(string)
				if !ok {
					continue
				}

				securityGroup = append(securityGroup, map[string]interface{}{"name": value})
			}
		}
	}

	newServer, err := serversApi.CreateServer(gophercloud.NewServer{
		Name:          name,
		ImageRef:      rs.Attributes["image_ref"],
		FlavorRef:     rs.Attributes["flavor_ref"],
		Networks:      osNetworks,
		SecurityGroup: securityGroup,
	})

	if err != nil {
		return nil, err
	}

	rs.ID = newServer.Id
	rs.Attributes["id"] = newServer.Id
	rs.Attributes["name"] = name
	rs.Attributes["admin_pass"] = newServer.AdminPass

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"BUILD", "BUILDING"},
		Target:     "ACTIVE",
		Refresh:    WaitForServerState(serversApi, rs.Attributes["id"]),
		Timeout:    15 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()

	if err != nil {
		return nil, err
	}

	if pool, ok := rs.Attributes["floating_ip_pool"]; ok {
		newIp, err := serversApi.CreateFloatingIp(pool)
		if err != nil {
			return nil, err
		}

		err = serversApi.AssociateFloatingIp(newServer.Id, newIp)
		if err != nil {
			return nil, err
		}

		rs.Attributes["floating_ip"] = newIp.Ip
	}

	// TODO fill rs.Conn

	return rs, nil
}

func resource_openstack_compute_update(
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

	if attr, ok := d.Attributes["name"]; ok {
		_, err := serversApi.UpdateServer(rs.ID, gophercloud.NewServerSettings{
			Name: attr.New,
		})

		if err != nil {
			return nil, err
		}

		rs.Attributes["name"] = attr.New
	}

	if attr, ok := d.Attributes["flavor_ref"]; ok {
		err := serversApi.ResizeServer(rs.Attributes["id"], rs.Attributes["name"], attr.New, "")

		if err != nil {
			return nil, err
		}

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"ACTIVE", "RESIZE"},
			Target:     "VERIFY_RESIZE",
			Refresh:    WaitForServerState(serversApi, rs.Attributes["id"]),
			Timeout:    10 * time.Minute,
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, err = stateConf.WaitForState()

		if err != nil {
			return nil, err
		}

		err = serversApi.ConfirmResize(rs.Attributes["id"])
	}

	return rs, nil
}

func resource_openstack_compute_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {

	p := meta.(*ResourceProvider)
	client := p.client

	serversApi, err := gophercloud.ServersApi(client.AccessProvider, gophercloud.ApiCriteria{
		Name:      "nova",
		UrlChoice: gophercloud.PublicURL,
	})
	if err != nil {
		return err
	}

	err = serversApi.DeleteServerById(s.ID)

	return err
}

func resource_openstack_compute_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {

	p := meta.(*ResourceProvider)
	client := p.client

	serversApi, err := gophercloud.ServersApi(client.AccessProvider, gophercloud.ApiCriteria{
		Name:      "nova",
		UrlChoice: gophercloud.PublicURL,
	})
	if err != nil {
		return nil, err
	}

	server, err := serversApi.ServerById(s.ID)
	if err != nil {
		httpError, ok := err.(*perigee.UnexpectedResponseCodeError)
		if !ok {
			return nil, err
		}

		if httpError.Actual == 404 {
			return nil, nil
		}

		return nil, err
	}

	// TODO check networks, seucrity groups and floating ip

	s.Attributes["name"] = server.Name
	s.Attributes["flavor_ref"] = server.Flavor.Id

	return s, nil
}

func resource_openstack_compute_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"image_ref":        diff.AttrTypeCreate,
			"flavor_ref":       diff.AttrTypeUpdate,
			"name":             diff.AttrTypeUpdate,
			"networks":         diff.AttrTypeCreate,
			"security_groups":  diff.AttrTypeCreate,
			"floating_ip_pool": diff.AttrTypeUpdate,
		},

		ComputedAttrs: []string{
			"name",
			"id",
			"floating_ip",
			"admin_pass",
		},
	}

	return b.Diff(s, c)
}

// randomString generates a string of given length, but random content.
// All content will be within the ASCII graphic character set.
// (Implementation from Even Shaw's contribution on
// http://stackoverflow.com/questions/12771930/what-is-the-fastest-way-to-generate-a-long-random-string-in-go).
func randomString(n int) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}

func WaitForServerState(api gophercloud.CloudServersProvider, id string) resource.StateRefreshFunc {

	return func() (interface{}, string, error) {
		s, err := api.ServerById(id)
		if err != nil {
			return nil, "", err
		}

		return s, s.Status, nil

	}
}
