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

func TestAccKubernetesReplicationController_basic(t *testing.T) {
	var conf api.ReplicationController

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesReplicationControllerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccKubernetesReplicationControllerConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKubernetesReplicationControllerExists("kubernetes_replication_controller.bar", &conf),
				),
			},
		},
	})
}

func testAccCheckKubernetesReplicationControllerDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*client.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_replication_controller" {
			continue
		}

		rc, err := conn.ReplicationControllers(api.NamespaceDefault).Get(rs.Primary.Attributes["name"])
		if err == nil {
			if string(rc.UID) == rs.Primary.ID {
				return fmt.Errorf("Replication Controller still exists")
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

func testAccCheckKubernetesReplicationControllerExists(n string, res *api.ReplicationController) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Replication Controller ID is set")
		}

		conn := testAccProvider.Meta().(*client.Client)
		rc, err := conn.ReplicationControllers(api.NamespaceDefault).Get(rs.Primary.Attributes["name"])
		if err != nil {
			return err
		}

		if string(rc.UID) != rs.Primary.ID {
			return fmt.Errorf("Replication Controller not found")
		}
		*res = *rc

		return nil
	}
}

const testAccKubernetesReplicationControllerConfig_basic = `
resource "kubernetes_replication_controller" "bar" {
    name = "wordpress"
    labels {
        Name = "WordPress"
    }
    spec = <<SPEC
replicas: 2
selector:
  app: nginx
template:
  metadata:
    labels:
      app: nginx
  spec:
    containers:
    - name: nginx
      image: nginx
      ports:
      - containerPort: 80
SPEC
}
`
