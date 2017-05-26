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

func TestAccKubernetesResourceQuota_basic(t *testing.T) {
	var conf api.ResourceQuota
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_resource_quota.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesResourceQuotaDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesResourceQuotaConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaExists("kubernetes_resource_quota.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one"}),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.labels.TestLabelFour", "four"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelThree": "three", "TestLabelFour": "four"}),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.hard.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.hard.limits.cpu", "2"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.hard.limits.memory", "2Gi"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.hard.pods", "4"),
				),
			},
			{
				Config: testAccKubernetesResourceQuotaConfig_metaModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaExists("kubernetes_resource_quota.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.hard.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.hard.limits.cpu", "2"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.hard.limits.memory", "2Gi"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.hard.pods", "4"),
				),
			},
			{
				Config: testAccKubernetesResourceQuotaConfig_specModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaExists("kubernetes_resource_quota.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.hard.%", "4"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.hard.limits.cpu", "4"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.hard.requests.cpu", "1"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.hard.limits.memory", "4Gi"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.hard.pods", "10"),
				),
			},
		},
	})
}

func TestAccKubernetesResourceQuota_generatedName(t *testing.T) {
	var conf api.ResourceQuota
	prefix := "tf-acc-test-"

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_resource_quota.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesResourceQuotaDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesResourceQuotaConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaExists("kubernetes_resource_quota.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.hard.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.hard.pods", "10"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.scopes.#", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesResourceQuota_withScopes(t *testing.T) {
	var conf api.ResourceQuota
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_resource_quota.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesResourceQuotaDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesResourceQuotaConfig_withScopes(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaExists("kubernetes_resource_quota.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.hard.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.hard.pods", "10"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.scopes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.scopes.193563370", "BestEffort"),
				),
			},
			{
				Config: testAccKubernetesResourceQuotaConfig_withScopesModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaExists("kubernetes_resource_quota.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_resource_quota.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.hard.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.hard.pods", "10"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.scopes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_resource_quota.test", "spec.0.scopes.3022121741", "NotBestEffort"),
				),
			},
		},
	})
}

func TestAccKubernetesResourceQuota_importBasic(t *testing.T) {
	resourceName := "kubernetes_resource_quota.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesResourceQuotaDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesResourceQuotaConfig_basic(name),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckKubernetesResourceQuotaDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*kubernetes.Clientset)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_resource_quota" {
			continue
		}
		namespace, name := idParts(rs.Primary.ID)
		resp, err := conn.CoreV1().ResourceQuotas(namespace).Get(name, meta_v1.GetOptions{})
		if err == nil {
			if resp.Namespace == namespace && resp.Name == name {
				return fmt.Errorf("Resource Quota still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesResourceQuotaExists(n string, obj *api.ResourceQuota) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*kubernetes.Clientset)
		namespace, name := idParts(rs.Primary.ID)
		out, err := conn.CoreV1().ResourceQuotas(namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesResourceQuotaConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_resource_quota" "test" {
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
		hard {
			"limits.cpu" = 2
			"limits.memory" = "2Gi"
			pods = 4
		}
	}
}
`, name)
}

func testAccKubernetesResourceQuotaConfig_metaModified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_resource_quota" "test" {
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
		hard {
			"limits.cpu" = 2
			"limits.memory" = "2Gi"
			pods = 4
		}
	}
}
`, name)
}

func testAccKubernetesResourceQuotaConfig_specModified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_resource_quota" "test" {
	metadata {
		name = "%s"
	}
	spec {
		hard {
			"limits.cpu" = 4
			"requests.cpu" = 1
			"limits.memory" = "4Gi"
			pods = 10
		}
	}
}
`, name)
}

func testAccKubernetesResourceQuotaConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_resource_quota" "test" {
	metadata {
		generate_name = "%s"
	}
	spec {
		hard {
			pods = 10
		}
	}
}
`, prefix)
}

func testAccKubernetesResourceQuotaConfig_withScopes(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_resource_quota" "test" {
	metadata {
		name = "%s"
	}
	spec {
		hard {
			pods = 10
		}
		scopes = ["BestEffort"]
	}
}
`, name)
}

func testAccKubernetesResourceQuotaConfig_withScopesModified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_resource_quota" "test" {
	metadata {
		name = "%s"
	}
	spec {
		hard {
			pods = 10
		}
		scopes = ["NotBestEffort"]
	}
}
`, name)
}
