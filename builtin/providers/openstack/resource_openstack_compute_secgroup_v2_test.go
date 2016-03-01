package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/secgroups"
)

func TestAccComputeV2SecGroup_basic(t *testing.T) {
	var secgroup secgroups.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2SecGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2SecGroup_basic_orig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2SecGroupExists(t, "openstack_compute_secgroup_v2.foo", &secgroup),
				),
			},
		},
	})
}

func TestAccComputeV2SecGroup_update(t *testing.T) {
	var secgroup secgroups.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2SecGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2SecGroup_basic_orig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2SecGroupExists(t, "openstack_compute_secgroup_v2.foo", &secgroup),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2SecGroup_basic_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2SecGroupExists(t, "openstack_compute_secgroup_v2.foo", &secgroup),
					testAccCheckComputeV2SecGroupRuleCount(t, &secgroup, 2),
				),
			},
		},
	})
}

func TestAccComputeV2SecGroup_groupID(t *testing.T) {
	var secgroup1, secgroup2, secgroup3 secgroups.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2SecGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2SecGroup_groupID_orig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2SecGroupExists(t, "openstack_compute_secgroup_v2.test_group_1", &secgroup1),
					testAccCheckComputeV2SecGroupExists(t, "openstack_compute_secgroup_v2.test_group_2", &secgroup2),
					testAccCheckComputeV2SecGroupExists(t, "openstack_compute_secgroup_v2.test_group_3", &secgroup3),
					testAccCheckComputeV2SecGroupGroupIDMatch(t, &secgroup1, &secgroup3),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2SecGroup_groupID_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2SecGroupExists(t, "openstack_compute_secgroup_v2.test_group_1", &secgroup1),
					testAccCheckComputeV2SecGroupExists(t, "openstack_compute_secgroup_v2.test_group_2", &secgroup2),
					testAccCheckComputeV2SecGroupExists(t, "openstack_compute_secgroup_v2.test_group_3", &secgroup3),
					testAccCheckComputeV2SecGroupGroupIDMatch(t, &secgroup2, &secgroup3),
				),
			},
		},
	})
}

func TestAccComputeV2SecGroup_self(t *testing.T) {
	var secgroup secgroups.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2SecGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2SecGroup_self,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2SecGroupExists(t, "openstack_compute_secgroup_v2.test_group_1", &secgroup),
					testAccCheckComputeV2SecGroupGroupIDMatch(t, &secgroup, &secgroup),
					resource.TestCheckResourceAttr(
						"openstack_compute_secgroup_v2.test_group_1", "rule.3170486100.self", "true"),
					resource.TestCheckResourceAttr(
						"openstack_compute_secgroup_v2.test_group_1", "rule.3170486100.from_group_id", ""),
				),
			},
		},
	})
}

func TestAccComputeV2SecGroup_icmpZero(t *testing.T) {
	var secgroup secgroups.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2SecGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2SecGroup_icmpZero,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2SecGroupExists(t, "openstack_compute_secgroup_v2.test_group_1", &secgroup),
				),
			},
		},
	})
}

func testAccCheckComputeV2SecGroupDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	computeClient, err := config.computeV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckComputeV2SecGroupDestroy) Error creating OpenStack compute client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_compute_secgroup_v2" {
			continue
		}

		_, err := secgroups.Get(computeClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Security group still exists")
		}
	}

	return nil
}

func testAccCheckComputeV2SecGroupExists(t *testing.T, n string, secgroup *secgroups.SecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)
		computeClient, err := config.computeV2Client(OS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("(testAccCheckComputeV2SecGroupExists) Error creating OpenStack compute client: %s", err)
		}

		found, err := secgroups.Get(computeClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Security group not found")
		}

		*secgroup = *found

		return nil
	}
}

func testAccCheckComputeV2SecGroupRuleCount(t *testing.T, secgroup *secgroups.SecurityGroup, count int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(secgroup.Rules) != count {
			return fmt.Errorf("Security group rule count does not match. Expected %d, got %d", count, len(secgroup.Rules))
		}

		return nil
	}
}

func testAccCheckComputeV2SecGroupGroupIDMatch(t *testing.T, sg1, sg2 *secgroups.SecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(sg2.Rules) == 1 {
			if sg1.Name != sg2.Rules[0].Group.Name || sg1.TenantID != sg2.Rules[0].Group.TenantID {
				return fmt.Errorf("%s was not correctly applied to %s", sg1.Name, sg2.Name)
			}
		} else {
			return fmt.Errorf("%s rule count is incorrect", sg2.Name)
		}

		return nil
	}
}

var testAccComputeV2SecGroup_basic_orig = fmt.Sprintf(`
	resource "openstack_compute_secgroup_v2" "foo" {
		name = "test_group_1"
		description = "first test security group"
		rule {
			from_port = 22
			to_port = 22
			ip_protocol = "tcp"
			cidr = "0.0.0.0/0"
		}
		rule {
			from_port = 1
			to_port = 65535
			ip_protocol = "udp"
			cidr = "0.0.0.0/0"
		}
		rule {
			from_port = -1
			to_port = -1
			ip_protocol = "icmp"
			cidr = "0.0.0.0/0"
		}
	}`)

var testAccComputeV2SecGroup_basic_update = fmt.Sprintf(`
	resource "openstack_compute_secgroup_v2" "foo" {
		name = "test_group_1"
		description = "first test security group"
		rule {
			from_port = 2200
			to_port = 2200
			ip_protocol = "tcp"
			cidr = "0.0.0.0/0"
		}
		rule {
			from_port = -1
			to_port = -1
			ip_protocol = "icmp"
			cidr = "0.0.0.0/0"
		}
}`)

var testAccComputeV2SecGroup_groupID_orig = fmt.Sprintf(`
	resource "openstack_compute_secgroup_v2" "test_group_1" {
		name = "test_group_1"
		description = "first test security group"
		rule {
			from_port = 22
			to_port = 22
			ip_protocol = "tcp"
			cidr = "0.0.0.0/0"
		}
	}

	resource "openstack_compute_secgroup_v2" "test_group_2" {
		name = "test_group_2"
		description = "second test security group"
		rule {
			from_port = -1
			to_port = -1
			ip_protocol = "icmp"
			cidr = "0.0.0.0/0"
		}
	}

	resource "openstack_compute_secgroup_v2" "test_group_3" {
		name = "test_group_3"
		description = "third test security group"
		rule {
			from_port = 80
			to_port = 80
			ip_protocol = "tcp"
			from_group_id = "${openstack_compute_secgroup_v2.test_group_1.id}"
		}
	}`)

var testAccComputeV2SecGroup_groupID_update = fmt.Sprintf(`
	resource "openstack_compute_secgroup_v2" "test_group_1" {
		name = "test_group_1"
		description = "first test security group"
		rule {
			from_port = 22
			to_port = 22
			ip_protocol = "tcp"
			cidr = "0.0.0.0/0"
		}
	}

	resource "openstack_compute_secgroup_v2" "test_group_2" {
		name = "test_group_2"
		description = "second test security group"
		rule {
			from_port = -1
			to_port = -1
			ip_protocol = "icmp"
			cidr = "0.0.0.0/0"
		}
	}

	resource "openstack_compute_secgroup_v2" "test_group_3" {
		name = "test_group_3"
		description = "third test security group"
		rule {
			from_port = 80
			to_port = 80
			ip_protocol = "tcp"
			from_group_id = "${openstack_compute_secgroup_v2.test_group_2.id}"
		}
	}`)

var testAccComputeV2SecGroup_self = fmt.Sprintf(`
	resource "openstack_compute_secgroup_v2" "test_group_1" {
		name = "test_group_1"
		description = "first test security group"
		rule {
			from_port = 22
			to_port = 22
			ip_protocol = "tcp"
			self = true
		}
	}`)

var testAccComputeV2SecGroup_icmpZero = fmt.Sprintf(`
	resource "openstack_compute_secgroup_v2" "test_group_1" {
		name = "test_group_1"
		description = "first test security group"
		rule {
			from_port = 0
			to_port = 0
			ip_protocol = "icmp"
			cidr = "0.0.0.0/0"
		}
	}`)
