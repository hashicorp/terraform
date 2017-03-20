package digitalocean

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDigitalOceanSnapshotDataSource_droplet(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDigitalOceanDropletSnapshotDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanSnapshotDataSourceID("data.digitalocean_snapshot.droplet_snapshot"),
					resource.TestCheckResourceAttr("data.digitalocean_snapshot.droplet_snapshot", "name", "web-01"),
					resource.TestCheckResourceAttr("data.digitalocean_snapshot.droplet_snapshot", "created_at", "2017-03-01T04:04:58Z"),
					resource.TestCheckResourceAttr("data.digitalocean_snapshot.droplet_snapshot", "min_disk_size", "30"),
					resource.TestCheckResourceAttr("data.digitalocean_snapshot.droplet_snapshot", "size_gigabytes", "2.69"),
					resource.TestCheckResourceAttr("data.digitalocean_snapshot.droplet_snapshot", "resource_id", "35539772"),
					resource.TestCheckResourceAttr("data.digitalocean_snapshot.droplet_snapshot", "regions.0", "nyc3"),
				),
			},
		},
	})
}

func TestAccDigitalOceanSnapshotDataSource_volume(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDigitalOceanVolumeSnapshotDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanSnapshotDataSourceID("data.digitalocean_snapshot.volume_snapshot"),
					resource.TestCheckResourceAttr("data.digitalocean_snapshot.volume_snapshot", "name", "volume-01"),
					resource.TestCheckResourceAttr("data.digitalocean_snapshot.volume_snapshot", "created_at", "2017-03-02T04:04:58Z"),
					resource.TestCheckResourceAttr("data.digitalocean_snapshot.volume_snapshot", "min_disk_size", "100"),
					resource.TestCheckResourceAttr("data.digitalocean_snapshot.volume_snapshot", "size_gigabytes", "1"),
					resource.TestCheckResourceAttr("data.digitalocean_snapshot.volume_snapshot", "resource_id", "a3feba45-914f-11e6-bd40-000f53315820"),
					resource.TestCheckResourceAttr("data.digitalocean_snapshot.volume_snapshot", "regions.0", "nyc3"),
				),
			},
		},
	})
}

func TestResourceValidateNameRegex(t *testing.T) {
	type testCases struct {
		Value    string
		ErrCount int
	}

	invalidCases := []testCases{
		{
			Value:    `\`,
			ErrCount: 1,
		},
		{
			Value:    `**`,
			ErrCount: 1,
		},
		{
			Value:    `(.+`,
			ErrCount: 1,
		},
	}

	for _, tc := range invalidCases {
		_, errors := validateNameRegex(tc.Value, "name_regex")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q to trigger a validation error.", tc.Value)
		}
	}

	validCases := []testCases{
		{
			Value:    `\/`,
			ErrCount: 0,
		},
		{
			Value:    `.*`,
			ErrCount: 0,
		},
		{
			Value:    `\b(?:\d{1,3}\.){3}\d{1,3}\b`,
			ErrCount: 0,
		},
	}

	for _, tc := range validCases {
		_, errors := validateNameRegex(tc.Value, "name_regex")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q not to trigger a validation error.", tc.Value)
		}
	}
}

func testAccCheckDigitalOceanSnapshotDataSourceDestroy(s *terraform.State) error {
	return nil
}

func testAccCheckDigitalOceanSnapshotDataSourceID(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find Snapshot data source: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Snapshot data source ID not set")
		}
		return nil
	}
}

const testAccCheckDigitalOceanDropletSnapshotDataSourceConfig = `
data "digitalocean_snapshot" "droplet_snapshot" {
    most_recent = true
    resource_type = "droplet"
    name_regex = "^web"
}
`

const testAccCheckDigitalOceanVolumeSnapshotDataSourceConfig = `
data "digitalocean_snapshot" "volume_snapshot" {
    most_recent = true
    resource_type = "volume"
    name_regex = "^volume"
}
`
