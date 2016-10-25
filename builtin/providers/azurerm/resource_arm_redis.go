package azurerm

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/arm/redis"
	"github.com/hashicorp/terraform/helper/schema"
	"net/http"
	"strings"
)

func resourceArmRedis() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmRedisCreate,
		Read:   resourceArmRedisRead,
		Update: resourceArmRedisCreate,
		Delete: resourceArmRedisDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": {
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"redis_version": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"capacity": {
				Type:     schema.TypeInt,
				Required: true,
			},

			"family": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateRedisFamily,
			},

			"sku_name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateRedisSku,
			},

			"shard_count": {
				Type:     schema.TypeInt,
				Optional: true,
				// NOTE: this only applies to Premium SKU's
			},

			"enable_non_ssl_port": {
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
			},

			"redis_configuration": {
				Type:     schema.TypeMap,
				Optional: true,
			},

			"hostname": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"port": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"ssl_port": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmRedisCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).redisClient
	log.Printf("[INFO] preparing arguments for Azure ARM Redis creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)

	redisVersion := d.Get("redis_version").(string)
	enableNonSSLPort := d.Get("enable_non_ssl_port").(bool)

	capacity := int32(d.Get("capacity").(int))
	family := redis.SkuFamily(d.Get("family").(string))
	sku := redis.SkuName(d.Get("sku_name").(string))

	tags := d.Get("tags").(map[string]interface{})
	expandedTags := expandTags(tags)

	parameters := redis.CreateOrUpdateParameters{
		Name:     &name,
		Location: &location,
		Properties: &redis.Properties{
			EnableNonSslPort: &enableNonSSLPort,
			RedisVersion:     &redisVersion,
			Sku: &redis.Sku{
				Capacity: &capacity,
				Family:   family,
				Name:     sku,
			},
		},
		Tags: expandedTags,
	}

	if v, ok := d.GetOk("shard_count"); ok {
		shardCount := int32(v.(int))
		parameters.Properties.ShardCount = &shardCount
	}

	/*
		if v, ok := d.GetOk("redis_configuration"); ok {
			params := v.(map[string]interface{})

			redisConfiguration := make(map[string]*string, len(params))
			for key, val := range params {
				redisConfiguration[key] = struct {
					Value *string
				}{
					Value: val.(string),
				}
			}

			parameters.Properties.RedisConfiguration = &redisConfiguration
		}
	*/

	_, err := client.CreateOrUpdate(resGroup, name, parameters)
	if err != nil {
		return err
	}

	read, err := client.Get(resGroup, name)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Redis %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmRedisRead(d, meta)
}

func resourceArmRedisRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).redisClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["redis"]

	resp, err := client.Get(resGroup, name)
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure Redis %s: %s", name, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	d.Set("name", resp.Name)
	d.Set("resource_group_name", resGroup)
	//d.Set("location", azureRMNormalizeLocation(*resp.Location))

	if resp.Properties != nil {

		d.Set("redis_version", resp.Properties.RedisVersion)
		d.Set("enable_non_ssl_port", resp.Properties.EnableNonSslPort)

		if resp.Properties.Sku != nil {
			d.Set("capacity", resp.Properties.Sku.Capacity)
			d.Set("family", resp.Properties.Sku.Family)
			d.Set("sku_name", resp.Properties.Sku.Name)
		}

		/*
			if resp.Properties.ShardCount > 0 {
				d.Set("shard_count", resp.Properties.ShardCount)
			}
		*/
	}

	// TODO: Redis Configuation

	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmRedisDelete(d *schema.ResourceData, meta interface{}) error {
	redisClient := meta.(*ArmClient).redisClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["redis"]

	resp, err := redisClient.Delete(resGroup, name)

	if resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("Error issuing Azure ARM delete request of Redis Instance '%s': %s", name, err)
	}

	return nil
}

func validateRedisFamily(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	families := map[string]bool{
		"c": true,
		"p": true,
	}

	if !families[value] {
		errors = append(errors, fmt.Errorf("Redis Family can only be C or P"))
	}
	return
}

func validateRedisSku(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	skus := map[string]bool{
		"basic":    true,
		"standard": true,
		"premium":  true,
	}

	if !skus[value] {
		errors = append(errors, fmt.Errorf("Redis SKU can only be Basic, Standard or Premium"))
	}
	return
}
