package netapp

import (
	"fmt"
	"log"
	"strings"

	"github.com/candidpartners/occm-sdk-go/api/workenv"
	"github.com/candidpartners/occm-sdk-go/api/workenv/vsa"
	"github.com/candidpartners/occm-sdk-go/util"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceCloudVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudVolumeCreate,
		Read:   resourceCloudVolumeRead,
		Update: resourceCloudVolumeUpdate,
		Delete: resourceCloudVolumeDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"workenv_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"svm_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"aggregate_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					"nfs",
					"cifs",
				}, false),
			},
			"size": {
				Type:     schema.TypeFloat,
				Required: true,
				ForceNew: true,
			},
			"size_unit": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					"GB",
					"TB",
				}, true),
			},
			"initial_size": {
				Type:     schema.TypeFloat,
				Optional: true,
			},
			"initial_size_unit": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					"GB",
					"TB",
				}, true),
			},
			"snapshot_policy": {
				Type:     schema.TypeString,
				Required: true,
			},
			"export_policy": {
				Type:          schema.TypeList,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"share"},
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"share": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MinItems: 1,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"permission": {
							Type:     schema.TypeList,
							MinItems: 1,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:     schema.TypeString,
										Required: true,
										ValidateFunc: validation.StringInSlice([]string{
											"read",
											"change",
											"full_control",
											"no_access",
										}, false),
									},
									"users": {
										Type:     schema.TypeList,
										Required: true,
										MinItems: 1,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
								},
							},
						},
					},
				},
			},
			"thin_provisioning": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  true,
			},
			"compression": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  true,
			},
			"deduplication": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  true,
			},
			"max_num_disks_approved_to_add": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"verify_name_uniqueness": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"provider_volume_type": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "gp2",
				ValidateFunc: validation.StringInSlice([]string{
					"gp2",
					"st1",
					"io1",
					"sc1",
				}, false),
			},
			"iops": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"sync_to_s3": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"capacity_tier": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"create_aggregate_if_not_found": {
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

func buildVolumeQuoteRequest(d *schema.ResourceData) *vsa.VSAVolumeQuoteRequest {
	req := vsa.VSAVolumeQuoteRequest{
		WorkingEnvironmentId: d.Get("workenv_id").(string),
		SvmName:              d.Get("svm_name").(string),
		Name:                 d.Get("name").(string),
		Size: &workenv.Capacity{
			Size: d.Get("size").(float64),
			Unit: d.Get("size_unit").(string),
		},
		ThinProvisioning: d.Get("thin_provisioning").(bool),
	}

	if attr, ok := d.GetOk("aggregate_name"); ok {
		req.AggregateName = attr.(string)
	}

	if attrSize, okSize := d.GetOk("initial_size"); okSize {
		if attrUnit, okUnit := d.GetOk("initial_size_unit"); okUnit {
			req.InitialSize = &workenv.Capacity{
				Size: attrSize.(float64),
				Unit: attrUnit.(string),
			}
		}
	}

	if attr, ok := d.GetOk("verify_name_uniqueness"); ok {
		req.VerifyNameUniqueness = attr.(bool)
	}

	if attr, ok := d.GetOk("capacity_tier"); ok {
		req.CapacityTier = attr.(string)
	}

	if attr, ok := d.GetOk("provider_volume_type"); ok {
		req.ProviderVolumeType = attr.(string)
	}

	if attr, ok := d.GetOk("iops"); ok {
		if d.Get("provider_volume_type").(string) != "io1" {
			log.Printf("[INFO] IOPS only supported for io1 volume types, ignoring")
		} else {
			req.IOPS = attr.(int)
		}
	}

	return &req
}

func buildVolumeCreateRequest(d *schema.ResourceData) *vsa.VSAVolumeCreateRequest {
	req := vsa.VSAVolumeCreateRequest{
		WorkingEnvironmentId: d.Get("workenv_id").(string),
		SvmName:              d.Get("svm_name").(string),
		AggregateName:        d.Get("aggregate_name").(string),
		Name:                 d.Get("name").(string),
		Size: &workenv.Capacity{
			Size: d.Get("size").(float64),
			Unit: d.Get("size_unit").(string),
		},
		SnapshotPolicyName: d.Get("snapshot_policy").(string),
		ThinProvisioning:   d.Get("thin_provisioning").(bool),
		Compression:        d.Get("compression").(bool),
		Deduplication:      d.Get("deduplication").(bool),
	}

	if attrSize, okSize := d.GetOk("initial_size"); okSize {
		if attrUnit, okUnit := d.GetOk("initial_size_unit"); okUnit {
			req.InitialSize = &workenv.Capacity{
				Size: attrSize.(float64),
				Unit: attrUnit.(string),
			}
		}
	}

	volumeType := d.Get("type").(string)
	if volumeType == "nfs" {
		req.ExportPolicyInfo = parsePolicyInfo(d)
	} else {
		req.ShareInfo = parseCreateShareInfoRequest(d)
	}

	if attr, ok := d.GetOk("capacity_tier"); ok {
		req.CapacityTier = attr.(string)
	}

	if attr, ok := d.GetOk("provider_volume_type"); ok {
		req.ProviderVolumeType = attr.(string)
	}

	if attr, ok := d.GetOk("iops"); ok {
		if d.Get("provider_volume_type").(string) != "io1" {
			log.Printf("[INFO] IOPS only supported for io1 volume types, ignoring")
		} else {
			req.IOPS = attr.(int)
		}
	}

	if attr, ok := d.GetOkExists("max_num_disks_approved_to_add"); ok {
		req.MaxNumOfDisksApprovedToAdd = attr.(int)
	}

	if attr, ok := d.GetOk("sync_to_s3"); ok {
		req.SyncToS3 = attr.(bool)
	}

	return &req
}

func buildVolumeModifyRequest(d *schema.ResourceData) *workenv.VolumeModifyRequest {
	req := workenv.VolumeModifyRequest{}

	if d.HasChange("snapshot_policy") {
		log.Printf("[DEBUG] Detected snapshot policy change")
		req.SnapshotPolicyName = d.Get("snapshot_policy").(string)
	}

	volumeType := d.Get("type").(string)
	if volumeType == "nfs" {
		if d.HasChange("export_policy") {
			log.Printf("[DEBUG] Detected export policy changes")
			req.ExportPolicyInfo = parsePolicyInfo(d)
		}
	} else {
		if d.HasChange("share") {
			log.Printf("[DEBUG] Detected share changes")
			req.ShareInfo = parseShareInfo(d)
		}
	}

	return &req
}

func buildVolumeTierChangeRequest(d *schema.ResourceData) *workenv.ChangeVolumeTierRequest {
  var createAggregateIfNotFound bool
	if attr, ok := d.GetOkExists("create_aggregate_if_not_found"); ok {
		createAggregateIfNotFound = attr.(bool)
	} else {
		createAggregateIfNotFound = true
	}

  req := workenv.ChangeVolumeTierRequest{
		AggregateName: d.Get("aggregate_name").(string),
		NewAggregate:  createAggregateIfNotFound,
	}

	if attr, ok := d.GetOkExists("max_num_disks_approved_to_add"); d.HasChange("max_num_disks_approved_to_add") && ok {
		req.NumOfDisks = attr.(int)
	}

	if attr, ok := d.GetOk("provider_volume_type"); ok {
		req.NewDiskTypeName = attr.(string)
	}

	if attr, ok := d.GetOk("iops"); ok {
		if d.Get("provider_volume_type").(string) != "io1" {
			log.Printf("[INFO] IOPS only supported for io1 volume types, ignoring")
		} else {
			req.IOPS = attr.(int)
		}
	}

	if attr, ok := d.GetOk("capacity_tier"); d.HasChange("capacity_tier") && ok {
		req.NewCapacityTier = attr.(string)
	}

	return &req
}

func parsePolicyInfo(d *schema.ResourceData) *workenv.ExportPolicyInfo {
	policyInfo := workenv.ExportPolicyInfo{
		PolicyType: "none",
		IPs:        []string{},
	}

	if exportPolicy, ok := d.GetOk("export_policy"); ok {
		policyInfo.PolicyType = "custom"

		for _, ip := range exportPolicy.([]interface{}) {
			policyInfo.IPs = append(policyInfo.IPs, ip.(string))
		}
	}

	return &policyInfo
}

func parseCreateShareInfoRequest(d *schema.ResourceData) *workenv.CreateCIFSShareInfoRequest {
	share := d.Get("share").([]interface{})[0].(map[string]interface{})
	p := share["permission"].([]interface{})[0].(map[string]interface{})

	users := []string{}
	for _, user := range p["users"].([]interface{}) {
		users = append(users, user.(string))
	}

	permissions := workenv.CIFSShareUserPermissions{
		Permission: p["type"].(string),
		Users:      users,
	}

	shareInfo := workenv.CreateCIFSShareInfoRequest{
		ShareName:     share["name"].(string),
		AccessControl: permissions,
	}

	return &shareInfo
}

func parseShareInfo(d *schema.ResourceData) *workenv.CIFSShareInfo {
	share := d.Get("share").([]interface{})[0].(map[string]interface{})

	permissions := []workenv.CIFSShareUserPermissions{}
	for _, pp := range share["permission"].([]interface{}) {
		p := pp.(map[string]interface{})

		users := []string{}
		for _, user := range p["users"].([]interface{}) {
			users = append(users, user.(string))
		}

		permission := workenv.CIFSShareUserPermissions{
			Permission: p["type"].(string),
			Users:      users,
		}

		permissions = append(permissions, permission)
	}

	shareInfo := workenv.CIFSShareInfo{
		ShareName:         share["name"].(string),
		AccessControlList: permissions,
	}

	return &shareInfo
}

func processVolumeResourceData(d *schema.ResourceData, res *workenv.VolumeResponse, workenvId string) error {
	d.Set("name", res.Name)
	d.Set("svm_name", res.SvmName)
	d.Set("workenv_id", workenvId)
	if _, ok := d.GetOk("aggregate_name"); ok {
		// only set the aggregate name if it is explicitly provided in config
		d.Set("aggregate_name", res.AggregateName)
	}
	d.Set("size", res.Size.Size)
	d.Set("size_unit", res.Size.Unit)
	d.Set("thin_provisioning", res.ThinProvisioning)
	d.Set("compression", res.Compression)
	d.Set("deduplication", res.Deduplication)
	d.Set("snapshot_policy", res.SnapshotPolicy)
	d.Set("provider_volume_type", res.ProviderVolumeType)
	d.Set("export_policy", flattenExportPolicy(res.ExportPolicyInfo))
	d.Set("share", flattenShareInfos(res.ShareInfo))
	if _, ok := d.GetOk("export_policy"); ok {
		d.Set("type", "nfs")
	} else {
		d.Set("type", "cifs")
	}

	return nil
}

func flattenExportPolicy(policy *workenv.NamedExportPolicyInfo) []interface{} {
	if policy.PolicyType != "custom" {
		return nil
	}

	result := []interface{}{}
	for _, ip := range policy.IPs {
		result = append(result, ip)
	}
	return result
}

func flattenShareInfos(infos []workenv.CIFSShareInfo) []interface{} {
	shares := []interface{}{}

	for _, info := range infos {
		shares = append(shares, flattenShareInfo(&info))
	}

	return shares
}

func flattenShareInfo(info *workenv.CIFSShareInfo) map[string]interface{} {
	permissions := []interface{}{}
	for _, acl := range info.AccessControlList {
		permission := map[string]interface{}{
			"type":  acl.Permission,
			"users": acl.Users,
		}
		permissions = append(permissions, permission)
	}

	share := map[string]interface{}{
		"name":       info.ShareName,
		"permission": permissions,
	}

	return share
}

func resourceCloudVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	apis := meta.(*APIs)

	workenvId := d.Get("workenv_id").(string)
	workenv, err := GetWorkingEnvironmentById(apis, workenvId)
	if err != nil {
		return err
	}

	// prepare volume quote first
	quoteReq := buildVolumeQuoteRequest(d)

	log.Printf("[DEBUG] Requesting a quote for volume %s", quoteReq.Name)
	log.Printf("[DEBUG] Quote request: %s", util.ToString(quoteReq))

	// quote the volume creation
	var quoteRes *vsa.VSAVolumeQuoteResponse
	if workenv.IsHA {
		quoteRes, err = apis.AWSHAWorkingEnvironmentAPI.QuoteVolume(quoteReq)
	} else {
		quoteRes, err = apis.VSAWorkingEnvironmentAPI.QuoteVolume(quoteReq)
	}

	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Quote response: %s", util.ToString(quoteRes))

	// if the aggregate creation is not defined, use the one returned by the quote response
	// NOTE: this is temporarily broken as providing the value of "false" will
	// use the value provided in the quote response - this is due to the way
	// the GetOk method works, there is no way to differentiate between not having
	// the value and the value being set to "false"
	var createAggregateIfNotFound bool
	if attr, ok := d.GetOkExists("create_aggregate_if_not_found"); ok {
		createAggregateIfNotFound = attr.(bool)
	} else {
		createAggregateIfNotFound = quoteRes.NewAggregate
	}

	log.Printf("[INFO] Creating volume %s", quoteReq.Name)

	// create the actual volume request
	req := buildVolumeCreateRequest(d)

	// set the request aggregate
	if attr, ok := d.GetOk("aggregate_name"); ok {
		req.AggregateName = attr.(string)
	} else {
		req.AggregateName = quoteRes.AggregateName
	}

	// if no max volumes was provided, use one from the quote response
	if _, ok := d.GetOkExists("max_num_disks_approved_to_add"); !ok {
		req.MaxNumOfDisksApprovedToAdd = quoteRes.NumOfDisks
	}

	log.Printf("[DEBUG] Setting create aggregate if not found flag to %t", createAggregateIfNotFound)
	log.Printf("[DEBUG] Request: %s", util.ToString(req))

	// actual the volume creation
	var requestId string
	if workenv.IsHA {
		requestId, err = apis.AWSHAWorkingEnvironmentAPI.CreateVolume(createAggregateIfNotFound, req)
	} else {
		requestId, err = apis.VSAWorkingEnvironmentAPI.CreateVolume(createAggregateIfNotFound, req)
	}

	if err != nil {
		return err
	}

	if err = WaitForRequest(apis, requestId); err != nil {
		return err
	}

	log.Printf("[INFO] Volume %s created successfully", quoteReq.Name)

	// set the ID
	volType := "vsa"
	if workenv.IsHA {
		volType = "ha"
	}
	d.SetId(fmt.Sprintf("%s/%s/%s/%s", volType, workenvId, workenv.SvmName, req.Name))

	return resourceCloudVolumeRead(d, meta)
}

func resourceCloudVolumeRead(d *schema.ResourceData, meta interface{}) error {
	apis := meta.(*APIs)

	volumeType, workenvId, _, volumeName, isHA, err := splitId(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading %s volume %s for work env %s", strings.ToUpper(volumeType), volumeName, workenvId)

	// get volume data
	var res *workenv.VolumeResponse
	if isHA {
		res, err = apis.AWSHAWorkingEnvironmentAPI.GetVolume(workenvId, volumeName)
	} else {
		res, err = apis.VSAWorkingEnvironmentAPI.GetVolume(workenvId, volumeName)
	}

	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Response: %s", util.ToString(res))
	log.Printf("[DEBUG] Processing volume %s", volumeName)

	// set volume details in resource data
	processVolumeResourceData(d, res, workenvId)

	log.Printf("[INFO] Volume %s read successfully", volumeName)

	return nil
}

func resourceCloudVolumeUpdate(d *schema.ResourceData, meta interface{}) error {
	apis := meta.(*APIs)

  // NOTE: the create_aggregate_if_not_found attribute is not supported by the
  // underlying NetApp API and therefore not used in the update call

	volumeType, workenvId, svmName, volumeName, isHA, err := splitId(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Updating %s volume %s for work env %s", strings.ToUpper(volumeType), volumeName, workenvId)

	// check for data that can be updated on the volume
	volumeDataChanged := false
	if d.HasChange("snapshot_policy") || d.HasChange("export_policy") || d.HasChange("share") {
		volumeDataChanged = true
	}

	// check if the volume tier has changed
	volumeTierChanged := false
	if d.HasChange("aggregate_name") || d.HasChange("provider_volume_type") || d.HasChange("capacity_tier") {
		volumeTierChanged = true
	}

	if volumeDataChanged {
		err = updateVolumeData(d, apis, volumeType, workenvId, svmName, volumeName, isHA)
		if err != nil {
			return err
		}

		d.SetPartial("snapshot_policy")
		d.SetPartial("export_policy")
		d.SetPartial("share")
	}

	if volumeTierChanged {
		err = updateVolumeTier(d, meta, apis, volumeType, workenvId, svmName, volumeName, isHA)
		if err != nil {
			return err
		}

		d.SetPartial("aggregate_name")
		d.SetPartial("provider_volume_type")
		d.SetPartial("capacity_tier")
	}

	d.Partial(false)

	log.Printf("[INFO] Volume %s updated successfully", volumeName)

	return nil
}

func resourceCloudVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	apis := meta.(*APIs)

	volumeType, workenvId, svmName, volumeName, isHA, err := splitId(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Deleting %s volume %s for work env %s", strings.ToUpper(volumeType), volumeName, workenvId)

	var requestId string
	if isHA {
		requestId, err = apis.AWSHAWorkingEnvironmentAPI.DeleteVolume(workenvId, svmName, volumeName)
	} else {
		requestId, err = apis.VSAWorkingEnvironmentAPI.DeleteVolume(workenvId, svmName, volumeName)
	}

	if err != nil {
		return err
	}

	if err = WaitForRequest(apis, requestId); err != nil {
		return err
	}

	log.Printf("[INFO] Volume %s deleted successfully", volumeName)

	d.SetId("")

	return nil
}

func updateVolumeData(d *schema.ResourceData, apis *APIs, volumeType, workenvId, svmName, volumeName string, isHA bool) error {
	req := buildVolumeModifyRequest(d)

	log.Printf("[INFO] Modifying %s volume %s for work env %s", strings.ToUpper(volumeType), volumeName, workenvId)
	log.Printf("[DEBUG] Request: %s", util.ToString(req))

	var err error
	var requestId string
	if isHA {
		requestId, err = apis.AWSHAWorkingEnvironmentAPI.ModifyVolume(workenvId, svmName, volumeName, req)
	} else {
		requestId, err = apis.VSAWorkingEnvironmentAPI.ModifyVolume(workenvId, svmName, volumeName, req)
	}

	if err != nil {
		return err
	}

	if err = WaitForRequest(apis, requestId); err != nil {
		return err
	}

	log.Printf("[INFO] Volume %s modified successfully", volumeName)

	return nil
}

func updateVolumeTier(d *schema.ResourceData, meta interface{}, apis *APIs, volumeType, workenvId, svmName, volumeName string, isHA bool) error {
	log.Printf("[INFO] Modifying tier for %s volume %s for work env %s", strings.ToUpper(volumeType), volumeName, workenvId)

	// prepare volume quote first
	quoteReq := buildVolumeQuoteRequest(d)

	log.Printf("[DEBUG] Requesting tier change quote for volume %s", quoteReq.Name)
	log.Printf("[DEBUG] Quote request: %s", util.ToString(quoteReq))

	// quote the volume creation
	var err error
	var quoteRes *vsa.VSAVolumeQuoteResponse
	if isHA {
		quoteRes, err = apis.AWSHAWorkingEnvironmentAPI.QuoteVolume(quoteReq)
	} else {
		quoteRes, err = apis.VSAWorkingEnvironmentAPI.QuoteVolume(quoteReq)
	}

	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Quote response: %s", util.ToString(quoteRes))

	req := buildVolumeTierChangeRequest(d)

	// set the request aggregate
	if attr, ok := d.GetOk("aggregate_name"); ok {
		req.AggregateName = attr.(string)
	} else {
		req.AggregateName = quoteRes.AggregateName
	}

	// if the aggregate control flag is not set, use the quote response one
	if attr, ok := d.GetOkExists("create_aggregate_if_not_found"); ok {
		req.NewAggregate = attr.(bool)
	} else {
		req.NewAggregate = quoteRes.NewAggregate
	}

	// if no max volumes was provided, use one from the quote response
	if _, ok := d.GetOkExists("max_num_disks_approved_to_add"); !ok {
		req.NumOfDisks = quoteRes.NumOfDisks
	}

	log.Printf("[DEBUG] Request: %s", util.ToString(req))

	var requestId string
	if isHA {
		requestId, err = apis.AWSHAWorkingEnvironmentAPI.ChangeVolumeTier(workenvId, svmName, volumeName, req)
	} else {
		requestId, err = apis.VSAWorkingEnvironmentAPI.ChangeVolumeTier(workenvId, svmName, volumeName, req)
	}

	if err != nil {
		return err
	}

	if err = WaitForRequest(apis, requestId); err != nil {
		return err
	}

	log.Printf("[INFO] Tier for volume %s modified successfully", volumeName)

	return resourceCloudVolumeRead(d, meta)
}

func splitId(id string) (string, string, string, string, bool, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 4 {
		return "", "", "", "", false, fmt.Errorf("Invalid volume ID format: %s", id)
	}

	volumeType := parts[0]
	workenvId := parts[1]
	svmName := parts[2]
	volumeName := parts[3]
	isHA := false
	if volumeType == "ha" {
		isHA = true
	}

	return volumeType, workenvId, svmName, volumeName, isHA, nil
}
