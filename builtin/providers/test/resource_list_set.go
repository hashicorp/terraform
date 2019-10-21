package test

import (
	"fmt"
	"math/rand"

	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceListSet() *schema.Resource {
	return &schema.Resource{
		Create: testResourceListSetCreate,
		Read:   testResourceListSetRead,
		Delete: testResourceListSetDelete,
		Update: testResourceListSetUpdate,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"list": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"set": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"elem": {
										Type:     schema.TypeString,
										Optional: true,
										DiffSuppressFunc: func(_, o, n string, _ *schema.ResourceData) bool {
											return o == n
										},
									},
								},
							},
							Set: func(v interface{}) int {
								raw := v.(map[string]interface{})
								if el, ok := raw["elem"]; ok {
									return schema.HashString(el)
								}
								return 42
							},
						},
					},
				},
			},
			"replication_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"role": {
							Type:     schema.TypeString,
							Required: true,
						},
						"rules": {
							Type:     schema.TypeSet,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"destination": {
										Type:     schema.TypeSet,
										MaxItems: 1,
										MinItems: 1,
										Required: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"account_id": {
													Type:     schema.TypeString,
													Optional: true,
												},
												"bucket": {
													Type:     schema.TypeString,
													Required: true,
												},
												"storage_class": {
													Type:     schema.TypeString,
													Optional: true,
												},
												"replica_kms_key_id": {
													Type:     schema.TypeString,
													Optional: true,
												},
												"access_control_translation": {
													Type:     schema.TypeList,
													Optional: true,
													MinItems: 1,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"owner": {
																Type:     schema.TypeString,
																Required: true,
															},
														},
													},
												},
											},
										},
									},
									"source_selection_criteria": {
										Type:     schema.TypeSet,
										Optional: true,
										MinItems: 1,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"sse_kms_encrypted_objects": {
													Type:     schema.TypeSet,
													Optional: true,
													MinItems: 1,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"enabled": {
																Type:     schema.TypeBool,
																Required: true,
															},
														},
													},
												},
											},
										},
									},
									"prefix": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"status": {
										Type:     schema.TypeString,
										Required: true,
									},
									"priority": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									"filter": {
										Type:     schema.TypeList,
										Optional: true,
										MinItems: 1,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"prefix": {
													Type:     schema.TypeString,
													Optional: true,
												},
												"tags": {
													Type:     schema.TypeMap,
													Optional: true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func testResourceListSetCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId(fmt.Sprintf("%x", rand.Int63()))
	return testResourceListSetRead(d, meta)
}

func testResourceListSetUpdate(d *schema.ResourceData, meta interface{}) error {
	return testResourceListSetRead(d, meta)
}

func testResourceListSetRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceListSetDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
