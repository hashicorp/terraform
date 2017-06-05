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

func TestAccKubernetesSecret_basic(t *testing.T) {
	var conf api.Secret
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_secret.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesSecretDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesSecretConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretExists("kubernetes_secret.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "data.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "data.one", "first"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "data.two", "second"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "type", "Opaque"),
					testAccCheckSecretData(&conf, map[string]string{"one": "first", "two": "second"}),
				),
			},
			{
				Config: testAccKubernetesSecretConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretExists("kubernetes_secret.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.annotations.Different", "1234"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "Different": "1234"}),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "data.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "data.one", "first"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "data.two", "second"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "data.nine", "ninth"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "type", "Opaque"),
					testAccCheckSecretData(&conf, map[string]string{"one": "first", "two": "second", "nine": "ninth"}),
				),
			},
			{
				Config: testAccKubernetesSecretConfig_noData(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretExists("kubernetes_secret.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "data.%", "0"),
					testAccCheckSecretData(&conf, map[string]string{}),
				),
			},
			{
				Config: testAccKubernetesSecretConfig_typeSpecified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretExists("kubernetes_secret.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "data.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "data.username", "admin"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "data.password", "password"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "type", "kubernetes.io/basic-auth"),
					testAccCheckSecretData(&conf, map[string]string{"username": "admin", "password": "password"}),
				),
			},
		},
	})
}

func TestAccKubernetesSecret_importBasic(t *testing.T) {
	resourceName := "kubernetes_secret.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesSecretDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesSecretConfig_basic(name),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccKubernetesSecret_generatedName(t *testing.T) {
	var conf api.Secret
	prefix := "tf-acc-test-gen-"

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_secret.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesSecretDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesSecretConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretExists("kubernetes_secret.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr("kubernetes_secret.test", "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.uid"),
				),
			},
		},
	})
}

func TestAccKubernetesSecret_importGeneratedName(t *testing.T) {
	resourceName := "kubernetes_secret.test"
	prefix := "tf-acc-test-gen-import-"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesSecretDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesSecretConfig_generatedName(prefix),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckSecretData(m *api.Secret, expected map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(expected) == 0 && len(m.Data) == 0 {
			return nil
		}
		if !reflect.DeepEqual(byteMapToStringMap(m.Data), expected) {
			return fmt.Errorf("%s data don't match.\nExpected: %q\nGiven: %q",
				m.Name, expected, m.Data)
		}
		return nil
	}
}

func testAccCheckKubernetesSecretDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*kubernetes.Clientset)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_secret" {
			continue
		}
		namespace, name := idParts(rs.Primary.ID)
		resp, err := conn.CoreV1().Secrets(namespace).Get(name, meta_v1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Secret still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesSecretExists(n string, obj *api.Secret) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*kubernetes.Clientset)
		namespace, name := idParts(rs.Primary.ID)
		out, err := conn.CoreV1().Secrets(namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesSecretConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_secret" "test" {
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

func testAccKubernetesSecretConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_secret" "test" {
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

func testAccKubernetesSecretConfig_noData(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_secret" "test" {
	metadata {
		name = "%s"
	}
}`, name)
}

func testAccKubernetesSecretConfig_typeSpecified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_secret" "test" {
	metadata {
		name = "%s"
	}
	data {
		username = "admin"
		password = "password"
	}
	type = "kubernetes.io/basic-auth"
}`, name)
}

func testAccKubernetesSecretConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_secret" "test" {
	metadata {
		generate_name = "%s"
	}
	data {
		one = "first"
		two = "second"
	}
}`, prefix)
}
