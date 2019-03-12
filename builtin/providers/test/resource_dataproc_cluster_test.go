package test

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/hashicorp/terraform/terraform"
)

var dataprocClusterSchema = map[string]*schema.Schema{
	"name": {
		Type:     schema.TypeString,
		Required: true,
		ForceNew: true,
	},

	"project": {
		Type:     schema.TypeString,
		Optional: true,
		Computed: true,
		ForceNew: true,
	},

	"region": {
		Type:     schema.TypeString,
		Optional: true,
		Default:  "global",
		ForceNew: true,
	},

	"labels": {
		Type:     schema.TypeMap,
		Optional: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
		// GCP automatically adds two labels
		//    'goog-dataproc-cluster-uuid'
		//    'goog-dataproc-cluster-name'
		Computed: true,
		DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
			if old != "" {
				return true
			}
			return false
		},
	},

	"tag_set": {
		Type:     schema.TypeSet,
		Optional: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
		Set:      schema.HashString,
	},

	"cluster_config": {
		Type:     schema.TypeList,
		Optional: true,
		Computed: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{

				"delete_autogen_bucket": {
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
					Removed: "If you need a bucket that can be deleted, please create" +
						"a new one and set the `staging_bucket` field",
				},

				"staging_bucket": {
					Type:     schema.TypeString,
					Optional: true,
					ForceNew: true,
				},
				"bucket": {
					Type:     schema.TypeString,
					Computed: true,
				},

				"gce_cluster_config": {
					Type:     schema.TypeList,
					Optional: true,
					Computed: true,
					MaxItems: 1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{

							"zone": {
								Type:     schema.TypeString,
								Optional: true,
								Computed: true,
								ForceNew: true,
							},

							"network": {
								Type:          schema.TypeString,
								Optional:      true,
								Computed:      true,
								ForceNew:      true,
								ConflictsWith: []string{"cluster_config.0.gce_cluster_config.0.subnetwork"},
							},

							"subnetwork": {
								Type:          schema.TypeString,
								Optional:      true,
								ForceNew:      true,
								ConflictsWith: []string{"cluster_config.0.gce_cluster_config.0.network"},
							},

							"tags": {
								Type:     schema.TypeSet,
								Optional: true,
								ForceNew: true,
								Elem:     &schema.Schema{Type: schema.TypeString},
							},

							"service_account": {
								Type:     schema.TypeString,
								Optional: true,
								ForceNew: true,
							},

							"service_account_scopes": {
								Type:     schema.TypeSet,
								Optional: true,
								Computed: true,
								ForceNew: true,
								Elem: &schema.Schema{
									Type: schema.TypeString,
								},
							},

							"internal_ip_only": {
								Type:     schema.TypeBool,
								Optional: true,
								ForceNew: true,
								Default:  false,
							},

							"metadata": {
								Type:     schema.TypeMap,
								Optional: true,
								Elem:     &schema.Schema{Type: schema.TypeString},
								ForceNew: true,
							},
						},
					},
				},

				"master_config": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Computed: true,
					MaxItems: 1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"num_instances": {
								Type:     schema.TypeInt,
								Optional: true,
								Computed: true,
							},

							"image_uri": {
								Type:     schema.TypeString,
								Optional: true,
								Computed: true,
								ForceNew: true,
							},

							"machine_type": {
								Type:     schema.TypeString,
								Optional: true,
								Computed: true,
								ForceNew: true,
							},

							"disk_config": {
								Type:     schema.TypeList,
								Optional: true,
								Computed: true,
								MaxItems: 1,

								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"num_local_ssds": {
											Type:     schema.TypeInt,
											Optional: true,
											Computed: true,
											ForceNew: true,
										},

										"boot_disk_size_gb": {
											Type:         schema.TypeInt,
											Optional:     true,
											Computed:     true,
											ForceNew:     true,
											ValidateFunc: validation.IntAtLeast(10),
										},

										"boot_disk_type": {
											Type:         schema.TypeString,
											Optional:     true,
											ForceNew:     true,
											ValidateFunc: validation.StringInSlice([]string{"pd-standard", "pd-ssd", ""}, false),
											Default:      "pd-standard",
										},
									},
								},
							},
							"accelerators": {
								Type:     schema.TypeSet,
								Optional: true,
								ForceNew: true,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"accelerator_type": {
											Type:     schema.TypeString,
											Required: true,
											ForceNew: true,
										},

										"accelerator_count": {
											Type:     schema.TypeInt,
											Required: true,
											ForceNew: true,
										},
									},
								},
							},
							"instance_names": {
								Type:     schema.TypeList,
								Computed: true,
								Elem:     &schema.Schema{Type: schema.TypeString},
							},
						},
					},
				},
				"preemptible_worker_config": {
					Type:     schema.TypeList,
					Optional: true,
					Computed: true,
					MaxItems: 1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"num_instances": {
								Type:     schema.TypeInt,
								Optional: true,
								Computed: true,
							},
							"disk_config": {
								Type:     schema.TypeList,
								Optional: true,
								Computed: true,
								MaxItems: 1,

								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"num_local_ssds": {
											Type:     schema.TypeInt,
											Optional: true,
											Computed: true,
											ForceNew: true,
										},

										"boot_disk_size_gb": {
											Type:         schema.TypeInt,
											Optional:     true,
											Computed:     true,
											ForceNew:     true,
											ValidateFunc: validation.IntAtLeast(10),
										},

										"boot_disk_type": {
											Type:         schema.TypeString,
											Optional:     true,
											ForceNew:     true,
											ValidateFunc: validation.StringInSlice([]string{"pd-standard", "pd-ssd", ""}, false),
											Default:      "pd-standard",
										},
									},
								},
							},

							"instance_names": {
								Type:     schema.TypeList,
								Computed: true,
								Elem:     &schema.Schema{Type: schema.TypeString},
							},
						},
					},
				},

				"software_config": {
					Type:     schema.TypeList,
					Optional: true,
					Computed: true,
					MaxItems: 1,

					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"image_version": {
								Type:     schema.TypeString,
								Optional: true,
								Computed: true,
								ForceNew: true,
							},

							"override_properties": {
								Type:     schema.TypeMap,
								Optional: true,
								ForceNew: true,
								Elem:     &schema.Schema{Type: schema.TypeString},
							},

							"properties": {
								Type:     schema.TypeMap,
								Computed: true,
							},
						},
					},
				},

				"initialization_action": {
					Type:     schema.TypeList,
					Optional: true,
					ForceNew: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"script": {
								Type:     schema.TypeString,
								Required: true,
								ForceNew: true,
							},

							"timeout_sec": {
								Type:     schema.TypeInt,
								Optional: true,
								Default:  300,
								ForceNew: true,
							},
						},
					},
				},
				"encryption_config": {
					Type:     schema.TypeList,
					Optional: true,
					MaxItems: 1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"kms_key_name": {
								Type:     schema.TypeString,
								Required: true,
							},
						},
					},
				},
			},
		},
	},
}

func TestDiffApply_dataprocCluster(t *testing.T) {
	priorAttrs := map[string]string{
		"cluster_config.#":                                                                                                       "1",
		"cluster_config.0.bucket":                                                                                                "dataproc-1dc18cb2-116e-4e92-85ea-ff63a1bf2745-us-central1",
		"cluster_config.0.delete_autogen_bucket":                                                                                 "false",
		"cluster_config.0.encryption_config.#":                                                                                   "0",
		"cluster_config.0.gce_cluster_config.#":                                                                                  "1",
		"cluster_config.0.gce_cluster_config.0.internal_ip_only":                                                                 "false",
		"cluster_config.0.gce_cluster_config.0.metadata.%":                                                                       "0",
		"cluster_config.0.gce_cluster_config.0.network":                                                                          "https://www.googleapis.com/compute/v1/projects/hc-terraform-testing/global/networks/default",
		"cluster_config.0.gce_cluster_config.0.service_account":                                                                  "",
		"cluster_config.0.gce_cluster_config.0.service_account_scopes.#":                                                         "7",
		"cluster_config.0.gce_cluster_config.0.service_account_scopes.1245378569":                                                "https://www.googleapis.com/auth/bigtable.admin.table",
		"cluster_config.0.gce_cluster_config.0.service_account_scopes.1328717722":                                                "https://www.googleapis.com/auth/devstorage.read_write",
		"cluster_config.0.gce_cluster_config.0.service_account_scopes.1693978638":                                                "https://www.googleapis.com/auth/devstorage.full_control",
		"cluster_config.0.gce_cluster_config.0.service_account_scopes.172152165":                                                 "https://www.googleapis.com/auth/logging.write",
		"cluster_config.0.gce_cluster_config.0.service_account_scopes.2401844655":                                                "https://www.googleapis.com/auth/bigquery",
		"cluster_config.0.gce_cluster_config.0.service_account_scopes.299921284":                                                 "https://www.googleapis.com/auth/bigtable.data",
		"cluster_config.0.gce_cluster_config.0.service_account_scopes.3804780973":                                                "https://www.googleapis.com/auth/cloud.useraccounts.readonly",
		"cluster_config.0.gce_cluster_config.0.subnetwork":                                                                       "",
		"cluster_config.0.gce_cluster_config.0.tags.#":                                                                           "0",
		"cluster_config.0.gce_cluster_config.0.zone":                                                                             "us-central1-f",
		"cluster_config.0.initialization_action.#":                                                                               "0",
		"cluster_config.0.master_config.#":                                                                                       "1",
		"cluster_config.0.master_config.0.accelerators.#":                                                                        "0",
		"cluster_config.0.master_config.0.disk_config.#":                                                                         "1",
		"cluster_config.0.master_config.0.disk_config.0.boot_disk_size_gb":                                                       "500",
		"cluster_config.0.master_config.0.disk_config.0.boot_disk_type":                                                          "pd-standard",
		"cluster_config.0.master_config.0.disk_config.0.num_local_ssds":                                                          "0",
		"cluster_config.0.master_config.0.image_uri":                                                                             "https://www.googleapis.com/compute/v1/projects/cloud-dataproc/global/images/dataproc-1-3-deb9-20190228-000000-rc01",
		"cluster_config.0.master_config.0.instance_names.#":                                                                      "1",
		"cluster_config.0.master_config.0.instance_names.0":                                                                      "dproc-cluster-test-2ww3c60iww-m",
		"cluster_config.0.master_config.0.machine_type":                                                                          "n1-standard-4",
		"cluster_config.0.master_config.0.num_instances":                                                                         "1",
		"cluster_config.0.preemptible_worker_config.#":                                                                           "1",
		"cluster_config.0.preemptible_worker_config.0.disk_config.#":                                                             "1",
		"cluster_config.0.preemptible_worker_config.0.instance_names.#":                                                          "0",
		"cluster_config.0.preemptible_worker_config.0.num_instances":                                                             "0",
		"cluster_config.0.software_config.#":                                                                                     "1",
		"cluster_config.0.software_config.0.image_version":                                                                       "1.3.28-deb9",
		"cluster_config.0.software_config.0.override_properties.%":                                                               "0",
		"cluster_config.0.software_config.0.properties.%":                                                                        "14",
		"cluster_config.0.software_config.0.properties.capacity-scheduler:yarn.scheduler.capacity.root.default.ordering-policy":  "fair",
		"cluster_config.0.software_config.0.properties.core:fs.gs.block.size":                                                    "134217728",
		"cluster_config.0.software_config.0.properties.core:fs.gs.metadata.cache.enable":                                         "false",
		"cluster_config.0.software_config.0.properties.core:hadoop.ssl.enabled.protocols":                                        "TLSv1,TLSv1.1,TLSv1.2",
		"cluster_config.0.software_config.0.properties.distcp:mapreduce.map.java.opts":                                           "-Xmx768m",
		"cluster_config.0.software_config.0.properties.distcp:mapreduce.map.memory.mb":                                           "1024",
		"cluster_config.0.software_config.0.properties.distcp:mapreduce.reduce.java.opts":                                        "-Xmx768m",
		"cluster_config.0.software_config.0.properties.distcp:mapreduce.reduce.memory.mb":                                        "1024",
		"cluster_config.0.software_config.0.properties.hdfs:dfs.datanode.address":                                                "0.0.0.0:9866",
		"cluster_config.0.software_config.0.properties.hdfs:dfs.datanode.http.address":                                           "0.0.0.0:9864",
		"cluster_config.0.software_config.0.properties.hdfs:dfs.datanode.https.address":                                          "0.0.0.0:9865",
		"cluster_config.0.software_config.0.properties.hdfs:dfs.datanode.ipc.address":                                            "0.0.0.0:9867",
		"cluster_config.0.software_config.0.properties.hdfs:dfs.namenode.handler.count":                                          "20",
		"cluster_config.0.software_config.0.properties.hdfs:dfs.namenode.http-address":                                           "0.0.0.0:9870",
		"cluster_config.0.software_config.0.properties.hdfs:dfs.namenode.https-address":                                          "0.0.0.0:9871",
		"cluster_config.0.software_config.0.properties.hdfs:dfs.namenode.lifeline.rpc-address":                                   "dproc-cluster-test-2ww3c60iww-m:8050",
		"cluster_config.0.software_config.0.properties.hdfs:dfs.namenode.secondary.http-address":                                 "0.0.0.0:9868",
		"cluster_config.0.software_config.0.properties.hdfs:dfs.namenode.secondary.https-address":                                "0.0.0.0:9869",
		"cluster_config.0.software_config.0.properties.hdfs:dfs.namenode.service.handler.count":                                  "10",
		"cluster_config.0.software_config.0.properties.hdfs:dfs.namenode.servicerpc-address":                                     "dproc-cluster-test-2ww3c60iww-m:8051",
		"cluster_config.0.software_config.0.properties.mapred-env:HADOOP_JOB_HISTORYSERVER_HEAPSIZE":                             "3840",
		"cluster_config.0.software_config.0.properties.mapred:mapreduce.job.maps":                                                "21",
		"cluster_config.0.software_config.0.properties.mapred:mapreduce.job.reduce.slowstart.completedmaps":                      "0.95",
		"cluster_config.0.software_config.0.properties.mapred:mapreduce.job.reduces":                                             "7",
		"cluster_config.0.software_config.0.properties.mapred:mapreduce.map.cpu.vcores":                                          "1",
		"cluster_config.0.software_config.0.properties.mapred:mapreduce.map.java.opts":                                           "-Xmx2457m",
		"cluster_config.0.software_config.0.properties.mapred:mapreduce.map.memory.mb":                                           "3072",
		"cluster_config.0.software_config.0.properties.mapred:mapreduce.reduce.cpu.vcores":                                       "1",
		"cluster_config.0.software_config.0.properties.mapred:mapreduce.reduce.java.opts":                                        "-Xmx2457m",
		"cluster_config.0.software_config.0.properties.mapred:mapreduce.reduce.memory.mb":                                        "3072",
		"cluster_config.0.software_config.0.properties.mapred:mapreduce.task.io.sort.mb":                                         "256",
		"cluster_config.0.software_config.0.properties.mapred:yarn.app.mapreduce.am.command-opts":                                "-Xmx2457m",
		"cluster_config.0.software_config.0.properties.mapred:yarn.app.mapreduce.am.resource.cpu-vcores":                         "1",
		"cluster_config.0.software_config.0.properties.mapred:yarn.app.mapreduce.am.resource.mb":                                 "3072",
		"cluster_config.0.software_config.0.properties.presto-jvm:MaxHeapSize":                                                   "12288m",
		"cluster_config.0.software_config.0.properties.presto:query.max-memory-per-node":                                         "7372MB",
		"cluster_config.0.software_config.0.properties.presto:query.max-total-memory-per-node":                                   "7372MB",
		"cluster_config.0.software_config.0.properties.spark-env:SPARK_DAEMON_MEMORY":                                            "3840m",
		"cluster_config.0.software_config.0.properties.spark:spark.driver.maxResultSize":                                         "1920m",
		"cluster_config.0.software_config.0.properties.spark:spark.driver.memory":                                                "3840m",
		"cluster_config.0.software_config.0.properties.spark:spark.executor.cores":                                               "2",
		"cluster_config.0.software_config.0.properties.spark:spark.executor.instances":                                           "2",
		"cluster_config.0.software_config.0.properties.spark:spark.executor.memory":                                              "5586m",
		"cluster_config.0.software_config.0.properties.spark:spark.executorEnv.OPENBLAS_NUM_THREADS":                             "1",
		"cluster_config.0.software_config.0.properties.spark:spark.scheduler.mode":                                               "FAIR",
		"cluster_config.0.software_config.0.properties.spark:spark.sql.cbo.enabled":                                              "true",
		"cluster_config.0.software_config.0.properties.spark:spark.yarn.am.memory":                                               "640m",
		"cluster_config.0.software_config.0.properties.yarn-env:YARN_TIMELINESERVER_HEAPSIZE":                                    "3840",
		"cluster_config.0.software_config.0.properties.yarn:yarn.nodemanager.resource.memory-mb":                                 "12288",
		"cluster_config.0.software_config.0.properties.yarn:yarn.resourcemanager.nodemanager-graceful-decommission-timeout-secs": "86400",
		"cluster_config.0.software_config.0.properties.yarn:yarn.scheduler.maximum-allocation-mb":                                "12288",
		"cluster_config.0.software_config.0.properties.yarn:yarn.scheduler.minimum-allocation-mb":                                "1024",
		"cluster_config.0.staging_bucket":                                                                                        "",
		"id":                                                                                                                     "dproc-cluster-test-ktbyrniu4e",
		"labels.%":                                                                                                               "4",
		"labels.goog-dataproc-cluster-name":                                                                                      "dproc-cluster-test-ktbyrniu4e",
		"labels.goog-dataproc-cluster-uuid":                                                                                      "d576c4e0-8fda-4ad1-abf5-ec951ab25855",
		"labels.goog-dataproc-location":                                                                                          "us-central1",
		"labels.key1":                                                                                                            "value1",
		"tag_set.#":                                                                                                              "0",
	}

	diff := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"labels.%":                          &terraform.ResourceAttrDiff{Old: "4", New: "1", NewComputed: false, NewRemoved: false, NewExtra: interface{}(nil), RequiresNew: false, Sensitive: false, Type: 0x0},
			"labels.goog-dataproc-cluster-name": &terraform.ResourceAttrDiff{Old: "dproc-cluster-test-ktbyrniu4e", New: "", NewComputed: false, NewRemoved: true, NewExtra: interface{}(nil), RequiresNew: false, Sensitive: false, Type: 0x0},
			"labels.goog-dataproc-cluster-uuid": &terraform.ResourceAttrDiff{Old: "d576c4e0-8fda-4ad1-abf5-ec951ab25855", New: "", NewComputed: false, NewRemoved: true, NewExtra: interface{}(nil), RequiresNew: false, Sensitive: false, Type: 0x0},
			"labels.goog-dataproc-location":     &terraform.ResourceAttrDiff{Old: "us-central1", New: "", NewComputed: false, NewRemoved: true, NewExtra: interface{}(nil), RequiresNew: false, Sensitive: false, Type: 0x0},
		},
	}

	newAttrs, err := diff.Apply(priorAttrs, (&schema.Resource{Schema: dataprocClusterSchema}).CoreConfigSchema())
	if err != nil {
		t.Fatal(err)
	}

	// the diff'ed labale elements should be removed
	delete(priorAttrs, "labels.goog-dataproc-cluster-name")
	delete(priorAttrs, "labels.goog-dataproc-cluster-uuid")
	delete(priorAttrs, "labels.goog-dataproc-location")
	priorAttrs["labels.%"] = "1"

	// the missing required "name" should be added
	priorAttrs["name"] = ""

	if !reflect.DeepEqual(priorAttrs, newAttrs) {
		t.Fatal(cmp.Diff(priorAttrs, newAttrs))
	}
}
