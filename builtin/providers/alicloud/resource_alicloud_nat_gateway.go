package alicloud

import (
	"fmt"

	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"strings"
	"time"
)

func resourceAliyunNatGateway() *schema.Resource {
	return &schema.Resource{
		Create: resourceAliyunNatGatewayCreate,
		Read:   resourceAliyunNatGatewayRead,
		Update: resourceAliyunNatGatewayUpdate,
		Delete: resourceAliyunNatGatewayDelete,

		Schema: map[string]*schema.Schema{
			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"spec": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"bandwidth_package_ids": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"bandwidth_packages": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ip_count": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"bandwidth": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"zone": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Required: true,
				MaxItems: 4,
			},
		},
	}
}

func resourceAliyunNatGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).vpcconn

	args := &ecs.CreateNatGatewayArgs{
		RegionId: getRegion(d, meta),
		VpcId:    d.Get("vpc_id").(string),
		Spec:     d.Get("spec").(string),
	}

	bandwidthPackages := d.Get("bandwidth_packages").([]interface{})

	bandwidthPackageTypes := []ecs.BandwidthPackageType{}

	for _, e := range bandwidthPackages {
		pack := e.(map[string]interface{})
		bandwidthPackage := ecs.BandwidthPackageType{
			IpCount:   pack["ip_count"].(int),
			Bandwidth: pack["bandwidth"].(int),
		}
		if pack["zone"].(string) != "" {
			bandwidthPackage.Zone = pack["zone"].(string)
		}

		bandwidthPackageTypes = append(bandwidthPackageTypes, bandwidthPackage)
	}

	args.BandwidthPackage = bandwidthPackageTypes

	var name string
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	}

	args.Name = name

	if v, ok := d.GetOk("description"); ok {
		args.Description = v.(string)
	}
	resp, err := conn.CreateNatGateway(args)
	if err != nil {
		return fmt.Errorf("CreateNatGateway got error: %#v", err)
	}

	d.SetId(resp.NatGatewayId)

	return resourceAliyunNatGatewayRead(d, meta)
}

func resourceAliyunNatGatewayRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*AliyunClient)

	natGateway, err := client.DescribeNatGateway(d.Id())
	if err != nil {
		if notFoundError(err) {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", natGateway.Name)
	d.Set("spec", natGateway.Spec)
	d.Set("bandwidth_package_ids", strings.Join(natGateway.BandwidthPackageIds.BandwidthPackageId, ","))
	d.Set("description", natGateway.Description)
	d.Set("vpc_id", natGateway.VpcId)

	return nil
}

func resourceAliyunNatGatewayUpdate(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*AliyunClient)
	conn := client.vpcconn

	natGateway, err := client.DescribeNatGateway(d.Id())
	if err != nil {
		return err
	}

	d.Partial(true)
	attributeUpdate := false
	args := &ecs.ModifyNatGatewayAttributeArgs{
		RegionId:     natGateway.RegionId,
		NatGatewayId: natGateway.NatGatewayId,
	}

	if d.HasChange("name") {
		d.SetPartial("name")
		var name string
		if v, ok := d.GetOk("name"); ok {
			name = v.(string)
		} else {
			return fmt.Errorf("cann't change name to empty string")
		}
		args.Name = name

		attributeUpdate = true
	}

	if d.HasChange("description") {
		d.SetPartial("description")
		var description string
		if v, ok := d.GetOk("description"); ok {
			description = v.(string)
		} else {
			return fmt.Errorf("can to change description to empty string")
		}

		args.Description = description

		attributeUpdate = true
	}

	if attributeUpdate {
		if err := conn.ModifyNatGatewayAttribute(args); err != nil {
			return err
		}
	}

	if d.HasChange("spec") {
		d.SetPartial("spec")
		var spec ecs.NatGatewaySpec
		if v, ok := d.GetOk("spec"); ok {
			spec = ecs.NatGatewaySpec(v.(string))
		} else {
			// set default to small spec
			spec = ecs.NatGatewaySmallSpec
		}

		args := &ecs.ModifyNatGatewaySpecArgs{
			RegionId:     natGateway.RegionId,
			NatGatewayId: natGateway.NatGatewayId,
			Spec:         spec,
		}

		err := conn.ModifyNatGatewaySpec(args)
		if err != nil {
			return fmt.Errorf("%#v %#v", err, *args)
		}

	}
	d.Partial(false)

	return resourceAliyunNatGatewayRead(d, meta)
}

func resourceAliyunNatGatewayDelete(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*AliyunClient)
	conn := client.vpcconn

	return resource.Retry(5*time.Minute, func() *resource.RetryError {

		packages, err := conn.DescribeBandwidthPackages(&ecs.DescribeBandwidthPackagesArgs{
			RegionId:     getRegion(d, meta),
			NatGatewayId: d.Id(),
		})
		if err != nil {
			log.Printf("[ERROR] Describe bandwidth package is failed, natGateway Id: %s", d.Id())
			return resource.NonRetryableError(err)
		}

		retry := false
		for _, pack := range packages {
			err = conn.DeleteBandwidthPackage(&ecs.DeleteBandwidthPackageArgs{
				RegionId:           getRegion(d, meta),
				BandwidthPackageId: pack.BandwidthPackageId,
			})

			if err != nil {
				er, _ := err.(*common.Error)
				if er.ErrorResponse.Code == NatGatewayInvalidRegionId {
					log.Printf("[ERROR] Delete bandwidth package is failed, bandwidthPackageId: %#v", pack.BandwidthPackageId)
					return resource.NonRetryableError(err)
				}
				retry = true
			}
		}

		if retry {
			return resource.RetryableError(fmt.Errorf("Bandwidth package in use - trying again while it is deleted."))
		}

		args := &ecs.DeleteNatGatewayArgs{
			RegionId:     client.Region,
			NatGatewayId: d.Id(),
		}

		err = conn.DeleteNatGateway(args)
		if err != nil {
			er, _ := err.(*common.Error)
			if er.ErrorResponse.Code == DependencyViolationBandwidthPackages {
				return resource.RetryableError(fmt.Errorf("NatGateway in use - trying again while it is deleted."))
			}
		}

		describeArgs := &ecs.DescribeNatGatewaysArgs{
			RegionId:     client.Region,
			NatGatewayId: d.Id(),
		}
		gw, _, gwErr := conn.DescribeNatGateways(describeArgs)

		if gwErr != nil {
			log.Printf("[ERROR] Describe NatGateways failed.")
			return resource.NonRetryableError(gwErr)
		} else if gw == nil || len(gw) < 1 {
			return nil
		}

		return resource.RetryableError(fmt.Errorf("NatGateway in use - trying again while it is deleted."))
	})
}
