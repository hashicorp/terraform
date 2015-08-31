package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/errors"
	client "k8s.io/kubernetes/pkg/client/unversioned"
)

func TestAccKubernetesPersistentVolume_basic(t *testing.T) {
	var conf api.PersistentVolume

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccKubernetesPersistentVolumeConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.bar", &conf),
				),
			},
		},
	})
}

func testAccCheckKubernetesPersistentVolumeDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*client.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_persistent_volume" {
			continue
		}

		pv, err := conn.PersistentVolumes().Get(rs.Primary.Attributes["name"])
		if err == nil {
			if string(pv.UID) == rs.Primary.ID {
				return fmt.Errorf("Persistent Volume still exists")
			}
		}

		se, ok := err.(*errors.StatusError)
		if !ok {
			return err
		}
		if se.ErrStatus.Code == 404 {
			continue
		}

		return err
	}

	return nil
}

func testAccCheckKubernetesPersistentVolumeExists(n string, res *api.PersistentVolume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Persistent Volume ID is set")
		}

		conn := testAccProvider.Meta().(*client.Client)
		pv, err := conn.PersistentVolumes().Get(rs.Primary.Attributes["name"])
		if err != nil {
			return err
		}

		if string(pv.UID) != rs.Primary.ID {
			return fmt.Errorf("Persistent Volume not found")
		}
		*res = *pv

		return nil
	}
}

const testAccKubernetesPersistentVolumeConfig_basic = `
resource "kubernetes_persistent_volume" "bar" {
    name = "wordpress"
    labels {
        Name = "WordPress"
    }
    spec = <<SPEC
capacity:
  storage: 10Gi
accessModes:
  - ReadWriteOnce
hostPath:
  path: "/tmp/data01"
SPEC
}
`
