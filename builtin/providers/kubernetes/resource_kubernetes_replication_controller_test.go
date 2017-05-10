package kubernetes

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "k8s.io/kubernetes/pkg/api/v1"
	kubernetes "k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
)

func TestAccKubernetesReplicationController_basic(t *testing.T) {
	var conf api.ReplicationController
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_replication_controller.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesReplicationControllerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerExists("kubernetes_replication_controller.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.uid"),
					// TODO
				),
			},
			{
				Config: testAccKubernetesReplicationControllerConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerExists("kubernetes_replication_controller.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.annotations.Different", "1234"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "Different": "1234"}),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.uid"),
					// TODO
				),
			},
		},
	})
}

func TestAccKubernetesReplicationController_importBasic(t *testing.T) {
	resourceName := "kubernetes_replication_controller.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesReplicationControllerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerConfig_basic(name),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccKubernetesReplicationController_generatedName(t *testing.T) {
	var conf api.ReplicationController
	prefix := "tf-acc-test-gen-"

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_replication_controller.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesReplicationControllerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerExists("kubernetes_replication_controller.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.labels.%", "3"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr("kubernetes_replication_controller.test", "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.uid"),
				),
			},
		},
	})
}

func TestAccKubernetesReplicationController_importGeneratedName(t *testing.T) {
	resourceName := "kubernetes_replication_controller.test"
	prefix := "tf-acc-test-gen-import-"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesReplicationControllerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerConfig_generatedName(prefix),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckKubernetesReplicationControllerDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*kubernetes.Clientset)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_replication_controller" {
			continue
		}
		namespace, name := idParts(rs.Primary.ID)
		resp, err := conn.CoreV1().ReplicationControllers(namespace).Get(name, meta_v1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Replication Controller still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesReplicationControllerExists(n string, obj *api.ReplicationController) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*kubernetes.Clientset)
		namespace, name := idParts(rs.Primary.ID)
		out, err := conn.CoreV1().ReplicationControllers(namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesReplicationControllerConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_replication_controller" "test" {
  metadata {
    annotations {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }
    labels {
      TestLabelOne = "one"
      TestLabelTwo = "two"
      TestLabelThree = "three"
    }
    name = "%s"
  }
  spec {
    selector {
      TestLabelOne = "one"
      TestLabelTwo = "two"
      TestLabelThree = "three"
    }
    template {
      container {
        image = "nginx:1.7.9"
        name  = "tf-acc-test"
      }
    }
  }
}
`, name)
}

func testAccKubernetesReplicationControllerConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_replication_controller" "test" {
  metadata {
    annotations {
      TestAnnotationOne = "one"
      Different = "1234"
    }
    labels {
      TestLabelOne = "one"
      TestLabelThree = "three"
    }
    name = "%s"
  }
  spec {
    selector {
      TestLabelOne = "one"
      TestLabelTwo = "two"
      TestLabelThree = "three"
    }
    template {
      container {
        image = "nginx:1.7.9"
        name  = "tf-acc-test"
      }
    }
  }
}`, name)
}

func testAccKubernetesReplicationControllerConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_replication_controller" "test" {
  metadata {
    labels {
      TestLabelOne = "one"
      TestLabelTwo = "two"
      TestLabelThree = "three"
    }
    generate_name = "%s"
  }
  spec {
    selector {
      TestLabelOne = "one"
      TestLabelTwo = "two"
      TestLabelThree = "three"
    }
    template {
      container {
        image = "nginx:1.7.9"
        name  = "tf-acc-test"
      }
    }
  }
}`, prefix)
}
