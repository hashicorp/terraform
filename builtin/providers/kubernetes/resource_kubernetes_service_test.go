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
	"k8s.io/apimachinery/pkg/util/intstr"
	api "k8s.io/kubernetes/pkg/api/v1"
	kubernetes "k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
)

func TestAccKubernetesService_basic(t *testing.T) {
	var conf api.Service
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_service.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists("kubernetes_service.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.#", "1"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.name", ""),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.node_port", "0"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.port", "8080"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.target_port", "80"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.session_affinity", "None"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.type", "ClusterIP"),
					testAccCheckServicePorts(&conf, []api.ServicePort{
						{
							Port:       int32(8080),
							Protocol:   api.ProtocolTCP,
							TargetPort: intstr.FromInt(80),
						},
					}),
				),
			},
			{
				Config: testAccKubernetesServiceConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists("kubernetes_service.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.#", "1"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.name", ""),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.node_port", "0"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.port", "8081"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.target_port", "80"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.session_affinity", "None"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.type", "ClusterIP"),
					testAccCheckServicePorts(&conf, []api.ServicePort{
						{
							Port:       int32(8081),
							Protocol:   api.ProtocolTCP,
							TargetPort: intstr.FromInt(80),
						},
					}),
				),
			},
		},
	})
}

func TestAccKubernetesService_loadBalancer(t *testing.T) {
	var conf api.Service
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_service.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_loadBalancer(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists("kubernetes_service.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.#", "1"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "spec.0.port.0.node_port"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.port", "8888"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.target_port", "80"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.external_ips.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.external_ips.1452553500", "10.0.0.4"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.external_ips.3371212991", "10.0.0.3"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.external_name", "ext-name-"+name),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.load_balancer_ip", "12.0.0.120"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.load_balancer_source_ranges.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.load_balancer_source_ranges.138364083", "10.0.0.5/32"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.load_balancer_source_ranges.445311837", "10.0.0.6/32"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.selector.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.selector.App", "MyApp"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.session_affinity", "ClientIP"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.type", "LoadBalancer"),
					testAccCheckServicePorts(&conf, []api.ServicePort{
						{
							Port:       int32(8888),
							Protocol:   api.ProtocolTCP,
							TargetPort: intstr.FromInt(80),
						},
					}),
				),
			},
			{
				Config: testAccKubernetesServiceConfig_loadBalancer_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists("kubernetes_service.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.#", "1"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.external_ips.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.external_ips.1452553500", "10.0.0.4"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.external_ips.563283338", "10.0.0.5"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.external_name", "ext-name-modified-"+name),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.load_balancer_ip", "12.0.0.125"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.load_balancer_source_ranges.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.load_balancer_source_ranges.2271073252", "10.0.0.1/32"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.load_balancer_source_ranges.2515041290", "10.0.0.2/32"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.#", "1"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "spec.0.port.0.node_port"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.port", "9999"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.target_port", "81"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.selector.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.selector.App", "MyModifiedApp"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.selector.NewSelector", "NewValue"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.session_affinity", "ClientIP"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.type", "LoadBalancer"),
					testAccCheckServicePorts(&conf, []api.ServicePort{
						{
							Port:       int32(9999),
							Protocol:   api.ProtocolTCP,
							TargetPort: intstr.FromInt(81),
						},
					}),
				),
			},
		},
	})
}

func TestAccKubernetesService_nodePort(t *testing.T) {
	var conf api.Service
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_service.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_nodePort(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists("kubernetes_service.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.#", "1"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.external_ips.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.external_ips.1452553500", "10.0.0.4"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.external_ips.563283338", "10.0.0.5"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.external_name", "ext-name-"+name),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.load_balancer_ip", "12.0.0.125"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.name", "first"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "spec.0.port.0.node_port"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.port", "10222"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.target_port", "22"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.1.name", "second"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "spec.0.port.1.node_port"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.1.port", "10333"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.1.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.1.target_port", "33"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.selector.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.selector.App", "MyApp"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.session_affinity", "ClientIP"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.type", "NodePort"),
					testAccCheckServicePorts(&conf, []api.ServicePort{
						{
							Name:       "first",
							Port:       int32(10222),
							Protocol:   api.ProtocolTCP,
							TargetPort: intstr.FromInt(22),
						},
						{
							Name:       "second",
							Port:       int32(10333),
							Protocol:   api.ProtocolTCP,
							TargetPort: intstr.FromInt(33),
						},
					}),
				),
			},
		},
	})
}

func TestAccKubernetesService_importBasic(t *testing.T) {
	resourceName := "kubernetes_service.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_basic(name),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccKubernetesService_generatedName(t *testing.T) {
	var conf api.Service
	prefix := "tf-acc-test-gen-"

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_service.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists("kubernetes_service.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr("kubernetes_service.test", "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.uid"),
				),
			},
		},
	})
}

func TestAccKubernetesService_importGeneratedName(t *testing.T) {
	resourceName := "kubernetes_service.test"
	prefix := "tf-acc-test-gen-import-"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_generatedName(prefix),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckServicePorts(svc *api.Service, expected []api.ServicePort) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(expected) == 0 && len(svc.Spec.Ports) == 0 {
			return nil
		}

		ports := svc.Spec.Ports

		// Ignore NodePorts as these are assigned randomly
		for k, _ := range ports {
			ports[k].NodePort = 0
		}

		if !reflect.DeepEqual(ports, expected) {
			return fmt.Errorf("Service ports don't match.\nExpected: %#v\nGiven: %#v",
				expected, svc.Spec.Ports)
		}

		return nil
	}
}

func testAccCheckKubernetesServiceDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*kubernetes.Clientset)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_service" {
			continue
		}
		namespace, name := idParts(rs.Primary.ID)
		resp, err := conn.CoreV1().Services(namespace).Get(name, meta_v1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Service still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesServiceExists(n string, obj *api.Service) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*kubernetes.Clientset)
		namespace, name := idParts(rs.Primary.ID)
		out, err := conn.CoreV1().Services(namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesServiceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_service" "test" {
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
		port {
			port = 8080
			target_port = 80
		}
	}
}`, name)
}

func testAccKubernetesServiceConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_service" "test" {
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
		port {
			port = 8081
			target_port = 80
		}
	}
}`, name)
}

func testAccKubernetesServiceConfig_loadBalancer(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_service" "test" {
	metadata {
		name = "%s"
	}
	spec {
		external_name = "ext-name-%s"
		external_ips = ["10.0.0.3", "10.0.0.4"]
		load_balancer_ip = "12.0.0.120"
		load_balancer_source_ranges = ["10.0.0.5/32", "10.0.0.6/32"]
		selector {
			App = "MyApp"
		}
		session_affinity = "ClientIP"
		port {
			port = 8888
			target_port = 80
		}
		type = "LoadBalancer"
	}
}`, name, name)
}

func testAccKubernetesServiceConfig_loadBalancer_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_service" "test" {
	metadata {
		name = "%s"
	}
	spec {
		external_name = "ext-name-modified-%s"
		external_ips = ["10.0.0.4", "10.0.0.5"]
		load_balancer_ip = "12.0.0.125"
		load_balancer_source_ranges = ["10.0.0.1/32", "10.0.0.2/32"]
		selector {
			App = "MyModifiedApp"
			NewSelector = "NewValue"
		}
		session_affinity = "ClientIP"
		port {
			port = 9999
			target_port = 81
		}
		type = "LoadBalancer"
	}
}`, name, name)
}

func testAccKubernetesServiceConfig_nodePort(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_service" "test" {
	metadata {
		name = "%s"
	}
	spec {
		external_name = "ext-name-%s"
		external_ips = ["10.0.0.4", "10.0.0.5"]
		load_balancer_ip = "12.0.0.125"
		selector {
			App = "MyApp"
		}
		session_affinity = "ClientIP"
		port {
			name = "first"
			port = 10222
			target_port = 22
		}
		port {
			name = "second"
			port = 10333
			target_port = 33
		}
		type = "NodePort"
	}
}`, name, name)
}

func testAccKubernetesServiceConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_service" "test" {
	metadata {
		generate_name = "%s"
	}
	spec {
		port {
			port = 8080
			target_port = 80
		}
	}
}`, prefix)
}
