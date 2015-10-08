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

func TestAccKubernetesNamespace_basic(t *testing.T) {
	var conf api.Namespace

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesNamespaceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccKubernetesNamespaceConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKubernetesNamespaceExists("kubernetes_namespace.bar", &conf),
				),
			},
		},
	})
}

func testAccCheckKubernetesNamespaceDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*client.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_namespace" {
			continue
		}

		ns, err := conn.Namespaces().Get(rs.Primary.Attributes["name"])
		if err == nil {
			if string(ns.UID) == rs.Primary.ID {
				return fmt.Errorf("Namespace still exists")
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

func testAccCheckKubernetesNamespaceExists(n string, res *api.Namespace) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Namespace ID is set")
		}

		conn := testAccProvider.Meta().(*client.Client)
		ns, err := conn.Namespaces().Get(rs.Primary.Attributes["name"])
		if err != nil {
			return err
		}

		if string(ns.UID) != rs.Primary.ID {
			return fmt.Errorf("Namespace not found")
		}
		*res = *ns

		return nil
	}
}

const testAccKubernetesNamespaceConfig_basic = `
resource "kubernetes_namespace" "bar" {
    name = "myns"
    labels {
        name = "development"
    }
}
`
