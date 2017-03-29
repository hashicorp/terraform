package opc

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/oracle/terraform-provider-compute/sdk/compute"
	"testing"
)

func TestAccOPCInstance_Basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: opcResourceCheck(
			instanceResourceName,
			testAccCheckInstanceDestroyed),
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceBasic,
				Check: resource.ComposeTestCheckFunc(
					opcResourceCheck(
						instanceResourceName,
						testAccCheckInstanceExists),
					opcResourceCheck(
						keyResourceName,
						testAccCheckSSHKeyExists),
				),
			},
			{
				Config: modifySSHKey,
				Check: resource.ComposeTestCheckFunc(
					opcResourceCheck(
						instanceResourceName,
						testAccCheckInstanceExists),
					opcResourceCheck(
						keyResourceName,
						testAccCheckSSHKeyUpdated),
				),
			},
		},
	})
}

func testAccCheckInstanceExists(state *OPCResourceState) error {
	instanceName := getInstanceName(state)

	if _, err := state.Instances().GetInstance(instanceName); err != nil {
		return fmt.Errorf("Error retrieving state of instance %s: %s", instanceName, err)
	}

	return nil
}

func testAccCheckSSHKeyExists(state *OPCResourceState) error {
	keyName := state.Attributes["name"]

	if _, err := state.SSHKeys().GetSSHKey(keyName); err != nil {
		return fmt.Errorf("Error retrieving state of key %s: %s", keyName, err)
	}

	return nil
}

func testAccCheckSSHKeyUpdated(state *OPCResourceState) error {
	keyName := state.Attributes["name"]
	info, err := state.SSHKeys().GetSSHKey(keyName)
	if err != nil {
		return err
	}
	if info.Key != updatedKey {
		return fmt.Errorf("Expected key\n\t%s\nbut was\n\t%s", updatedKey, info.Key)
	}
	return nil
}

func getInstanceName(rs *OPCResourceState) *compute.InstanceName {
	return &compute.InstanceName{
		Name: rs.Attributes["name"],
		ID:   rs.Attributes["opcId"],
	}
}

func testAccCheckInstanceDestroyed(state *OPCResourceState) error {
	instanceName := getInstanceName(state)
	if info, err := state.Instances().GetInstance(instanceName); err == nil {
		return fmt.Errorf("Instance %s still exists: %#v", instanceName, info)
	}

	return nil
}

const instanceName = "test_instance"
const keyName = "test_key"

var instanceResourceName = fmt.Sprintf("opc_compute_instance.%s", instanceName)
var keyResourceName = fmt.Sprintf("opc_compute_ssh_key.%s", keyName)

const originalKey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCqw6JwbjIkZEr5UcMojtxhk6Zum39NOihHNXEvRWDt5WssX8TH/ghpv3D25K1pJkf+wfAi17HwEmYwPMEyEHENS443v6RZbXvzCkUWzkJzq7Zvbdqld038km31La2QUoMMp1KL5zk1nM65xCeQDVcR/h++03EScB2CuzTpAV6khMdfgOJgxm361kfrDVRwc1HQrAOpOnzkpPfwqBrYWqN1UnKvuO77Wk8z5LBe03EPNru3bLE3s3qHI9hjO0gXMiVUi0KyNxdWfDO8esqQlKavHAeePyrRA55YF8kBB5dEl4tVNOqpY/8TRnGN1mOe0LWxa8Ytz1wbyS49knsNVTel"
const updatedKey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDHvb/2OSemgzUYLNW1/T3u33r7sZy1qbWtgVWiREH4gS5TVmDVPuvN1MFLdNqiWQA53gK8Gp24jtjNm9ftcPhicv81HVWJTB69C0sJGEfF0l4mgbemJLH3i37Mb6SdWJcGof9qHVDADPgiC8jIBVUhdiJSeq4fUJ3NQA2eUExBkRglQWairkNzPNA0mi3GL9KDGnoBnSCAXNGoKgDgIOqW0dYFP6oHyGWkF7V+/TME9aIQvmMpHjVzl7brZ/wED2t5vTJxxbgogHEmWnfs7p8EP5IsN6Vnjd0VNIt1tu3TduS8kH5npkPqZz8oIP93Ypxn0l7ZNEl9MahbhPj3gJ1YY7Cygrlt1VLC1ibBbOgIS2Lj6vGG/Yjkqs3Vw6qrmTRlsJ9c6bZO2xq0xzV11XQHvjPegBOClF6AztEe1jKU/RUFnzjIF8lUmM63fTaXuVkNERkTSE3E9XL3Uq6eqYdef7wHFFhCMSGotp3ANAb30kflysA9ID0b3o5QU2tB8OBxBicXQy11lh+u204YJuvIzeTXo+JAad5TWFlJcsUlbPFppLQdhUpoWaJouBGJV36DJb9R34i9T8Ze5tnJUQgPmMkERyPvb/+v5j3s2hs1A9WO6/MqmZd70gudsX/1bqWT898vCCOdM+CspNVY7nHVUtde7C6BrHzphr/C1YBXHw=="

var testAccInstanceBasic = fmt.Sprintf(`
resource "opc_compute_instance" "%s" {
	name = "test"
	label = "test"
	shape = "oc3"
	imageList = "/oracle/public/oel_6.4_2GB_v1"
	sshKeys = ["${opc_compute_ssh_key.test_key.name}"]
	attributes = "{\"foo\": \"bar\"}"
	storage = {
		index = 1
		volume = "${opc_compute_storage_volume.test_volume.name}"
	}
}

resource "opc_compute_storage_volume" "test_volume" {
	size = "3g"
	description = "My volume"
	name = "test_volume_b"
	tags = ["foo", "bar", "baz"]
}

resource "opc_compute_ssh_key" "%s" {
	name = "test-key"
	key = "%s"
	enabled = true
}
`, instanceName, keyName, originalKey)

var modifySSHKey = fmt.Sprintf(`
resource "opc_compute_instance" "%s" {
	name = "test"
	label = "test"
	shape = "oc3"
	imageList = "/oracle/public/oel_6.4_2GB_v1"
	sshKeys = ["${opc_compute_ssh_key.test_key.name}"]
	attributes = "{\"foo\": \"bar\"}"
	storage = {
		index = 1
		volume = "${opc_compute_storage_volume.test_volume.name}"
	}
}

resource "opc_compute_storage_volume" "test_volume" {
	size = "3g"
	description = "My volume"
	name = "test_volume_b"
	tags = ["foo", "bar", "baz"]
}

resource "opc_compute_ssh_key" "%s" {
	name = "test-key"
	key = "%s"
	enabled = true
}
`, instanceName, keyName, updatedKey)
