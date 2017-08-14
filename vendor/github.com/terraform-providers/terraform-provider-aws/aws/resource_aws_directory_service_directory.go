package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/directoryservice"
	"github.com/hashicorp/terraform/helper/resource"
)

var directoryCreationFuncs = map[string]func(*directoryservice.DirectoryService, *schema.ResourceData) (string, error){
	"SimpleAD":    createSimpleDirectoryService,
	"MicrosoftAD": createActiveDirectoryService,
	"ADConnector": createDirectoryConnector,
}

func resourceAwsDirectoryServiceDirectory() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDirectoryServiceDirectoryCreate,
		Read:   resourceAwsDirectoryServiceDirectoryRead,
		Update: resourceAwsDirectoryServiceDirectoryUpdate,
		Delete: resourceAwsDirectoryServiceDirectoryDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"password": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				Sensitive: true,
			},
			"size": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Large",
				ForceNew: true,
			},
			"alias": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"short_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"vpc_settings": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnet_ids": &schema.Schema{
							Type:     schema.TypeSet,
							Required: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"vpc_id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},
			"connect_settings": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"customer_username": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"customer_dns_ips": &schema.Schema{
							Type:     schema.TypeSet,
							Required: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"subnet_ids": &schema.Schema{
							Type:     schema.TypeSet,
							Required: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"vpc_id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},
			"enable_sso": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"access_url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"dns_ip_addresses": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
				Computed: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "SimpleAD",
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					validTypes := []string{"SimpleAD", "MicrosoftAD"}
					value := v.(string)
					for validType, _ := range directoryCreationFuncs {
						if validType == value {
							return
						}
					}
					es = append(es, fmt.Errorf("%q must be one of %q", k, validTypes))
					return
				},
			},
		},
	}
}

func buildVpcSettings(d *schema.ResourceData) (vpcSettings *directoryservice.DirectoryVpcSettings, err error) {
	if v, ok := d.GetOk("vpc_settings"); !ok {
		return nil, fmt.Errorf("vpc_settings is required for type = SimpleAD or MicrosoftAD")
	} else {
		settings := v.([]interface{})

		if len(settings) > 1 {
			return nil, fmt.Errorf("Only a single vpc_settings block is expected")
		} else if len(settings) == 1 {
			s := settings[0].(map[string]interface{})
			var subnetIds []*string
			for _, id := range s["subnet_ids"].(*schema.Set).List() {
				subnetIds = append(subnetIds, aws.String(id.(string)))
			}

			vpcSettings = &directoryservice.DirectoryVpcSettings{
				SubnetIds: subnetIds,
				VpcId:     aws.String(s["vpc_id"].(string)),
			}
		}
	}

	return vpcSettings, nil
}

func buildConnectSettings(d *schema.ResourceData) (connectSettings *directoryservice.DirectoryConnectSettings, err error) {
	if v, ok := d.GetOk("connect_settings"); !ok {
		return nil, fmt.Errorf("connect_settings is required for type = ADConnector")
	} else {
		settings := v.([]interface{})

		if len(settings) > 1 {
			return nil, fmt.Errorf("Only a single connect_settings block is expected")
		} else if len(settings) == 1 {
			s := settings[0].(map[string]interface{})

			var subnetIds []*string
			for _, id := range s["subnet_ids"].(*schema.Set).List() {
				subnetIds = append(subnetIds, aws.String(id.(string)))
			}

			var customerDnsIps []*string
			for _, id := range s["customer_dns_ips"].(*schema.Set).List() {
				customerDnsIps = append(customerDnsIps, aws.String(id.(string)))
			}

			connectSettings = &directoryservice.DirectoryConnectSettings{
				CustomerDnsIps:   customerDnsIps,
				CustomerUserName: aws.String(s["customer_username"].(string)),
				SubnetIds:        subnetIds,
				VpcId:            aws.String(s["vpc_id"].(string)),
			}
		}
	}

	return connectSettings, nil
}

func createDirectoryConnector(dsconn *directoryservice.DirectoryService, d *schema.ResourceData) (directoryId string, err error) {
	if _, ok := d.GetOk("size"); !ok {
		return "", fmt.Errorf("size is required for type = ADConnector")
	}

	input := directoryservice.ConnectDirectoryInput{
		Name:     aws.String(d.Get("name").(string)),
		Password: aws.String(d.Get("password").(string)),
		Size:     aws.String(d.Get("size").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}
	if v, ok := d.GetOk("short_name"); ok {
		input.ShortName = aws.String(v.(string))
	}

	input.ConnectSettings, err = buildConnectSettings(d)
	if err != nil {
		return "", err
	}

	log.Printf("[DEBUG] Creating Directory Connector: %s", input)
	out, err := dsconn.ConnectDirectory(&input)
	if err != nil {
		return "", err
	}
	log.Printf("[DEBUG] Directory Connector created: %s", out)

	return *out.DirectoryId, nil
}

func createSimpleDirectoryService(dsconn *directoryservice.DirectoryService, d *schema.ResourceData) (directoryId string, err error) {
	if _, ok := d.GetOk("size"); !ok {
		return "", fmt.Errorf("size is required for type = SimpleAD")
	}

	input := directoryservice.CreateDirectoryInput{
		Name:     aws.String(d.Get("name").(string)),
		Password: aws.String(d.Get("password").(string)),
		Size:     aws.String(d.Get("size").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}
	if v, ok := d.GetOk("short_name"); ok {
		input.ShortName = aws.String(v.(string))
	}

	input.VpcSettings, err = buildVpcSettings(d)
	if err != nil {
		return "", err
	}

	log.Printf("[DEBUG] Creating Simple Directory Service: %s", input)
	out, err := dsconn.CreateDirectory(&input)
	if err != nil {
		return "", err
	}
	log.Printf("[DEBUG] Simple Directory Service created: %s", out)

	return *out.DirectoryId, nil
}

func createActiveDirectoryService(dsconn *directoryservice.DirectoryService, d *schema.ResourceData) (directoryId string, err error) {
	input := directoryservice.CreateMicrosoftADInput{
		Name:     aws.String(d.Get("name").(string)),
		Password: aws.String(d.Get("password").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}
	if v, ok := d.GetOk("short_name"); ok {
		input.ShortName = aws.String(v.(string))
	}

	input.VpcSettings, err = buildVpcSettings(d)
	if err != nil {
		return "", err
	}

	log.Printf("[DEBUG] Creating Microsoft AD Directory Service: %s", input)
	out, err := dsconn.CreateMicrosoftAD(&input)
	if err != nil {
		return "", err
	}
	log.Printf("[DEBUG] Microsoft AD Directory Service created: %s", out)

	return *out.DirectoryId, nil
}

func resourceAwsDirectoryServiceDirectoryCreate(d *schema.ResourceData, meta interface{}) error {
	dsconn := meta.(*AWSClient).dsconn

	creationFunc, ok := directoryCreationFuncs[d.Get("type").(string)]
	if !ok {
		// Shouldn't happen as this is validated above
		return fmt.Errorf("Unsupported directory type: %s", d.Get("type"))
	}

	directoryId, err := creationFunc(dsconn, d)
	if err != nil {
		return err
	}

	d.SetId(directoryId)

	// Wait for creation
	log.Printf("[DEBUG] Waiting for DS (%q) to become available", d.Id())
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Requested", "Creating", "Created"},
		Target:  []string{"Active"},
		Refresh: func() (interface{}, string, error) {
			resp, err := dsconn.DescribeDirectories(&directoryservice.DescribeDirectoriesInput{
				DirectoryIds: []*string{aws.String(d.Id())},
			})
			if err != nil {
				log.Printf("Error during creation of DS: %q", err.Error())
				return nil, "", err
			}

			ds := resp.DirectoryDescriptions[0]
			log.Printf("[DEBUG] Creation of DS %q is in following stage: %q.",
				d.Id(), *ds.Stage)
			return ds, *ds.Stage, nil
		},
		Timeout: 60 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"Error waiting for Directory Service (%s) to become available: %s",
			d.Id(), err)
	}

	if v, ok := d.GetOk("alias"); ok {
		d.SetPartial("alias")

		input := directoryservice.CreateAliasInput{
			DirectoryId: aws.String(d.Id()),
			Alias:       aws.String(v.(string)),
		}

		log.Printf("[DEBUG] Assigning alias %q to DS directory %q",
			v.(string), d.Id())
		out, err := dsconn.CreateAlias(&input)
		if err != nil {
			return err
		}
		log.Printf("[DEBUG] Alias %q assigned to DS directory %q",
			*out.Alias, *out.DirectoryId)
	}

	return resourceAwsDirectoryServiceDirectoryUpdate(d, meta)
}

func resourceAwsDirectoryServiceDirectoryUpdate(d *schema.ResourceData, meta interface{}) error {
	dsconn := meta.(*AWSClient).dsconn

	if d.HasChange("enable_sso") {
		d.SetPartial("enable_sso")
		var err error

		if v, ok := d.GetOk("enable_sso"); ok && v.(bool) {
			log.Printf("[DEBUG] Enabling SSO for DS directory %q", d.Id())
			_, err = dsconn.EnableSso(&directoryservice.EnableSsoInput{
				DirectoryId: aws.String(d.Id()),
			})
		} else {
			log.Printf("[DEBUG] Disabling SSO for DS directory %q", d.Id())
			_, err = dsconn.DisableSso(&directoryservice.DisableSsoInput{
				DirectoryId: aws.String(d.Id()),
			})
		}

		if err != nil {
			return err
		}
	}

	return resourceAwsDirectoryServiceDirectoryRead(d, meta)
}

func resourceAwsDirectoryServiceDirectoryRead(d *schema.ResourceData, meta interface{}) error {
	dsconn := meta.(*AWSClient).dsconn

	input := directoryservice.DescribeDirectoriesInput{
		DirectoryIds: []*string{aws.String(d.Id())},
	}
	out, err := dsconn.DescribeDirectories(&input)
	if err != nil {
		return err

	}

	if len(out.DirectoryDescriptions) == 0 {
		log.Printf("[WARN] Directory %s not found", d.Id())
		d.SetId("")
		return nil
	}

	dir := out.DirectoryDescriptions[0]
	log.Printf("[DEBUG] Received DS directory: %s", dir)

	d.Set("access_url", *dir.AccessUrl)
	d.Set("alias", *dir.Alias)
	if dir.Description != nil {
		d.Set("description", *dir.Description)
	}

	if *dir.Type == "ADConnector" {
		d.Set("dns_ip_addresses", schema.NewSet(schema.HashString, flattenStringList(dir.ConnectSettings.ConnectIps)))
	} else {
		d.Set("dns_ip_addresses", schema.NewSet(schema.HashString, flattenStringList(dir.DnsIpAddrs)))
	}
	d.Set("name", *dir.Name)
	if dir.ShortName != nil {
		d.Set("short_name", *dir.ShortName)
	}
	if dir.Size != nil {
		d.Set("size", *dir.Size)
	}
	d.Set("type", *dir.Type)
	d.Set("vpc_settings", flattenDSVpcSettings(dir.VpcSettings))
	d.Set("connect_settings", flattenDSConnectSettings(dir.DnsIpAddrs, dir.ConnectSettings))
	d.Set("enable_sso", *dir.SsoEnabled)

	return nil
}

func resourceAwsDirectoryServiceDirectoryDelete(d *schema.ResourceData, meta interface{}) error {
	dsconn := meta.(*AWSClient).dsconn

	input := directoryservice.DeleteDirectoryInput{
		DirectoryId: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Delete Directory input: %s", input)
	_, err := dsconn.DeleteDirectory(&input)
	if err != nil {
		return err
	}

	// Wait for deletion
	log.Printf("[DEBUG] Waiting for DS (%q) to be deleted", d.Id())
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Deleting"},
		Target:  []string{"Deleted"},
		Refresh: func() (interface{}, string, error) {
			resp, err := dsconn.DescribeDirectories(&directoryservice.DescribeDirectoriesInput{
				DirectoryIds: []*string{aws.String(d.Id())},
			})
			if err != nil {
				if dserr, ok := err.(awserr.Error); ok && dserr.Code() == "EntityDoesNotExistException" {
					return 42, "Deleted", nil
				}
				return nil, "error", err
			}

			if len(resp.DirectoryDescriptions) == 0 {
				return 42, "Deleted", nil
			}

			ds := resp.DirectoryDescriptions[0]
			log.Printf("[DEBUG] Deletion of DS %q is in following stage: %q.",
				d.Id(), *ds.Stage)
			return ds, *ds.Stage, nil
		},
		Timeout: 60 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"Error waiting for Directory Service (%s) to be deleted: %q",
			d.Id(), err.Error())
	}

	return nil
}
