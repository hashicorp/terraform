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

func TestAccKubernetesConfigMap_basic(t *testing.T) {
	var conf api.ConfigMap
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_config_map.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesConfigMapDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfigMapConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapExists("kubernetes_config_map.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_config_map.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_config_map.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_config_map.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_config_map.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "data.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "data.one", "first"),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "data.two", "second"),
					testAccCheckConfigMapData(&conf, map[string]string{"one": "first", "two": "second"}),
				),
			},
			{
				Config: testAccKubernetesConfigMapConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapExists("kubernetes_config_map.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.annotations.Different", "1234"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "Different": "1234"}),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_config_map.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_config_map.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_config_map.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_config_map.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "data.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "data.one", "first"),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "data.two", "second"),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "data.nine", "ninth"),
					testAccCheckConfigMapData(&conf, map[string]string{"one": "first", "two": "second", "nine": "ninth"}),
				),
			},
			{
				Config: testAccKubernetesConfigMapConfig_noData(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapExists("kubernetes_config_map.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_config_map.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_config_map.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_config_map.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_config_map.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "data.%", "0"),
					testAccCheckConfigMapData(&conf, map[string]string{}),
				),
			},
		},
	})
}

func TestAccKubernetesConfigMap_importBasic(t *testing.T) {
	resourceName := "kubernetes_config_map.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesConfigMapDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfigMapConfig_basic(name),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccKubernetesConfigMap_generatedName(t *testing.T) {
	var conf api.ConfigMap
	prefix := "tf-acc-test-gen-"

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_config_map.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesConfigMapDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfigMapConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapExists("kubernetes_config_map.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr("kubernetes_config_map.test", "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet("kubernetes_config_map.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_config_map.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_config_map.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_config_map.test", "metadata.0.uid"),
				),
			},
		},
	})
}

func TestAccKubernetesConfigMap_importGeneratedName(t *testing.T) {
	resourceName := "kubernetes_config_map.test"
	prefix := "tf-acc-test-gen-import-"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesConfigMapDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfigMapConfig_generatedName(prefix),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckConfigMapData(m *api.ConfigMap, expected map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(expected) == 0 && len(m.Data) == 0 {
			return nil
		}
		if !reflect.DeepEqual(m.Data, expected) {
			return fmt.Errorf("%s data don't match.\nExpected: %q\nGiven: %q",
				m.Name, expected, m.Data)
		}
		return nil
	}
}

func testAccCheckKubernetesConfigMapDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*kubernetes.Clientset)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_config_map" {
			continue
		}
		namespace, name := idParts(rs.Primary.ID)
		resp, err := conn.CoreV1().ConfigMaps(namespace).Get(name, meta_v1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Config Map still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesConfigMapExists(n string, obj *api.ConfigMap) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*kubernetes.Clientset)
		namespace, name := idParts(rs.Primary.ID)
		out, err := conn.CoreV1().ConfigMaps(namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesConfigMapConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_config_map" "test" {
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
	data {
		one = "first"
		two = "second"
	}
}`, name)
}

func testAccKubernetesConfigMapConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_config_map" "test" {
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
	data {
		one = "first"
		two = "second"
		nine = "ninth"
	}
}`, name)
}

func testAccKubernetesConfigMapConfig_noData(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_config_map" "test" {
	metadata {
		name = "%s"
	}
}`, name)
}

func testAccKubernetesConfigMapConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_config_map" "test" {
	metadata {
		generate_name = "%s"
	}
	data {
		one = "first"
		two = "second"
	}
}`, prefix)
}
