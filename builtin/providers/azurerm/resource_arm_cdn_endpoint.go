package azurerm

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/cdn"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmCdnEndpoint() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmCdnEndpointCreate,
		Read:   resourceArmCdnEndpointRead,
		Update: resourceArmCdnEndpointUpdate,
		Delete: resourceArmCdnEndpointDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": locationSchema(),

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"profile_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"origin_host_header": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"is_http_allowed": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"is_https_allowed": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"origin": {
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"host_name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"http_port": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},

						"https_port": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
					},
				},
				Set: resourceArmCdnEndpointOriginHash,
			},

			"origin_path": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"querystring_caching_behaviour": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "IgnoreQueryString",
				ValidateFunc: validateCdnEndpointQuerystringCachingBehaviour,
			},

			"content_types_to_compress": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Set: schema.HashString,
			},

			"is_compression_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"host_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmCdnEndpointCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	cdnEndpointsClient := client.cdnEndpointsClient

	log.Printf("[INFO] preparing arguments for Azure ARM CDN EndPoint creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	profileName := d.Get("profile_name").(string)
	http_allowed := d.Get("is_http_allowed").(bool)
	https_allowed := d.Get("is_https_allowed").(bool)
	compression_enabled := d.Get("is_compression_enabled").(bool)
	caching_behaviour := d.Get("querystring_caching_behaviour").(string)
	tags := d.Get("tags").(map[string]interface{})

	properties := cdn.EndpointProperties{
		IsHTTPAllowed:              &http_allowed,
		IsHTTPSAllowed:             &https_allowed,
		IsCompressionEnabled:       &compression_enabled,
		QueryStringCachingBehavior: cdn.QueryStringCachingBehavior(caching_behaviour),
	}

	origins, originsErr := expandAzureRmCdnEndpointOrigins(d)
	if originsErr != nil {
		return fmt.Errorf("Error Building list of CDN Endpoint Origins: %s", originsErr)
	}
	if len(origins) > 0 {
		properties.Origins = &origins
	}

	if v, ok := d.GetOk("origin_host_header"); ok {
		host_header := v.(string)
		properties.OriginHostHeader = &host_header
	}

	if v, ok := d.GetOk("origin_path"); ok {
		origin_path := v.(string)
		properties.OriginPath = &origin_path
	}

	if v, ok := d.GetOk("content_types_to_compress"); ok {
		var content_types []string
		ctypes := v.(*schema.Set).List()
		for _, ct := range ctypes {
			str := ct.(string)
			content_types = append(content_types, str)
		}

		properties.ContentTypesToCompress = &content_types
	}

	cdnEndpoint := cdn.Endpoint{
		Location:           &location,
		EndpointProperties: &properties,
		Tags:               expandTags(tags),
	}

	_, error := cdnEndpointsClient.Create(resGroup, profileName, name, cdnEndpoint, make(<-chan struct{}))
	err := <-error
	if err != nil {
		return err
	}

	read, err := cdnEndpointsClient.Get(resGroup, profileName, name)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read CND Endpoint %s/%s (resource group %s) ID", profileName, name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmCdnEndpointRead(d, meta)
}

func resourceArmCdnEndpointRead(d *schema.ResourceData, meta interface{}) error {
	cdnEndpointsClient := meta.(*ArmClient).cdnEndpointsClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["endpoints"]
	profileName := id.Path["profiles"]
	if profileName == "" {
		profileName = id.Path["Profiles"]
	}
	log.Printf("[INFO] Trying to find the AzureRM CDN Endpoint %s (Profile: %s, RG: %s)", name, profileName, resGroup)
	resp, err := cdnEndpointsClient.Get(resGroup, profileName, name)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error making Read request on Azure CDN Endpoint %s: %s", name, err)
	}

	d.Set("name", resp.Name)
	d.Set("resource_group_name", resGroup)
	d.Set("location", azureRMNormalizeLocation(*resp.Location))
	d.Set("profile_name", profileName)
	d.Set("host_name", resp.EndpointProperties.HostName)
	d.Set("is_compression_enabled", resp.EndpointProperties.IsCompressionEnabled)
	d.Set("is_http_allowed", resp.EndpointProperties.IsHTTPAllowed)
	d.Set("is_https_allowed", resp.EndpointProperties.IsHTTPSAllowed)
	d.Set("querystring_caching_behaviour", resp.EndpointProperties.QueryStringCachingBehavior)
	if resp.EndpointProperties.OriginHostHeader != nil && *resp.EndpointProperties.OriginHostHeader != "" {
		d.Set("origin_host_header", resp.EndpointProperties.OriginHostHeader)
	}
	if resp.EndpointProperties.OriginPath != nil && *resp.EndpointProperties.OriginPath != "" {
		d.Set("origin_path", resp.EndpointProperties.OriginPath)
	}
	if resp.EndpointProperties.ContentTypesToCompress != nil {
		d.Set("content_types_to_compress", flattenAzureRMCdnEndpointContentTypes(resp.EndpointProperties.ContentTypesToCompress))
	}
	d.Set("origin", flattenAzureRMCdnEndpointOrigin(resp.EndpointProperties.Origins))

	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmCdnEndpointUpdate(d *schema.ResourceData, meta interface{}) error {
	cdnEndpointsClient := meta.(*ArmClient).cdnEndpointsClient

	if !d.HasChange("tags") {
		return nil
	}

	name := d.Get("name").(string)
	resGroup := d.Get("resource_group_name").(string)
	profileName := d.Get("profile_name").(string)
	http_allowed := d.Get("is_http_allowed").(bool)
	https_allowed := d.Get("is_https_allowed").(bool)
	compression_enabled := d.Get("is_compression_enabled").(bool)
	caching_behaviour := d.Get("querystring_caching_behaviour").(string)
	newTags := d.Get("tags").(map[string]interface{})

	properties := cdn.EndpointPropertiesUpdateParameters{
		IsHTTPAllowed:              &http_allowed,
		IsHTTPSAllowed:             &https_allowed,
		IsCompressionEnabled:       &compression_enabled,
		QueryStringCachingBehavior: cdn.QueryStringCachingBehavior(caching_behaviour),
	}

	if d.HasChange("origin_host_header") {
		host_header := d.Get("origin_host_header").(string)
		properties.OriginHostHeader = &host_header
	}

	if d.HasChange("origin_path") {
		origin_path := d.Get("origin_path").(string)
		properties.OriginPath = &origin_path
	}

	if d.HasChange("content_types_to_compress") {
		var content_types []string
		ctypes := d.Get("content_types_to_compress").(*schema.Set).List()
		for _, ct := range ctypes {
			str := ct.(string)
			content_types = append(content_types, str)
		}

		properties.ContentTypesToCompress = &content_types
	}

	updateProps := cdn.EndpointUpdateParameters{
		Tags: expandTags(newTags),
		EndpointPropertiesUpdateParameters: &properties,
	}

	_, error := cdnEndpointsClient.Update(resGroup, profileName, name, updateProps, make(<-chan struct{}))
	err := <-error
	if err != nil {
		return fmt.Errorf("Error issuing Azure ARM update request to update CDN Endpoint %q: %s", name, err)
	}

	return resourceArmCdnEndpointRead(d, meta)
}

func resourceArmCdnEndpointDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).cdnEndpointsClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	profileName := id.Path["profiles"]
	if profileName == "" {
		profileName = id.Path["Profiles"]
	}
	name := id.Path["endpoints"]

	accResp, error := client.Delete(resGroup, profileName, name, make(<-chan struct{}))
	resp := <-accResp
	err = <-error
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf("Error issuing AzureRM delete request for CDN Endpoint %q: %s", name, err)
	}

	return nil
}

func validateCdnEndpointQuerystringCachingBehaviour(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	cachingTypes := map[string]bool{
		"ignorequerystring": true,
		"bypasscaching":     true,
		"usequerystring":    true,
	}

	if !cachingTypes[value] {
		errors = append(errors, fmt.Errorf("CDN Endpoint querystringCachingBehaviours can only be IgnoreQueryString, BypassCaching or UseQueryString"))
	}
	return
}

func resourceArmCdnEndpointOriginHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["host_name"].(string)))

	return hashcode.String(buf.String())
}

func expandAzureRmCdnEndpointOrigins(d *schema.ResourceData) ([]cdn.DeepCreatedOrigin, error) {
	configs := d.Get("origin").(*schema.Set).List()
	origins := make([]cdn.DeepCreatedOrigin, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		host_name := data["host_name"].(string)

		properties := cdn.DeepCreatedOriginProperties{
			HostName: &host_name,
		}

		if v, ok := data["https_port"]; ok {
			https_port := int32(v.(int))
			properties.HTTPSPort = &https_port

		}

		if v, ok := data["http_port"]; ok {
			http_port := int32(v.(int))
			properties.HTTPPort = &http_port
		}

		name := data["name"].(string)

		origin := cdn.DeepCreatedOrigin{
			Name: &name,
			DeepCreatedOriginProperties: &properties,
		}

		origins = append(origins, origin)
	}

	return origins, nil
}

func flattenAzureRMCdnEndpointOrigin(list *[]cdn.DeepCreatedOrigin) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(*list))
	for _, i := range *list {
		l := map[string]interface{}{
			"name":      *i.Name,
			"host_name": *i.DeepCreatedOriginProperties.HostName,
		}

		if i.DeepCreatedOriginProperties.HTTPPort != nil {
			l["http_port"] = *i.DeepCreatedOriginProperties.HTTPPort
		}
		if i.DeepCreatedOriginProperties.HTTPSPort != nil {
			l["https_port"] = *i.DeepCreatedOriginProperties.HTTPSPort
		}
		result = append(result, l)
	}
	return result
}

func flattenAzureRMCdnEndpointContentTypes(list *[]string) []interface{} {
	vs := make([]interface{}, 0, len(*list))
	for _, v := range *list {
		vs = append(vs, v)
	}
	return vs
}
