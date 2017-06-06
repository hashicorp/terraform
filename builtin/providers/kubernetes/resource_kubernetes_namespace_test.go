package kubernetes

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "k8s.io/kubernetes/pkg/api/v1"
	kubernetes "k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
)

func TestAccKubernetesNamespace_basic(t *testing.T) {
	var conf api.Namespace
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_namespace.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceConfig_basic(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceExists("kubernetes_namespace.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.name", nsName),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.uid"),
				),
			},
			{
				Config: testAccKubernetesNamespaceConfig_smallerLists(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceExists("kubernetes_namespace.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.Different", "1234"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "Different": "1234"}),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.name", nsName),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.uid"),
				),
			},
			{
				Config: testAccKubernetesNamespaceConfig_noLists(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceExists("kubernetes_namespace.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.name", nsName),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.uid"),
				),
			},
		},
	})
}

func TestAccKubernetesNamespace_importBasic(t *testing.T) {
	resourceName := "kubernetes_namespace.test"
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceConfig_basic(nsName),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccKubernetesNamespace_generatedName(t *testing.T) {
	var conf api.Namespace
	prefix := "tf-acc-test-gen-"

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_namespace.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceExists("kubernetes_namespace.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr("kubernetes_namespace.test", "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.uid"),
				),
			},
		},
	})
}

func TestAccKubernetesNamespace_importGeneratedName(t *testing.T) {
	resourceName := "kubernetes_namespace.test"
	prefix := "tf-acc-test-gen-import-"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceConfig_generatedName(prefix),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckMetaAnnotations(om *meta_v1.ObjectMeta, expected map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(expected) == 0 && len(om.Annotations) == 0 {
			return nil
		}
		if !reflect.DeepEqual(om.Annotations, expected) {
			return fmt.Errorf("%s annotations don't match.\nExpected: %q\nGiven: %q",
				om.Name, expected, om.Annotations)
		}
		return nil
	}
}

func testAccCheckMetaLabels(om *meta_v1.ObjectMeta, expected map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(expected) == 0 && len(om.Labels) == 0 {
			return nil
		}
		if !reflect.DeepEqual(om.Labels, expected) {
			return fmt.Errorf("%s labels don't match.\nExpected: %q\nGiven: %q",
				om.Name, expected, om.Labels)
		}
		return nil
	}
}

func testAccCheckKubernetesNamespaceDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*kubernetes.Clientset)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_namespace" {
			continue
		}

		resp, err := conn.CoreV1().Namespaces().Get(rs.Primary.ID, meta_v1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Namespace still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesNamespaceExists(n string, obj *api.Namespace) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*kubernetes.Clientset)
		out, err := conn.CoreV1().Namespaces().Get(rs.Primary.ID, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesNamespaceConfig_basic(nsName string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace" "test" {
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
}`, nsName)
}

func testAccKubernetesNamespaceConfig_smallerLists(nsName string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace" "test" {
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
}`, nsName)
}

func testAccKubernetesNamespaceConfig_noLists(nsName string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace" "test" {
	metadata {
		name = "%s"
	}
}`, nsName)
}

func testAccKubernetesNamespaceConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace" "test" {
	metadata {
		generate_name = "%s"
	}
}`, prefix)
}
