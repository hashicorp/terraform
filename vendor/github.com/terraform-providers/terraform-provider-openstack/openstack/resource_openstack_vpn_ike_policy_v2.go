package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/vpnaas/ikepolicies"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceIKEPolicyV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceIKEPolicyV2Create,
		Read:   resourceIKEPolicyV2Read,
		Update: resourceIKEPolicyV2Update,
		Delete: resourceIKEPolicyV2Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"auth_algorithm": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "sha1",
			},
			"encryption_algorithm": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "aes-128",
			},
			"pfs": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "group5",
			},
			"phase1_negotiation_mode": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "main",
			},
			"ike_version": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "v1",
			},
			"lifetime": {
				Type:     schema.TypeSet,
				Computed: true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"units": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
						},
						"value": {
							Type:     schema.TypeInt,
							Computed: true,
							Optional: true,
						},
					},
				},
			},
			"tenant_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"value_specs": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceIKEPolicyV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	lifetime := resourceIKEPolicyV2LifetimeCreateOpts(d.Get("lifetime").(*schema.Set))
	authAlgorithm := resourceIKEPolicyV2AuthAlgorithm(d.Get("auth_algorithm").(string))
	encryptionAlgorithm := resourceIKEPolicyV2EncryptionAlgorithm(d.Get("encryption_algorithm").(string))
	pfs := resourceIKEPolicyV2PFS(d.Get("pfs").(string))
	ikeVersion := resourceIKEPolicyV2IKEVersion(d.Get("ike_version").(string))
	phase1NegotationMode := resourceIKEPolicyV2Phase1NegotiationMode(d.Get("phase1_negotiation_mode").(string))

	opts := IKEPolicyCreateOpts{
		ikepolicies.CreateOpts{
			Name:                  d.Get("name").(string),
			Description:           d.Get("description").(string),
			TenantID:              d.Get("tenant_id").(string),
			Lifetime:              &lifetime,
			AuthAlgorithm:         authAlgorithm,
			EncryptionAlgorithm:   encryptionAlgorithm,
			PFS:                   pfs,
			IKEVersion:            ikeVersion,
			Phase1NegotiationMode: phase1NegotationMode,
		},
		MapValueSpecs(d),
	}
	log.Printf("[DEBUG] Create IKE policy: %#v", opts)

	policy, err := ikepolicies.Create(networkingClient, opts).Extract()
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"PENDING_CREATE"},
		Target:     []string{"ACTIVE"},
		Refresh:    waitForIKEPolicyCreation(networkingClient, policy.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      0,
		MinTimeout: 2 * time.Second,
	}
	_, err = stateConf.WaitForState()

	log.Printf("[DEBUG] IKE policy created: %#v", policy)

	d.SetId(policy.ID)

	return resourceIKEPolicyV2Read(d, meta)
}

func resourceIKEPolicyV2Read(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Retrieve information about IKE policy: %s", d.Id())

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	policy, err := ikepolicies.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "IKE policy")
	}

	log.Printf("[DEBUG] Read OpenStack IKE Policy %s: %#v", d.Id(), policy)

	d.Set("name", policy.Name)
	d.Set("description", policy.Description)
	d.Set("auth_algorithm", policy.AuthAlgorithm)
	d.Set("encryption_algorithm", policy.EncryptionAlgorithm)
	d.Set("tenant_id", policy.TenantID)
	d.Set("pfs", policy.PFS)
	d.Set("phase1_negotiation_mode", policy.Phase1NegotiationMode)
	d.Set("ike_version", policy.IKEVersion)
	d.Set("region", GetRegion(d, config))

	// Set the lifetime
	var lifetimeMap map[string]interface{}
	lifetimeMap = make(map[string]interface{})
	lifetimeMap["units"] = policy.Lifetime.Units
	lifetimeMap["value"] = policy.Lifetime.Value
	var lifetime []map[string]interface{}
	lifetime = append(lifetime, lifetimeMap)
	if err := d.Set("lifetime", &lifetime); err != nil {
		log.Printf("[WARN] unable to set IKE policy lifetime")
	}

	return nil
}

func resourceIKEPolicyV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	opts := ikepolicies.UpdateOpts{}

	var hasChange bool

	if d.HasChange("name") {
		name := d.Get("name").(string)
		opts.Name = &name
		hasChange = true
	}

	if d.HasChange("description") {
		description := d.Get("description").(string)
		opts.Description = &description
		hasChange = true
	}

	if d.HasChange("pfs") {
		opts.PFS = resourceIKEPolicyV2PFS(d.Get("pfs").(string))
		hasChange = true
	}
	if d.HasChange("auth_algorithm") {
		opts.AuthAlgorithm = resourceIKEPolicyV2AuthAlgorithm(d.Get("auth_algorithm").(string))
		hasChange = true
	}
	if d.HasChange("encryption_algorithm") {
		opts.EncryptionAlgorithm = resourceIKEPolicyV2EncryptionAlgorithm(d.Get("encryption_algorithm").(string))
		hasChange = true
	}
	if d.HasChange("phase_1_negotiation_mode") {
		opts.Phase1NegotiationMode = resourceIKEPolicyV2Phase1NegotiationMode(d.Get("phase_1_negotiation_mode").(string))
		hasChange = true
	}
	if d.HasChange("ike_version") {
		opts.IKEVersion = resourceIKEPolicyV2IKEVersion(d.Get("ike_version").(string))
		hasChange = true
	}

	if d.HasChange("lifetime") {
		lifetime := resourceIKEPolicyV2LifetimeUpdateOpts(d.Get("lifetime").(*schema.Set))
		opts.Lifetime = &lifetime
		hasChange = true
	}

	log.Printf("[DEBUG] Updating IKE policy with id %s: %#v", d.Id(), opts)

	if hasChange {
		err = ikepolicies.Update(networkingClient, d.Id(), opts).Err
		if err != nil {
			return err
		}
		stateConf := &resource.StateChangeConf{
			Pending:    []string{"PENDING_UPDATE"},
			Target:     []string{"ACTIVE"},
			Refresh:    waitForIKEPolicyUpdate(networkingClient, d.Id()),
			Timeout:    d.Timeout(schema.TimeoutCreate),
			Delay:      0,
			MinTimeout: 2 * time.Second,
		}
		if _, err = stateConf.WaitForState(); err != nil {
			return err
		}
	}

	return resourceIKEPolicyV2Read(d, meta)
}

func resourceIKEPolicyV2Delete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Destroy IKE policy: %s", d.Id())

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     []string{"DELETED"},
		Refresh:    waitForIKEPolicyDeletion(networkingClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      0,
		MinTimeout: 2 * time.Second,
	}

	if _, err = stateConf.WaitForState(); err != nil {
		return err
	}

	return nil
}

func waitForIKEPolicyDeletion(networkingClient *gophercloud.ServiceClient, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		err := ikepolicies.Delete(networkingClient, id).Err
		if err == nil {
			return "", "DELETED", nil
		}

		return nil, "ACTIVE", err
	}
}

func waitForIKEPolicyCreation(networkingClient *gophercloud.ServiceClient, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		policy, err := ikepolicies.Get(networkingClient, id).Extract()
		if err != nil {
			return "", "PENDING_CREATE", nil
		}
		return policy, "ACTIVE", nil
	}
}

func waitForIKEPolicyUpdate(networkingClient *gophercloud.ServiceClient, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		policy, err := ikepolicies.Get(networkingClient, id).Extract()
		if err != nil {
			return "", "PENDING_UPDATE", nil
		}
		return policy, "ACTIVE", nil
	}
}

func resourceIKEPolicyV2AuthAlgorithm(v string) ikepolicies.AuthAlgorithm {
	var authAlgorithm ikepolicies.AuthAlgorithm
	switch v {
	case "sha1":
		authAlgorithm = ikepolicies.AuthAlgorithmSHA1
	case "sha256":
		authAlgorithm = ikepolicies.AuthAlgorithmSHA256
	case "sha384":
		authAlgorithm = ikepolicies.AuthAlgorithmSHA384
	case "sha512":
		authAlgorithm = ikepolicies.AuthAlgorithmSHA512
	}

	return authAlgorithm
}

func resourceIKEPolicyV2EncryptionAlgorithm(v string) ikepolicies.EncryptionAlgorithm {
	var encryptionAlgorithm ikepolicies.EncryptionAlgorithm
	switch v {
	case "3des":
		encryptionAlgorithm = ikepolicies.EncryptionAlgorithm3DES
	case "aes-128":
		encryptionAlgorithm = ikepolicies.EncryptionAlgorithmAES128
	case "aes-192":
		encryptionAlgorithm = ikepolicies.EncryptionAlgorithmAES192
	case "aes-256":
		encryptionAlgorithm = ikepolicies.EncryptionAlgorithmAES256
	}

	return encryptionAlgorithm
}

func resourceIKEPolicyV2PFS(v string) ikepolicies.PFS {
	var pfs ikepolicies.PFS
	switch v {
	case "group5":
		pfs = ikepolicies.PFSGroup5
	case "group2":
		pfs = ikepolicies.PFSGroup2
	case "group14":
		pfs = ikepolicies.PFSGroup14
	}
	return pfs
}

func resourceIKEPolicyV2IKEVersion(v string) ikepolicies.IKEVersion {
	var ikeVersion ikepolicies.IKEVersion
	switch v {
	case "v1":
		ikeVersion = ikepolicies.IKEVersionv1
	case "v2":
		ikeVersion = ikepolicies.IKEVersionv2
	}
	return ikeVersion
}

func resourceIKEPolicyV2Phase1NegotiationMode(v string) ikepolicies.Phase1NegotiationMode {
	var phase1NegotiationMode ikepolicies.Phase1NegotiationMode
	switch v {
	case "main":
		phase1NegotiationMode = ikepolicies.Phase1NegotiationModeMain
	}
	return phase1NegotiationMode
}

func resourceIKEPolicyV2Unit(v string) ikepolicies.Unit {
	var unit ikepolicies.Unit
	switch v {
	case "kilobytes":
		unit = ikepolicies.UnitKilobytes
	case "seconds":
		unit = ikepolicies.UnitSeconds
	}
	return unit
}

func resourceIKEPolicyV2LifetimeCreateOpts(d *schema.Set) ikepolicies.LifetimeCreateOpts {
	lifetimeCreateOpts := ikepolicies.LifetimeCreateOpts{}

	rawPairs := d.List()
	for _, raw := range rawPairs {
		rawMap := raw.(map[string]interface{})
		lifetimeCreateOpts.Units = resourceIKEPolicyV2Unit(rawMap["units"].(string))

		value := rawMap["value"].(int)
		lifetimeCreateOpts.Value = value
	}
	return lifetimeCreateOpts

}

func resourceIKEPolicyV2LifetimeUpdateOpts(d *schema.Set) ikepolicies.LifetimeUpdateOpts {
	lifetimeUpdateOpts := ikepolicies.LifetimeUpdateOpts{}

	rawPairs := d.List()
	for _, raw := range rawPairs {
		rawMap := raw.(map[string]interface{})
		lifetimeUpdateOpts.Units = resourceIKEPolicyV2Unit(rawMap["units"].(string))

		value := rawMap["value"].(int)
		lifetimeUpdateOpts.Value = value
	}
	return lifetimeUpdateOpts

}
