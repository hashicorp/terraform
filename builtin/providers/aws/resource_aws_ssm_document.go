package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSsmDocument() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSsmDocumentCreate,
		Read:   resourceAwsSsmDocumentRead,
		Update: resourceAwsSsmDocumentUpdate,
		Delete: resourceAwsSsmDocumentDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"content": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"created_date": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"hash": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"hash_type": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"owner": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"platform_type": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"parameter": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"default_value": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"description": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"permissions": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"account_ids": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsSsmDocumentCreate(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[INFO] Creating SSM Document: %s", d.Get("name").(string))

	docInput := &ssm.CreateDocumentInput{
		Name:    aws.String(d.Get("name").(string)),
		Content: aws.String(d.Get("content").(string)),
	}

	resp, err := ssmconn.CreateDocument(docInput)

	if err != nil {
		return fmt.Errorf("[ERROR] Error creating SSM document: %s", err)
	}

	d.SetId(*resp.DocumentDescription.Name)

	if v, ok := d.GetOk("permissions"); ok && v != nil {
		setDocumentPermissions(d, meta)
	} else {
		log.Printf("[DEBUG] not setting document permissions")
	}

	return resourceAwsSsmDocumentRead(d, meta)
}

func resourceAwsSsmDocumentRead(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[DEBUG] Reading SSM Document: %s", d.Get("name").(string))

	docInput := &ssm.DescribeDocumentInput{
		Name: aws.String(d.Get("name").(string)),
	}

	resp, err := ssmconn.DescribeDocument(docInput)

	if err != nil {
		return fmt.Errorf("[ERROR] Error describing SSM document: %s", err)
	}

	doc := resp.Document
	d.Set("created_date", doc.CreatedDate)
	d.Set("description", doc.Description)
	d.Set("hash", doc.Hash)
	d.Set("hash_type", doc.HashType)
	d.Set("name", doc.Name)
	d.Set("owner", doc.Owner)
	d.Set("platform_type", doc.PlatformTypes[0])
	d.Set("status", doc.Status)

	gp, err := getDocumentPermissions(d, meta)

	if err != nil {
		return fmt.Errorf("[ERROR] Error reading SSM document permissions: %s", err)
	}

	d.Set("permissions", gp)

	params := make([]map[string]interface{}, 0)
	for i := 0; i < len(doc.Parameters); i++ {

		dp := doc.Parameters[i]
		param := make(map[string]interface{})

		if dp.DefaultValue != nil {
			param["default_value"] = *dp.DefaultValue
		}
		param["description"] = *dp.Description
		param["name"] = *dp.Name
		param["type"] = *dp.Type
		params = append(params, param)
	}

	if len(params) == 0 {
		params = make([]map[string]interface{}, 1)
	}

	if err := d.Set("parameter", params); err != nil {
		return err
	}

	return nil
}

func resourceAwsSsmDocumentUpdate(d *schema.ResourceData, meta interface{}) error {

	if _, ok := d.GetOk("permissions"); ok {
		setDocumentPermissions(d, meta)
	} else {
		log.Printf("[DEBUG] not setting document permissions")
	}

	return resourceAwsSsmDocumentRead(d, meta)
}

func resourceAwsSsmDocumentDelete(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	deleteDocumentPermissions(d, meta)

	log.Printf("[INFO] Deleting SSM Document: %s", d.Get("name").(string))

	params := &ssm.DeleteDocumentInput{
		Name: aws.String(d.Get("name").(string)),
	}

	_, err := ssmconn.DeleteDocument(params)

	if err != nil {
		return err
	}

	return nil
}

func setDocumentPermissions(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[INFO] Setting permissions for document: %s", d.Get("name").(string))
	permission := d.Get("permissions").(map[string]interface{})

	ids := aws.StringSlice([]string{permission["account_ids"].(string)})

	if strings.Contains(permission["account_ids"].(string), ",") {
		ids = aws.StringSlice(strings.Split(permission["account_ids"].(string), ","))
	}

	permInput := &ssm.ModifyDocumentPermissionInput{
		Name:            aws.String(d.Get("name").(string)),
		PermissionType:  aws.String(permission["type"].(string)),
		AccountIdsToAdd: ids,
	}

	_, err := ssmconn.ModifyDocumentPermission(permInput)

	if err != nil {
		return fmt.Errorf("[ERROR] Error setting permissions for SSM document: %s", err)
	}

	return nil
}

func getDocumentPermissions(d *schema.ResourceData, meta interface{}) (map[string]interface{}, error) {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[INFO] Getting permissions for document: %s", d.Get("name").(string))

	//How to get from nested scheme resource?
	permissionType := "Share"

	permInput := &ssm.DescribeDocumentPermissionInput{
		Name:           aws.String(d.Get("name").(string)),
		PermissionType: aws.String(permissionType),
	}

	resp, err := ssmconn.DescribeDocumentPermission(permInput)

	if err != nil {
		return nil, fmt.Errorf("[ERROR] Error setting permissions for SSM document: %s", err)
	}

	var account_ids = make([]string, len(resp.AccountIds))
	for i := 0; i < len(resp.AccountIds); i++ {
		account_ids[i] = *resp.AccountIds[i]
	}

	var ids = ""
	if len(account_ids) == 1 {
		ids = account_ids[0]
	} else if len(account_ids) > 1 {
		ids = strings.Join(account_ids, ",")
	} else {
		ids = ""
	}

	if ids == "" {
		return nil, nil
	}

	perms := make(map[string]interface{})
	perms["type"] = permissionType
	perms["account_ids"] = ids

	return perms, nil
}

func deleteDocumentPermissions(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[INFO] Removing permissions from document: %s", d.Get("name").(string))

	permInput := &ssm.ModifyDocumentPermissionInput{
		Name:               aws.String(d.Get("name").(string)),
		PermissionType:     aws.String("Share"),
		AccountIdsToRemove: aws.StringSlice(strings.Split("all", ",")),
	}

	_, err := ssmconn.ModifyDocumentPermission(permInput)

	if err != nil {
		return fmt.Errorf("[ERROR] Error removing permissions for SSM document: %s", err)
	}

	return nil
}
