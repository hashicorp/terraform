package ibmcloud

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/filter"
	"github.com/softlayer/softlayer-go/helpers/hardware"
	"github.com/softlayer/softlayer-go/helpers/location"
	"github.com/softlayer/softlayer-go/helpers/product"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
	"github.com/softlayer/softlayer-go/sl"
)

const (
	additionalServicesPackageType            = "ADDITIONAL_SERVICES"
	additionalServicesNetworkVlanPackageType = "ADDITIONAL_SERVICES_NETWORK_VLAN"

	vlanMask = "id,name,primaryRouter[datacenter[name]],primaryRouter[hostname],vlanNumber," +
		"billingItem[recurringFee],guestNetworkComponentCount,subnets[networkIdentifier,cidr,subnetType]"

	vlanTypePublic  = "PUBLIC"
	vlanTypePrivate = "PRIVATE"
)

func resourceIBMCloudInfraVlan() *schema.Resource {
	return &schema.Resource{
		Create:   resourceIBMCloudInfraVlanCreate,
		Read:     resourceIBMCloudInfraVlanRead,
		Update:   resourceIBMCloudInfraVlanUpdate,
		Delete:   resourceIBMCloudInfraVlanDelete,
		Exists:   resourceIBMCloudInfraVlanExists,
		Importer: &schema.ResourceImporter{},

		Schema: map[string]*schema.Schema{
			"datacenter": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errs []error) {
					vlanType := v.(string)
					if vlanType != "PRIVATE" && vlanType != "PUBLIC" {
						errs = append(errs, errors.New(
							"vlan type should be either 'PRIVATE' or 'PUBLIC'"))
					}
					return
				},
			},
			"subnet_size": {
				Type:         schema.TypeInt,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateSubnetSize,
			},

			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateVlanName,
			},

			"router_hostname": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateRouterHostname,
			},

			"vlan_number": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"softlayer_managed": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"child_resource_count": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"subnets": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnet": {
							Type:     schema.TypeString,
							Required: true,
						},
						"subnet_type": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceIBMCloudInfraVlanCreate(d *schema.ResourceData, meta interface{}) error {
	sess := meta.(ClientSession).SoftLayerSession()
	vlanType := d.Get("type").(string)
	if router, ok := d.GetOk("route_hostname"); ok {
		if (vlanType == vlanTypePrivate && strings.HasPrefix(router.(string), "fcr")) ||
			(vlanType == vlanTypePublic && strings.HasPrefix(router.(string), "bcr")) {
			return fmt.Errorf("Error creating vlan: mismatch between vlan_type '%s' and router_hostname '%s'", vlanType, router.(string))
		}
	}

	// Find price items with AdditionalServicesNetworkVlan
	productOrderContainer, err := buildVlanProductOrderContainer(d, sess, additionalServicesNetworkVlanPackageType)
	if err != nil {
		// Find price items with AdditionalServices
		productOrderContainer, err = buildVlanProductOrderContainer(d, sess, additionalServicesPackageType)
		if err != nil {
			return fmt.Errorf("Error creating vlan: %s", err)
		}
	}

	log.Println("[INFO] Creating vlan")

	receipt, err := services.GetProductOrderService(sess).
		PlaceOrder(productOrderContainer, sl.Bool(false))
	if err != nil {
		return fmt.Errorf("Error during creation of vlan: %s", err)
	}

	vlan, err := findVlanByOrderID(sess, *receipt.OrderId)

	if name, ok := d.GetOk("name"); ok {
		_, err = services.GetNetworkVlanService(sess).
			Id(*vlan.Id).EditObject(&datatypes.Network_Vlan{Name: sl.String(name.(string))})
		if err != nil {
			return fmt.Errorf("Error updating vlan: %s", err)
		}
	}

	d.SetId(fmt.Sprintf("%d", *vlan.Id))
	return resourceIBMCloudInfraVlanRead(d, meta)
}

func resourceIBMCloudInfraVlanRead(d *schema.ResourceData, meta interface{}) error {
	sess := meta.(ClientSession).SoftLayerSession()
	service := services.GetNetworkVlanService(sess)

	vlanID, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("Not a valid vlan ID, must be an integer: %s", err)
	}

	vlan, err := service.Id(vlanID).Mask(vlanMask).GetObject()

	if err != nil {
		return fmt.Errorf("Error retrieving vlan: %s", err)
	}

	d.Set("vlan_number", vlan.VlanNumber)
	d.Set("child_resource_count", vlan.GuestNetworkComponentCount)
	d.Set("name", vlan.Name)

	if vlan.PrimaryRouter != nil {
		d.Set("router_hostname", vlan.PrimaryRouter.Hostname)
		if strings.HasPrefix(*vlan.PrimaryRouter.Hostname, "fcr") {
			d.Set("type", vlanTypePublic)
		} else {
			d.Set("type", vlanTypePrivate)
		}
		if vlan.PrimaryRouter.Datacenter != nil {
			d.Set("datacenter", vlan.PrimaryRouter.Datacenter.Name)
		}
	}

	d.Set("softlayer_managed", vlan.BillingItem == nil)

	// Subnets
	subnets := make([]map[string]interface{}, 0)

	for _, elem := range vlan.Subnets {
		subnet := make(map[string]interface{})
		subnet["subnet"] = fmt.Sprintf("%s/%s", *elem.NetworkIdentifier, strconv.Itoa(*elem.Cidr))
		subnet["subnet_type"] = *elem.SubnetType
		subnets = append(subnets, subnet)
	}
	d.Set("subnets", subnets)

	if vlan.Subnets != nil && len(vlan.Subnets) > 0 {
		d.Set("subnet_size", 1<<(uint)(32-*vlan.Subnets[0].Cidr))
	} else {
		d.Set("subnet_size", 0)
	}

	return nil
}

func resourceIBMCloudInfraVlanUpdate(d *schema.ResourceData, meta interface{}) error {
	sess := meta.(ClientSession).SoftLayerSession()
	service := services.GetNetworkVlanService(sess)

	vlanID, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("Not a valid vlan ID, must be an integer: %s", err)
	}

	opts := datatypes.Network_Vlan{}

	if d.HasChange("name") {
		opts.Name = sl.String(d.Get("name").(string))
	}

	_, err = service.Id(vlanID).EditObject(&opts)

	if err != nil {
		return fmt.Errorf("Error updating vlan: %s", err)
	}
	return resourceIBMCloudInfraVlanRead(d, meta)
}

func resourceIBMCloudInfraVlanDelete(d *schema.ResourceData, meta interface{}) error {
	sess := meta.(ClientSession).SoftLayerSession()
	service := services.GetNetworkVlanService(sess)

	vlanID, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("Not a valid vlan ID, must be an integer: %s", err)
	}

	billingItem, err := service.Id(vlanID).GetBillingItem()
	if err != nil {
		return fmt.Errorf("Error deleting vlan: %s", err)
	}

	// VLANs which don't have billing items are managed by SoftLayer. They can't be deleted by
	// users. If a target VLAN doesn't have a billing item, the function will return nil without
	// errors and only VLAN resource information in a terraform state file will be deleted.
	// Physical VLAN will be deleted automatically which the VLAN doesn't have any child resources.
	if billingItem.Id == nil {
		return nil
	}

	// If the VLAN has a billing item, the function deletes the billing item and returns so that
	// the VLAN resource in a terraform state file can be deleted. Physical VLAN will be deleted
	// automatically which the VLAN doesn't have any child resources.
	_, err = services.GetBillingItemService(sess).Id(*billingItem.Id).CancelService()

	return err
}

func resourceIBMCloudInfraVlanExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	sess := meta.(ClientSession).SoftLayerSession()
	service := services.GetNetworkVlanService(sess)

	vlanID, err := strconv.Atoi(d.Id())
	if err != nil {
		return false, fmt.Errorf("Not a valid vlan ID, must be an integer: %s", err)
	}

	_, err = service.Id(vlanID).Mask("id").GetObject()

	return err == nil, err
}

func findVlanByOrderID(sess *session.Session, orderID int) (datatypes.Network_Vlan, error) {
	const pendingState = "pending"
	const completeState = "complete"

	stateConf := &resource.StateChangeConf{
		Pending: []string{pendingState},
		Target:  []string{completeState},
		Refresh: func() (interface{}, string, error) {
			vlans, err := services.GetAccountService(sess).
				Filter(filter.Path("networkVlans.billingItem.orderItem.order.id").
					Eq(strconv.Itoa(orderID)).Build()).
				Mask("id").
				GetNetworkVlans()
			if err != nil {
				return datatypes.Network_Vlan{}, "", err
			}

			if len(vlans) == 1 {
				return vlans[0], completeState, nil
			} else if len(vlans) == 0 {
				return nil, pendingState, nil
			} else {
				return nil, "", fmt.Errorf("Expected one vlan: %s", err)
			}
		},
		Timeout:    10 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	pendingResult, err := stateConf.WaitForState()

	if err != nil {
		return datatypes.Network_Vlan{}, err
	}

	var result, ok = pendingResult.(datatypes.Network_Vlan)

	if ok {
		return result, nil
	}

	return datatypes.Network_Vlan{},
		fmt.Errorf("Cannot find vlan with order id '%d'", orderID)
}

func buildVlanProductOrderContainer(d *schema.ResourceData, sess *session.Session, packageType string) (
	*datatypes.Container_Product_Order_Network_Vlan, error) {
	var rt datatypes.Hardware
	router := d.Get("router_hostname").(string)

	vlanType := d.Get("type").(string)
	datacenter := d.Get("datacenter").(string)

	if datacenter == "" {
		return &datatypes.Container_Product_Order_Network_Vlan{},
			errors.New("datacenter name is empty")
	}

	dc, err := location.GetDatacenterByName(sess, datacenter, "id")
	if err != nil {
		return &datatypes.Container_Product_Order_Network_Vlan{}, err
	}

	// 1. Get a package
	pkg, err := product.GetPackageByType(sess, packageType)
	if err != nil {
		return &datatypes.Container_Product_Order_Network_Vlan{}, err
	}

	// 2. Get all prices for the package
	productItems, err := product.GetPackageProducts(sess, *pkg.Id)
	if err != nil {
		return &datatypes.Container_Product_Order_Network_Vlan{}, err
	}

	// 3. Find vlan and subnet prices
	vlanKeyname := vlanType + "_NETWORK_VLAN"
	subnetKeyname := strconv.Itoa(d.Get("subnet_size").(int)) + "_STATIC_PUBLIC_IP_ADDRESSES"

	// 4. Select items with a matching keyname
	vlanItems := []datatypes.Product_Item{}
	subnetItems := []datatypes.Product_Item{}
	for _, item := range productItems {
		if *item.KeyName == vlanKeyname {
			vlanItems = append(vlanItems, item)
		}
		if strings.Contains(*item.KeyName, subnetKeyname) {
			subnetItems = append(subnetItems, item)
		}
	}

	if len(vlanItems) == 0 {
		return &datatypes.Container_Product_Order_Network_Vlan{},
			fmt.Errorf("No product items matching %s could be found", vlanKeyname)
	}

	if len(subnetItems) == 0 {
		return &datatypes.Container_Product_Order_Network_Vlan{},
			fmt.Errorf("No product items matching %s could be found", subnetKeyname)
	}

	productOrderContainer := datatypes.Container_Product_Order_Network_Vlan{
		Container_Product_Order: datatypes.Container_Product_Order{
			PackageId: pkg.Id,
			Location:  sl.String(strconv.Itoa(*dc.Id)),
			Prices: []datatypes.Product_Item_Price{
				{
					Id: vlanItems[0].Prices[0].Id,
				},
				{
					Id: subnetItems[0].Prices[0].Id,
				},
			},
			Quantity: sl.Int(1),
		},
	}

	if len(router) > 0 {
		rt, err = hardware.GetRouterByName(sess, router, "id")
		productOrderContainer.RouterId = rt.Id
		if err != nil {
			return &datatypes.Container_Product_Order_Network_Vlan{},
				fmt.Errorf("Error creating vlan: %s", err)
		}
	}

	return &productOrderContainer, nil
}
