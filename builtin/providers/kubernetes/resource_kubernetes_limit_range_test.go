package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "k8s.io/kubernetes/pkg/api/v1"
	kubernetes "k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
)

func TestAccKubernetesLimitRange_basic(t *testing.T) {
	var conf api.LimitRange
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_limit_range.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesLimitRangeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeExists("kubernetes_limit_range.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one"}),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.labels.TestLabelFour", "four"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelThree": "three", "TestLabelFour": "four"}),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.default.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.default.cpu", "200m"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.default.memory", "512M"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.default_request.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.default_request.cpu", "100m"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.default_request.memory", "256M"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.type", "Container"),
				),
			},
			{
				Config: testAccKubernetesLimitRangeConfig_metaModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeExists("kubernetes_limit_range.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.default.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.default.cpu", "200m"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.default.memory", "512M"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.default_request.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.default_request.cpu", "100m"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.default_request.memory", "256M"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.type", "Container"),
				),
			},
			{
				Config: testAccKubernetesLimitRangeConfig_specModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeExists("kubernetes_limit_range.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.default.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.default.cpu", "200m"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.default.memory", "1024M"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.default_request.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.default_request.cpu", "100m"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.default_request.memory", "256M"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.max.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.max.cpu", "500m"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.min.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.min.cpu", "10m"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.min.memory", "10M"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.type", "Container"),
				),
			},
		},
	})
}

func TestAccKubernetesLimitRange_generatedName(t *testing.T) {
	var conf api.LimitRange
	prefix := "tf-acc-test-"

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_limit_range.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesLimitRangeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeExists("kubernetes_limit_range.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.type", "Pod"),
				),
			},
		},
	})
}

func TestAccKubernetesLimitRange_typeChange(t *testing.T) {
	var conf api.LimitRange
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_limit_range.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesLimitRangeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeConfig_typeChange(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeExists("kubernetes_limit_range.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.default.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.default.cpu", "200m"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.default.memory", "1024M"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.type", "Container"),
				),
			},
			{
				Config: testAccKubernetesLimitRangeConfig_typeChangeModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeExists("kubernetes_limit_range.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.min.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.min.cpu", "200m"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.min.memory", "1024M"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.type", "Pod"),
				),
			},
		},
	})
}

func TestAccKubernetesLimitRange_multipleLimits(t *testing.T) {
	var conf api.LimitRange
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_limit_range.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesLimitRangeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeConfig_multipleLimits(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeExists("kubernetes_limit_range.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_limit_range.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.max.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.max.cpu", "200m"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.max.memory", "1024M"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.0.type", "Pod"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.1.min.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.1.min.storage", "24M"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.1.type", "PersistentVolumeClaim"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.2.default.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.2.default.cpu", "50m"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.2.default.memory", "24M"),
					resource.TestCheckResourceAttr("kubernetes_limit_range.test", "spec.0.limit.2.type", "Container"),
				),
			},
		},
	})
}

func TestAccKubernetesLimitRange_importBasic(t *testing.T) {
	resourceName := "kubernetes_limit_range.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesLimitRangeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeConfig_basic(name),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckKubernetesLimitRangeDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*kubernetes.Clientset)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_limit_range" {
			continue
		}
		namespace, name := idParts(rs.Primary.ID)
		resp, err := conn.CoreV1().LimitRanges(namespace).Get(name, meta_v1.GetOptions{})
		if err == nil {
			if resp.Namespace == namespace && resp.Name == name {
				return fmt.Errorf("Limit Range still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesLimitRangeExists(n string, obj *api.LimitRange) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*kubernetes.Clientset)
		namespace, name := idParts(rs.Primary.ID)
		out, err := conn.CoreV1().LimitRanges(namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesLimitRangeConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_limit_range" "test" {
	metadata {
		annotations {
			TestAnnotationOne = "one"
		}
		labels {
			TestLabelOne = "one"
			TestLabelThree = "three"
			TestLabelFour = "four"
		}
		name = "%s"
	}
	spec {
		limit {
			type = "Container"

			default {
				cpu = "200m"
				memory = "512M"
			}

			default_request {
				cpu = "100m"
				memory = "256M"
			}
		}
	}
}
`, name)
}

func testAccKubernetesLimitRangeConfig_metaModified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_limit_range" "test" {
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
		limit {
			type = "Container"

			default {
				cpu = "200m"
				memory = "512M"
			}

			default_request {
				cpu = "100m"
				memory = "256M"
			}
		}
	}
}
`, name)
}

func testAccKubernetesLimitRangeConfig_specModified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_limit_range" "test" {
	metadata {
		name = "%s"
	}
	spec {
		limit {
			type = "Container"

			default {
				cpu = "200m"
				memory = "1024M"
			}

			max {
				cpu = "500m"
			}

			min {
				cpu = "10m"
				memory = "10M"
			}
		}
	}
}
`, name)
}

func testAccKubernetesLimitRangeConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_limit_range" "test" {
	metadata {
		generate_name = "%s"
	}
	spec {
		limit {
			type = "Pod"
		}
	}
}
`, prefix)
}

func testAccKubernetesLimitRangeConfig_typeChange(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_limit_range" "test" {
	metadata {
		name = "%s"
	}
	spec {
		limit {
			type = "Container"
			default {
				cpu = "200m"
				memory = "1024M"
			}
		}
	}
}
`, name)
}

func testAccKubernetesLimitRangeConfig_typeChangeModified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_limit_range" "test" {
	metadata {
		name = "%s"
	}
	spec {
		limit {
			type = "Pod"
			min {
				cpu = "200m"
				memory = "1024M"
			}
		}
	}
}
`, name)
}

func testAccKubernetesLimitRangeConfig_multipleLimits(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_limit_range" "test" {
	metadata {
		name = "%s"
	}
	spec {
		limit {
			type = "Pod"
			max {
				cpu = "200m"
				memory = "1024M"
			}
		}
		limit {
			type = "PersistentVolumeClaim"
			min {
				storage = "24M"
			}
		}
		limit {
			type = "Container"
			default {
				cpu = "50m"
				memory = "24M"
			}
		}
	}
}
`, name)
}
