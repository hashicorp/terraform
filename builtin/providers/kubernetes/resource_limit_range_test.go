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

func TestAccKubernetesLimitRange_basic(t *testing.T) {
	var conf api.LimitRange

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesLimitRangeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccKubernetesLimitRangeConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKubernetesLimitRangeExists("kubernetes_limit_range.bar", &conf),
				),
			},
		},
	})
}

func testAccCheckKubernetesLimitRangeDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*client.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_limit_range" {
			continue
		}

		lr, err := conn.LimitRanges(api.NamespaceDefault).Get(rs.Primary.Attributes["name"])
		if err == nil {
			if string(lr.UID) == rs.Primary.ID {
				return fmt.Errorf("Limit Range still exists")
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

func testAccCheckKubernetesLimitRangeExists(n string, res *api.LimitRange) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Limit Range ID is set")
		}

		conn := testAccProvider.Meta().(*client.Client)
		lr, err := conn.LimitRanges(api.NamespaceDefault).Get(rs.Primary.Attributes["name"])
		if err != nil {
			return err
		}

		if string(lr.UID) != rs.Primary.ID {
			return fmt.Errorf("Limit Range not found")
		}
		*res = *lr

		return nil
	}
}

const testAccKubernetesLimitRangeConfig_basic = `
resource "kubernetes_limit_range" "bar" {
    name = "wordpress"
    labels {
        Name = "WordPress"
    }
    spec = <<SPEC
limits:
  - max:
      cpu: "2"
      memory: 1Gi
    min:
      cpu: 250m
      memory: 6Mi
    type: Pod
  - default:
      cpu: 250m
      memory: 100Mi
    max:
      cpu: "2"
      memory: 1Gi
    min:
      cpu: 250m
      memory: 6Mi
    type: Container
SPEC
}
`
