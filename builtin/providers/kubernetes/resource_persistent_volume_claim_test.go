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

func TestAccKubernetesPersistentVolumeClaim_basic(t *testing.T) {
	var conf api.PersistentVolumeClaim

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesPersistentVolumeClaimDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccKubernetesPersistentVolumeClaimConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimExists("kubernetes_persistent_volume_claim.bar", &conf),
				),
			},
		},
	})
}

func testAccCheckKubernetesPersistentVolumeClaimDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*client.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_persistent_volume_claim" {
			continue
		}

		pvc, err := conn.PersistentVolumeClaims(api.NamespaceDefault).Get(rs.Primary.Attributes["name"])
		if err == nil {
			if string(pvc.UID) == rs.Primary.ID {
				return fmt.Errorf("Persistent Volume Claim still exists")
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

func testAccCheckKubernetesPersistentVolumeClaimExists(n string, res *api.PersistentVolumeClaim) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Persistent Volume Claim ID is set")
		}

		conn := testAccProvider.Meta().(*client.Client)
		pvc, err := conn.PersistentVolumeClaims(api.NamespaceDefault).Get(rs.Primary.Attributes["name"])
		if err != nil {
			return err
		}

		if string(pvc.UID) != rs.Primary.ID {
			return fmt.Errorf("Persistent Volume Claim not found")
		}
		*res = *pvc

		return nil
	}
}

const testAccKubernetesPersistentVolumeClaimConfig_basic = `
resource "kubernetes_persistent_volume_claim" "bar" {
    name = "wordpress"
    labels {
        Name = "WordPress"
    }
    spec = <<SPEC
accessModes:
  - ReadWriteOnce
resources:
  requests:
    storage: 8Gi
SPEC
}
`
