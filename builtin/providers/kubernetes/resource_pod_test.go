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

func TestAccKubernetesPod_basic(t *testing.T) {
	var conf api.Pod

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccKubernetesPodConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.bar", &conf),
				),
			},
		},
	})
}

func testAccCheckKubernetesPodDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*client.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_pod" {
			continue
		}

		pod, err := conn.Pods(api.NamespaceDefault).Get(rs.Primary.Attributes["name"])
		if err == nil {
			if string(pod.UID) == rs.Primary.ID {
				return fmt.Errorf("Pod still exists")
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

func testAccCheckKubernetesPodExists(n string, res *api.Pod) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Pod ID is set")
		}

		conn := testAccProvider.Meta().(*client.Client)
		pod, err := conn.Pods(api.NamespaceDefault).Get(rs.Primary.Attributes["name"])
		if err != nil {
			return err
		}

		if string(pod.UID) != rs.Primary.ID {
			return fmt.Errorf("Pod not found")
		}
		*res = *pod

		return nil
	}
}

const testAccKubernetesPodConfig_basic = `
resource "kubernetes_pod" "test" {
    name = "small-pod"
    container {
        image = "nginx"
        name = "web"

        port {
            container_port = 8000
            name = "web"
            protocol = "UDP"
        }

        volume_mount {
            name = "nfs"
            mount_path = "/usr/share/nginx/html"
        }

        image_pull_policy = "Always"

    }

    volume {
        name = "nfs"
        volume_source {
            nfs {
                server = "nfs-server.default.kube.local"
                path = "/"
            }
        }
    }
}
`
