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

func TestAccKubernetesResourceQuota_basic(t *testing.T) {
	var conf api.ResourceQuota

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesResourceQuotaDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccKubernetesResourceQuotaConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKubernetesResourceQuotaExists("kubernetes_resource_quota.bar", &conf),
				),
			},
		},
	})
}

func testAccCheckKubernetesResourceQuotaDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*client.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_resource_quota" {
			continue
		}

		rq, err := conn.ResourceQuotas(api.NamespaceDefault).Get(rs.Primary.Attributes["name"])
		if err == nil {
			if string(rq.UID) == rs.Primary.ID {
				return fmt.Errorf("Resource Quota still exists")
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

func testAccCheckKubernetesResourceQuotaExists(n string, res *api.ResourceQuota) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Resource Quota ID is set")
		}

		conn := testAccProvider.Meta().(*client.Client)
		rq, err := conn.ResourceQuotas(api.NamespaceDefault).Get(rs.Primary.Attributes["name"])
		if err != nil {
			return err
		}

		if string(rq.UID) != rs.Primary.ID {
			return fmt.Errorf("Resource Quota not found")
		}
		*res = *rq

		return nil
	}
}

const testAccKubernetesResourceQuotaConfig_basic = `
resource "kubernetes_resource_quota" "bar" {
    name = "wordpress"
    labels {
        Name = "WordPress"
    }
    spec = <<SPEC
{
  "hard": {
    "memory": "1Gi",
    "cpu": "20",
    "pods": "10",
    "services": "5",
    "replicationcontrollers":"20",
    "resourcequotas":"1"
  }
}
SPEC
}
`
