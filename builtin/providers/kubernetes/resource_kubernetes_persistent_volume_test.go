package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	api "k8s.io/kubernetes/pkg/api/v1"
	kubernetes "k8s.io/kubernetes/pkg/client/clientset_generated/release_1_5"
)

func TestAccKubernetesPersistentVolume_basic(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_persistent_volume.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "123Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1254135962", "ReadWriteMany"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.gce_persistent_disk.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.gce_persistent_disk.0.pd_name", "test123"),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "42Mi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1254135962", "ReadWriteMany"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.gce_persistent_disk.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.gce_persistent_disk.0.fs_type", "ntfs"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.gce_persistent_disk.0.pd_name", "test123"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.gce_persistent_disk.0.read_only", "true"),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolume_importBasic(t *testing.T) {
	resourceName := "kubernetes_persistent_volume.test"
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-import-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeConfig_basic(name),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccKubernetesPersistentVolume_volumeSource(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_persistent_volume.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeConfig_volumeSource(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "123Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1254135962", "ReadWriteMany"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.aws_elastic_block_store.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.aws_elastic_block_store.0.volume_id", "vol-12345678"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.aws_elastic_block_store.0.fs_type", "ntfs"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.aws_elastic_block_store.0.partition", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.aws_elastic_block_store.0.read_only", "true"),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeConfig_volumeSource_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "123Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1254135962", "ReadWriteMany"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.host_path.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.host_path.0.path", "/custom/testing/path"),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolume_cephFsSecretRef(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_persistent_volume.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeConfig_cephFsSecretRef(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "2Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1254135962", "ReadWriteMany"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.ceph_fs.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.ceph_fs.0.monitors.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.ceph_fs.0.monitors.2848821021", "10.16.154.78:6789"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.ceph_fs.0.monitors.4263435410", "10.16.154.82:6789"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.ceph_fs.0.secret_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.ceph_fs.0.secret_ref.0.name", "ceph-secret"),
				),
			},
		},
	})
}

func testAccCheckKubernetesPersistentVolumeDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*kubernetes.Clientset)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_persistent_volume" {
			continue
		}
		name := rs.Primary.ID
		resp, err := conn.CoreV1().PersistentVolumes().Get(name)
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Persistent Volume still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesPersistentVolumeExists(n string, obj *api.PersistentVolume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*kubernetes.Clientset)
		name := rs.Primary.ID
		out, err := conn.CoreV1().PersistentVolumes().Get(name)
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesPersistentVolumeConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
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
		capacity {
			storage = "123Gi"
		}
		access_modes = ["ReadWriteMany"]
		persistent_volume_source {
			gce_persistent_disk {
				pd_name = "test123"
			}
		}
	}
}`, name)
}

func testAccKubernetesPersistentVolumeConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
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
		capacity {
			storage = "42Mi"
		}
		access_modes = ["ReadWriteMany", "ReadWriteOnce"]
		persistent_volume_source {
			gce_persistent_disk {
				fs_type = "ntfs"
				pd_name = "test123"
				read_only = true
			}
		}
	}
}`, name)
}

func testAccKubernetesPersistentVolumeConfig_volumeSource(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
	metadata {
		name = "%s"
	}
	spec {
		capacity {
			storage = "123Gi"
		}
		access_modes = ["ReadWriteMany"]
		persistent_volume_source {
			aws_elastic_block_store {
				volume_id = "vol-12345678"
				fs_type = "ntfs"
				partition = 1
				read_only = true
			}
		}
	}
}`, name)
}

func testAccKubernetesPersistentVolumeConfig_volumeSource_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
	metadata {
		name = "%s"
	}
	spec {
		capacity {
			storage = "123Gi"
		}
		access_modes = ["ReadWriteMany"]
		persistent_volume_source {
			host_path {
				path = "/custom/testing/path"
			}
		}
	}
}`, name)
}

func testAccKubernetesPersistentVolumeConfig_cephFsSecretRef(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
	metadata {
		name = "%s"
	}
	spec {
		capacity {
			storage = "2Gi"
		}
		access_modes = ["ReadWriteMany"]
		persistent_volume_source {
			ceph_fs {
				monitors = ["10.16.154.78:6789", "10.16.154.82:6789"]
				secret_ref {
					name = "ceph-secret"
				}
			}
		}
	}
}`, name)
}
