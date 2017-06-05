package digitalocean

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDigitalOceanImage_Basic(t *testing.T) {
	var droplet godo.Droplet
	var snapshotsId []int
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanDropletDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDigitalOceanDropletConfig_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanDropletExists("digitalocean_droplet.foobar", &droplet),
					takeSnapshotsOfDroplet(rInt, &droplet, &snapshotsId),
				),
			},
			{
				Config: testAccCheckDigitalOceanImageConfig_basic(rInt, 1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.digitalocean_image.foobar", "name", fmt.Sprintf("snap-%d-1", rInt)),
					resource.TestCheckResourceAttr(
						"data.digitalocean_image.foobar", "min_disk_size", "20"),
					resource.TestCheckResourceAttr(
						"data.digitalocean_image.foobar", "private", "true"),
					resource.TestCheckResourceAttr(
						"data.digitalocean_image.foobar", "type", "snapshot"),
				),
			},
			{
				Config:      testAccCheckDigitalOceanImageConfig_basic(rInt, 0),
				ExpectError: regexp.MustCompile(`.*too many user images found with name snap-.*\ .found 2, expected 1.`),
			},
			{
				Config:      testAccCheckDigitalOceanImageConfig_nonexisting(rInt),
				Destroy:     false,
				ExpectError: regexp.MustCompile(`.*no user image found with name snap-.*-nonexisting`),
			},
			{
				Config: " ",
				Check: resource.ComposeTestCheckFunc(
					deleteSnapshots(&snapshotsId),
				),
			},
		},
	})
}

func takeSnapshotsOfDroplet(rInt int, droplet *godo.Droplet, snapshotsId *[]int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*godo.Client)
		for i := 0; i < 3; i++ {
			err := takeSnapshotOfDroplet(rInt, i%2, droplet)
			if err != nil {
				return err
			}
		}
		retrieveDroplet, _, err := client.Droplets.Get(context.Background(), (*droplet).ID)
		if err != nil {
			return err
		}
		*snapshotsId = retrieveDroplet.SnapshotIDs
		return nil
	}
}

func takeSnapshotOfDroplet(rInt, sInt int, droplet *godo.Droplet) error {
	client := testAccProvider.Meta().(*godo.Client)
	action, _, err := client.DropletActions.Snapshot(context.Background(), (*droplet).ID, fmt.Sprintf("snap-%d-%d", rInt, sInt))
	if err != nil {
		return err
	}
	waitForAction(client, action)
	return nil
}

func deleteSnapshots(snapshotsId *[]int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		log.Printf("XXX Deleting snaps")
		client := testAccProvider.Meta().(*godo.Client)
		snapshots := *snapshotsId
		for _, value := range snapshots {
			log.Printf("XXX Deleting %d", value)
			_, err := client.Images.Delete(context.Background(), value)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func testAccCheckDigitalOceanImageConfig_basic(rInt, sInt int) string {
	return fmt.Sprintf(`
data "digitalocean_image" "foobar" {
  name               = "snap-%d-%d"
}
`, rInt, sInt)
}

func testAccCheckDigitalOceanImageConfig_nonexisting(rInt int) string {
	return fmt.Sprintf(`
data "digitalocean_image" "foobar" {
  name               = "snap-%d-nonexisting"
}
`, rInt)
}
