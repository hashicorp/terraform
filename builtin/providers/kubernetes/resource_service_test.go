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

func TestAccKubernetesService_basic(t *testing.T) {
	var conf api.Service

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccKubernetesServiceConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKubernetesServiceExists("kubernetes_service.bar", &conf),
				),
			},
		},
	})
}

func testAccCheckKubernetesServiceDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*client.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_service" {
			continue
		}

		svc, err := conn.Services(api.NamespaceDefault).Get(rs.Primary.Attributes["name"])
		if err == nil {
			if string(svc.UID) == rs.Primary.ID {
				return fmt.Errorf("Service still exists")
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

func testAccCheckKubernetesServiceExists(n string, res *api.Service) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Service ID is set")
		}

		conn := testAccProvider.Meta().(*client.Client)
		svc, err := conn.Services(api.NamespaceDefault).Get(rs.Primary.Attributes["name"])
		if err != nil {
			return err
		}

		if string(svc.UID) != rs.Primary.ID {
			return fmt.Errorf("Service not found")
		}
		*res = *svc

		return nil
	}
}

const testAccKubernetesServiceConfig_basic = `
resource "kubernetes_service" "bar" {
    name = "wordpress"
    labels {
        Name = "WordPress"
    }
    spec = <<SPEC
ports:
- port: 8000
  targetPort: 80
  protocol: TCP
selector:
  app: nginx
SPEC
}
`
