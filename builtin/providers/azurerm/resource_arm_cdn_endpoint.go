package azurerm

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/cdn"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmCdnEndpoint() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmCdnEndpointCreate,
		Read:   resourceArmCdnEndpointRead,
		Update: resourceArmCdnEndpointUpdate,
		Delete: resourceArmCdnEndpointDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"profile_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"origin_host_header": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"is_http_allowed": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"is_https_allowed": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"origin": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"host_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"http_port": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},

						"https_port": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
					},
				},
				Set: resourceArmCdnEndpointOriginHash,
			},

			"origin_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"querystring_caching_behaviour": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "IgnoreQueryString",
				ValidateFunc: validateCdnEndpointQuerystringCachingBehaviour,
			},

			"content_types_to_compress": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"is_compression_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"host_name": &schema.Schema{
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

	properties := cdn.EndpointPropertiesCreateUpdateParameters{
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

	cdnEndpoint := cdn.EndpointCreateParameters{
		Location:   &location,
		Properties: &properties,
		Tags:       expandTags(tags),
	}

	resp, err := cdnEndpointsClient.Create(name, cdnEndpoint, profileName, resGroup)
	if err != nil {
		return err
	}

	d.SetId(*resp.ID)

	log.Printf("[DEBUG] Waiting for CDN Endpoint (%s) to become available", name)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted", "Updating", "Creating"},
		Target:  []string{"Succeeded"},
		Refresh: cdnEndpointStateRefreshFunc(client, resGroup, profileName, name),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for CDN Endpoint (%s) to become available: %s", name, err)
	}

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
	resp, err := cdnEndpointsClient.Get(name, profileName, resGroup)
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure CDN Endpoint %s: %s", name, err)
	}

	d.Set("name", resp.Name)
	d.Set("host_name", resp.Properties.HostName)
	d.Set("is_compression_enabled", resp.Properties.IsCompressionEnabled)
	d.Set("is_http_allowed", resp.Properties.IsHTTPAllowed)
	d.Set("is_https_allowed", resp.Properties.IsHTTPSAllowed)
	d.Set("querystring_caching_behaviour", resp.Properties.QueryStringCachingBehavior)
	if resp.Properties.OriginHostHeader != nil && *resp.Properties.OriginHostHeader != "" {
		d.Set("origin_host_header", resp.Properties.OriginHostHeader)
	}
	if resp.Properties.OriginPath != nil && *resp.Properties.OriginPath != "" {
		d.Set("origin_path", resp.Properties.OriginPath)
	}
	if resp.Properties.ContentTypesToCompress != nil && len(*resp.Properties.ContentTypesToCompress) > 0 {
		d.Set("content_types_to_compress", flattenAzureRMCdnEndpointContentTypes(resp.Properties.ContentTypesToCompress))
	}
	d.Set("origin", flattenAzureRMCdnEndpointOrigin(resp.Properties.Origins))

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

	properties := cdn.EndpointPropertiesCreateUpdateParameters{
		IsHTTPAllowed:              &http_allowed,
		IsHTTPSAllowed:             &https_allowed,
		IsCompressionEnabled:       &compression_enabled,
		QueryStringCachingBehavior: cdn.QueryStringCachingBehavior(caching_behaviour),
	}

	if d.HasChange("origin") {
		origins, originsErr := expandAzureRmCdnEndpointOrigins(d)
		if originsErr != nil {
			return fmt.Errorf("Error Building list of CDN Endpoint Origins: %s", originsErr)
		}
		if len(origins) > 0 {
			properties.Origins = &origins
		}
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
		Tags:       expandTags(newTags),
		Properties: &properties,
	}

	_, err := cdnEndpointsClient.Update(name, updateProps, profileName, resGroup)
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

	accResp, err := client.DeleteIfExists(name, profileName, resGroup)
	if err != nil {
		if accResp.StatusCode == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf("Error issuing AzureRM delete request for CDN Endpoint %q: %s", name, err)
	}
	_, err = pollIndefinitelyAsNeeded(client.Client, accResp.Response, http.StatusNotFound)
	if err != nil {
		return fmt.Errorf("Error polling for AzureRM delete request for CDN Endpoint %q: %s", name, err)
	}

	return err
}

func cdnEndpointStateRefreshFunc(client *ArmClient, resourceGroupName string, profileName string, name string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.cdnEndpointsClient.Get(name, profileName, resourceGroupName)
		if err != nil {
			return nil, "", fmt.Errorf("Error issuing read request in cdnEndpointStateRefreshFunc to Azure ARM for CDN Endpoint '%s' (RG: '%s'): %s", name, resourceGroupName, err)
		}
		return res, string(res.Properties.ProvisioningState), nil
	}
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
			https_port := v.(int)
			properties.HTTPSPort = &https_port

		}

		if v, ok := data["http_port"]; ok {
			http_port := v.(int)
			properties.HTTPPort = &http_port
		}

		name := data["name"].(string)

		origin := cdn.DeepCreatedOrigin{
			Name:       &name,
			Properties: &properties,
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
			"host_name": *i.Properties.HostName,
		}

		if i.Properties.HTTPPort != nil {
			l["http_port"] = *i.Properties.HTTPPort
		}
		if i.Properties.HTTPSPort != nil {
			l["https_port"] = *i.Properties.HTTPSPort
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
