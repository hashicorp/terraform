package netapp

import (
	"fmt"
	"testing"

	"github.com/candidpartners/occm-sdk-go/api/workenv"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccCloudVolume_nfs_vsa_basic(t *testing.T) {
	var id *string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudVolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudVolume_nfs_vsa_with_aggregate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudVolumeExists("netapp_cloud_volume.vsa-nfs-volume", id),
					resource.TestCheckResourceAttrSet(
						"netapp_cloud_volume.vsa-nfs-volume", "workenv_id"),
					resource.TestCheckResourceAttrSet(
						"netapp_cloud_volume.vsa-nfs-volume", "svm_name"),
					resource.TestCheckResourceAttrSet(
						"netapp_cloud_volume.vsa-nfs-volume", "aggregate_name"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-nfs-volume", "name", "vsa_nfs_vol"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-nfs-volume", "type", "nfs"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-nfs-volume", "size", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-nfs-volume", "size_unit", "GB"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-nfs-volume", "snapshot_policy", "default"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-nfs-volume", "export_policy.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-nfs-volume", "export_policy.0", "10.11.12.13/32"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-nfs-volume", "thin_provisioning", "true"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-nfs-volume", "compression", "true"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-nfs-volume", "deduplication", "true"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-nfs-volume", "provider_volume_type", "gp2"),
				),
			},
		},
	})
}

func TestAccCloudVolume_nfs_vsa_data_change(t *testing.T) {
	var id *string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudVolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudVolume_nfs_vsa,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudVolumeExists("netapp_cloud_volume.vsa-nfs-volume", id),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-nfs-volume", "export_policy.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-nfs-volume", "export_policy.0", "10.11.12.13/32"),
				),
			},
			resource.TestStep{
				Config: testAccCloudVolume_nfs_vsa_data_change,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudVolumeExists("netapp_cloud_volume.vsa-nfs-volume", id),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-nfs-volume", "export_policy.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-nfs-volume", "export_policy.0", "20.11.12.13/32"),
				),
			},
		},
	})
}

func TestAccCloudVolume_nfs_vsa_tier_change(t *testing.T) {
	var id *string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudVolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudVolume_nfs_vsa,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudVolumeExists("netapp_cloud_volume.vsa-nfs-volume", id),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-nfs-volume", "name", "vsa_nfs_vol"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-nfs-volume", "provider_volume_type", "gp2"),
				),
			},
			resource.TestStep{
				Config: testAccCloudVolume_nfs_vsa_tier_change,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudVolumeExists("netapp_cloud_volume.vsa-nfs-volume", id),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-nfs-volume", "name", "vsa_nfs_vol"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-nfs-volume", "provider_volume_type", "st1"),
				),
			},
		},
	})
}

func TestAccCloudVolume_nfs_awsha_basic(t *testing.T) {
	var id *string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudVolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudVolume_nfs_awsha,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudVolumeExists("netapp_cloud_volume.awsha-nfs-volume", id),
					resource.TestCheckResourceAttrSet(
						"netapp_cloud_volume.awsha-nfs-volume", "workenv_id"),
					resource.TestCheckResourceAttrSet(
						"netapp_cloud_volume.awsha-nfs-volume", "svm_name"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-nfs-volume", "name", "awsha_nfs_vol"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-nfs-volume", "type", "nfs"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-nfs-volume", "size", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-nfs-volume", "size_unit", "GB"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-nfs-volume", "snapshot_policy", "default"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-nfs-volume", "export_policy.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-nfs-volume", "export_policy.0", "12.13.14.15/32"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-nfs-volume", "thin_provisioning", "true"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-nfs-volume", "compression", "false"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-nfs-volume", "deduplication", "false"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-nfs-volume", "provider_volume_type", "st1"),
				),
			},
		},
	})
}

func TestAccCloudVolume_nfs_awsha_tier_change(t *testing.T) {
	var id *string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudVolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudVolume_nfs_awsha,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudVolumeExists("netapp_cloud_volume.awsha-nfs-volume", id),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-nfs-volume", "name", "awsha_nfs_vol"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-nfs-volume", "provider_volume_type", "st1"),
				),
			},
			resource.TestStep{
				Config: testAccCloudVolume_nfs_awsha_tier_change,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudVolumeExists("netapp_cloud_volume.awsha-nfs-volume", id),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-nfs-volume", "name", "awsha_nfs_vol"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-nfs-volume", "provider_volume_type", "gp2"),
				),
			},
		},
	})
}

func TestAccCloudVolume_nfs_awsha_data_change(t *testing.T) {
	var id *string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudVolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudVolume_nfs_awsha,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudVolumeExists("netapp_cloud_volume.awsha-nfs-volume", id),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-nfs-volume", "export_policy.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-nfs-volume", "export_policy.0", "12.13.14.15/32"),
				),
			},
			resource.TestStep{
				Config: testAccCloudVolume_nfs_awsha_data_change,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudVolumeExists("netapp_cloud_volume.awsha-nfs-volume", id),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-nfs-volume", "export_policy.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-nfs-volume", "export_policy.0", "22.13.14.15/32"),
				),
			},
		},
	})
}

func TestAccCloudVolume_cifs_vsa_basic(t *testing.T) {
	var id *string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudVolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudVolume_cifs_vsa,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudVolumeExists("netapp_cloud_volume.vsa-cifs-volume", id),
					resource.TestCheckResourceAttrSet(
						"netapp_cloud_volume.vsa-cifs-volume", "workenv_id"),
					resource.TestCheckResourceAttrSet(
						"netapp_cloud_volume.vsa-cifs-volume", "svm_name"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "name", "vsa_cifs_vol"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "type", "cifs"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "size", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "size_unit", "GB"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "snapshot_policy", "default"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "share.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "share.0.name", "cifs_volume_share"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "share.0.permission.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "share.0.permission.0.type", "read"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "share.0.permission.0.users.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "share.0.permission.0.users.0", "Everyone"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "thin_provisioning", "true"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "compression", "true"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "deduplication", "true"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "provider_volume_type", "gp2"),
				),
			},
		},
	})
}

func TestAccCloudVolume_cifs_vsa_data_change(t *testing.T) {
	var id *string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudVolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudVolume_cifs_vsa,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudVolumeExists("netapp_cloud_volume.vsa-cifs-volume", id),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "share.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "share.0.name", "cifs_volume_share"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "share.0.permission.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "share.0.permission.0.type", "read"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "share.0.permission.0.users.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "share.0.permission.0.users.0", "Everyone"),
				),
			},
			resource.TestStep{
				Config: testAccCloudVolume_cifs_vsa_data_change,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudVolumeExists("netapp_cloud_volume.vsa-cifs-volume", id),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "share.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "share.0.name", "cifs_volume_share"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "share.0.permission.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "share.0.permission.0.type", "full_control"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "share.0.permission.0.users.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.vsa-cifs-volume", "share.0.permission.0.users.0", "Administrator"),
				),
			},
		},
	})
}

func TestAccCloudVolume_cifs_awsha_basic(t *testing.T) {
	var id *string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudVolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudVolume_cifs_awsha,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudVolumeExists("netapp_cloud_volume.awsha-cifs-volume", id),
					resource.TestCheckResourceAttrSet(
						"netapp_cloud_volume.awsha-cifs-volume", "workenv_id"),
					resource.TestCheckResourceAttrSet(
						"netapp_cloud_volume.awsha-cifs-volume", "svm_name"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "name", "awsha_cifs_vol"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "type", "cifs"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "size", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "size_unit", "GB"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "snapshot_policy", "default"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "share.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "share.0.name", "cifs_volume_share"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "share.0.permission.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "share.0.permission.0.type", "read"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "share.0.permission.0.users.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "share.0.permission.0.users.0", "Everyone"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "thin_provisioning", "true"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "compression", "true"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "deduplication", "true"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "provider_volume_type", "gp2"),
				),
			},
		},
	})
}

func TestAccCloudVolume_cifs_awsha_data_change(t *testing.T) {
	var id *string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudVolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudVolume_cifs_awsha,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudVolumeExists("netapp_cloud_volume.awsha-cifs-volume", id),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "share.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "share.0.name", "cifs_volume_share"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "share.0.permission.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "share.0.permission.0.type", "read"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "share.0.permission.0.users.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "share.0.permission.0.users.0", "Everyone"),
				),
			},
			resource.TestStep{
				Config: testAccCloudVolume_cifs_awsha_data_change,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudVolumeExists("netapp_cloud_volume.awsha-cifs-volume", id),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "share.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "share.0.name", "cifs_volume_share"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "share.0.permission.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "share.0.permission.0.type", "full_control"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "share.0.permission.0.users.#", "1"),
					resource.TestCheckResourceAttr(
						"netapp_cloud_volume.awsha-cifs-volume", "share.0.permission.0.users.0", "Administrator"),
				),
			},
		},
	})
}

func testAccCheckCloudVolumeDestroy(s *terraform.State) error {
	apis := testAccProvider.Meta().(*APIs)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netapp_cloud_volume" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		_, err := getCloudVolume(apis, rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Volume for ID %s still exists", rs.Primary.ID)
		}
	}
	return nil
}

func testAccCheckCloudVolumeExists(n string, id *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		apis := testAccProvider.Meta().(*APIs)

		_, err := getCloudVolume(apis, rs.Primary.ID)
		if err != nil {
			return err
		}

		id = &rs.Primary.ID

		return nil
	}
}

func getCloudVolume(apis *APIs, id string) (*workenv.VolumeResponse, error) {
	_, workenvId, _, volumeName, isHA, err := splitId(id)
	if err != nil {
		return nil, fmt.Errorf("Error splitting volume ID %s: %s", id, err)
	}

	var res *workenv.VolumeResponse
	if isHA {
		res, err = apis.AWSHAWorkingEnvironmentAPI.GetVolume(workenvId, volumeName)
	} else {
		res, err = apis.VSAWorkingEnvironmentAPI.GetVolume(workenvId, volumeName)
	}

	return res, err
}

const testAccCloudVolume_nfs_vsa_with_aggregate = `
data "netapp_cloud_workenv" "vsa-workenv" {
  name = "vsaenv"
}

resource "netapp_cloud_volume" "vsa-nfs-volume" {
  workenv_id = "${data.netapp_cloud_workenv.vsa-workenv.public_id}"
  svm_name = "${data.netapp_cloud_workenv.vsa-workenv.svm_name}"
  aggregate_name = "aggr1"
  name = "vsa_nfs_vol"
  type = "nfs"
  size = 1
  size_unit = "GB"
  snapshot_policy = "default"
  export_policy = ["10.11.12.13/32"]
  thin_provisioning = true
  compression = true
  deduplication = true
}
`
const testAccCloudVolume_nfs_vsa = `
data "netapp_cloud_workenv" "vsa-workenv" {
  name = "vsaenv"
}

resource "netapp_cloud_volume" "vsa-nfs-volume" {
  workenv_id = "${data.netapp_cloud_workenv.vsa-workenv.public_id}"
  svm_name = "${data.netapp_cloud_workenv.vsa-workenv.svm_name}"
  name = "vsa_nfs_vol"
  type = "nfs"
  size = 1
  size_unit = "GB"
  snapshot_policy = "default"
  export_policy = ["10.11.12.13/32"]
  thin_provisioning = true
  compression = true
  deduplication = true
}
`
const testAccCloudVolume_nfs_vsa_tier_change = `
data "netapp_cloud_workenv" "vsa-workenv" {
  name = "vsaenv"
}

resource "netapp_cloud_volume" "vsa-nfs-volume" {
  workenv_id = "${data.netapp_cloud_workenv.vsa-workenv.public_id}"
  svm_name = "${data.netapp_cloud_workenv.vsa-workenv.svm_name}"
  name = "vsa_nfs_vol"
  type = "nfs"
  size = 1
  size_unit = "GB"
  snapshot_policy = "default"
  export_policy = ["10.11.12.13/32"]
  provider_volume_type = "st1"
  thin_provisioning = true
  compression = true
  deduplication = true
}
`
const testAccCloudVolume_nfs_vsa_data_change = `
data "netapp_cloud_workenv" "vsa-workenv" {
  name = "vsaenv"
}

resource "netapp_cloud_volume" "vsa-nfs-volume" {
  workenv_id = "${data.netapp_cloud_workenv.vsa-workenv.public_id}"
  svm_name = "${data.netapp_cloud_workenv.vsa-workenv.svm_name}"
  name = "vsa_nfs_vol"
  type = "nfs"
  size = 1
  size_unit = "GB"
  snapshot_policy = "default"
  export_policy = ["20.11.12.13/32"]
  thin_provisioning = true
  compression = true
  deduplication = true
}
`
const testAccCloudVolume_nfs_awsha = `
data "netapp_cloud_workenv" "awsha-workenv" {
  name = "awshaenv"
}

resource "netapp_cloud_volume" "awsha-nfs-volume" {
  workenv_id = "${data.netapp_cloud_workenv.awsha-workenv.public_id}"
  svm_name = "${data.netapp_cloud_workenv.awsha-workenv.svm_name}"
  name = "awsha_nfs_vol"
  type = "nfs"
  size = 1
  size_unit = "GB"
  snapshot_policy = "default"
  export_policy = ["12.13.14.15/32"]
  provider_volume_type = "st1"
  thin_provisioning = true
  compression = false
  deduplication = false
}
`
const testAccCloudVolume_nfs_awsha_tier_change = `
data "netapp_cloud_workenv" "awsha-workenv" {
  name = "awshaenv"
}

resource "netapp_cloud_volume" "awsha-nfs-volume" {
  workenv_id = "${data.netapp_cloud_workenv.awsha-workenv.public_id}"
  svm_name = "${data.netapp_cloud_workenv.awsha-workenv.svm_name}"
  name = "awsha_nfs_vol"
  type = "nfs"
  size = 1
  size_unit = "GB"
  snapshot_policy = "default"
  export_policy = ["12.13.14.15/32"]
  provider_volume_type = "gp2"
  thin_provisioning = true
  compression = false
  deduplication = false
}
`
const testAccCloudVolume_nfs_awsha_data_change = `
data "netapp_cloud_workenv" "awsha-workenv" {
  name = "awshaenv"
}

resource "netapp_cloud_volume" "awsha-nfs-volume" {
  workenv_id = "${data.netapp_cloud_workenv.awsha-workenv.public_id}"
  svm_name = "${data.netapp_cloud_workenv.awsha-workenv.svm_name}"
  name = "awsha_nfs_vol"
  type = "nfs"
  size = 1
  size_unit = "GB"
  snapshot_policy = "default"
  export_policy = ["22.13.14.15/32"]
  provider_volume_type = "st1"
  thin_provisioning = true
  compression = false
  deduplication = false
}
`
const testAccCloudVolume_cifs_vsa = `
data "netapp_cloud_workenv" "vsa-workenv" {
  name = "vsaenv"
}

resource "netapp_cloud_volume" "vsa-cifs-volume" {
  workenv_id = "${data.netapp_cloud_workenv.vsa-workenv.public_id}"
  svm_name = "${data.netapp_cloud_workenv.vsa-workenv.svm_name}"
  name = "vsa_cifs_vol"
  type = "cifs"
  size = 1
  size_unit = "GB"
  snapshot_policy = "default"
  share {
    name = "cifs_volume_share"
    permission {
      type = "read"
      users = ["Everyone"]
    }
  }
  thin_provisioning = true
  compression = true
  deduplication = true
}
`
const testAccCloudVolume_cifs_vsa_data_change = `
data "netapp_cloud_workenv" "vsa-workenv" {
  name = "vsaenv"
}

resource "netapp_cloud_volume" "vsa-cifs-volume" {
  workenv_id = "${data.netapp_cloud_workenv.vsa-workenv.public_id}"
  svm_name = "${data.netapp_cloud_workenv.vsa-workenv.svm_name}"
  name = "vsa_cifs_vol"
  type = "cifs"
  size = 1
  size_unit = "GB"
  snapshot_policy = "default"
  share {
    name = "cifs_volume_share"
    permission {
      type = "full_control"
      users = ["Administrator"]
    }
  }
  thin_provisioning = true
  compression = true
  deduplication = true
}
`
const testAccCloudVolume_cifs_awsha = `
data "netapp_cloud_workenv" "awsha-workenv" {
  name = "awshaenv"
}

resource "netapp_cloud_volume" "awsha-cifs-volume" {
  workenv_id = "${data.netapp_cloud_workenv.awsha-workenv.public_id}"
  svm_name = "${data.netapp_cloud_workenv.awsha-workenv.svm_name}"
  name = "awsha_cifs_vol"
  type = "cifs"
  size = 1
  size_unit = "GB"
  snapshot_policy = "default"
  share {
    name = "cifs_volume_share"
    permission {
      type = "read"
      users = ["Everyone"]
    }
  }
  thin_provisioning = true
  compression = true
  deduplication = true
}
`
const testAccCloudVolume_cifs_awsha_data_change = `
data "netapp_cloud_workenv" "awsha-workenv" {
  name = "awshaenv"
}

resource "netapp_cloud_volume" "awsha-cifs-volume" {
  workenv_id = "${data.netapp_cloud_workenv.awsha-workenv.public_id}"
  svm_name = "${data.netapp_cloud_workenv.awsha-workenv.svm_name}"
  name = "awsha_cifs_vol"
  type = "cifs"
  size = 1
  size_unit = "GB"
  snapshot_policy = "default"
  share {
    name = "cifs_volume_share"
    permission {
      type = "full_control"
      users = ["Administrator"]
    }
  }
  thin_provisioning = true
  compression = true
  deduplication = true
}
`
