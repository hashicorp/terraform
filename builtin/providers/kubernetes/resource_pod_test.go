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
resource "kubernetes_pod" "bar" {
    name = "wordpress"
    labels {
        Name = "WordPress"
    }
    spec = <<SPEC
containers:
  - image: wordpress
    name: wordpress
    env:
      - name: WORDPRESS_DB_PASSWORD
        # change this - must match mysql.yaml password
        value: yourpassword
    ports:
      - containerPort: 80
        name: wordpress
    volumeMounts:
        # name must match the volume name below
      - name: wordpress-persistent-storage
        # mount path within the container
        mountPath: /var/www/html
volumes:
  - name: wordpress-persistent-storage
    gcePersistentDisk:
      # This GCE PD must already exist.
      pdName: wordpress-disk
      fsType: ext4
SPEC
}
`
