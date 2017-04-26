package alicloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/rds"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"strconv"
	"strings"
	"time"
)

func resourceAlicloudDBInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceAlicloudDBInstanceCreate,
		Read:   resourceAlicloudDBInstanceRead,
		Update: resourceAlicloudDBInstanceUpdate,
		Delete: resourceAlicloudDBInstanceDelete,

		Schema: map[string]*schema.Schema{
			"engine": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validateAllowedStringValue([]string{"MySQL", "SQLServer", "PostgreSQL", "PPAS"}),
				ForceNew:     true,
				Required:     true,
			},
			"engine_version": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validateAllowedStringValue([]string{"5.5", "5.6", "5.7", "2008r2", "2012", "9.4", "9.3"}),
				ForceNew:     true,
				Required:     true,
			},
			"db_instance_class": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"db_instance_storage": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},

			"instance_charge_type": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validateAllowedStringValue([]string{string(rds.Postpaid), string(rds.Prepaid)}),
				Optional:     true,
				ForceNew:     true,
				Default:      rds.Postpaid,
			},
			"period": &schema.Schema{
				Type:         schema.TypeInt,
				ValidateFunc: validateAllowedIntValue([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 12, 24, 36}),
				Optional:     true,
				ForceNew:     true,
				Default:      1,
			},

			"zone_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"multi_az": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"db_instance_net_type": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validateAllowedStringValue([]string{string(common.Internet), string(common.Intranet)}),
				Optional:     true,
			},
			"allocate_public_connection": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"instance_network_type": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validateAllowedStringValue([]string{string(common.VPC), string(common.Classic)}),
				Optional:     true,
				Computed:     true,
			},
			"vswitch_id": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},

			"master_user_name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
			"master_user_password": &schema.Schema{
				Type:      schema.TypeString,
				ForceNew:  true,
				Optional:  true,
				Sensitive: true,
			},

			"preferred_backup_period": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Schema{Type: schema.TypeString},
				// terraform does not support ValidateFunc of TypeList attr
				// ValidateFunc: validateAllowedStringValue([]string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}),
				Optional: true,
			},
			"preferred_backup_time": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validateAllowedStringValue(rds.BACKUP_TIME),
				Optional:     true,
			},
			"backup_retention_period": &schema.Schema{
				Type:         schema.TypeInt,
				ValidateFunc: validateIntegerInRange(7, 730),
				Optional:     true,
			},

			"security_ips": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
				Optional: true,
			},

			"port": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"connections": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"connection_string": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"ip_type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"ip_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Computed: true,
			},

			"db_mappings": &schema.Schema{
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"db_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"character_set_name": &schema.Schema{
							Type:         schema.TypeString,
							ValidateFunc: validateAllowedStringValue(rds.CHARACTER_SET_NAME),
							Required:     true,
						},
						"db_description": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Optional: true,
				Set:      resourceAlicloudDatabaseHash,
			},
		},
	}
}

func resourceAlicloudDatabaseHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["db_name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["character_set_name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["db_description"].(string)))

	return hashcode.String(buf.String())
}

func resourceAlicloudDBInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)
	conn := client.rdsconn

	args, err := buildDBCreateOrderArgs(d, meta)
	if err != nil {
		return err
	}

	resp, err := conn.CreateOrder(args)

	if err != nil {
		return fmt.Errorf("Error creating Alicloud db instance: %#v", err)
	}

	instanceId := resp.DBInstanceId
	if instanceId == "" {
		return fmt.Errorf("Error get Alicloud db instance id")
	}

	d.SetId(instanceId)
	d.Set("instance_charge_type", d.Get("instance_charge_type"))
	d.Set("period", d.Get("period"))
	d.Set("period_type", d.Get("period_type"))

	// wait instance status change from Creating to running
	if err := conn.WaitForInstance(d.Id(), rds.Running, defaultLongTimeout); err != nil {
		return fmt.Errorf("WaitForInstance %s got error: %#v", rds.Running, err)
	}

	if err := modifySecurityIps(d.Id(), d.Get("security_ips"), meta); err != nil {
		return err
	}

	masterUserName := d.Get("master_user_name").(string)
	masterUserPwd := d.Get("master_user_password").(string)
	if masterUserName != "" && masterUserPwd != "" {
		if err := client.CreateAccountByInfo(d.Id(), masterUserName, masterUserPwd); err != nil {
			return fmt.Errorf("Create db account %s error: %v", masterUserName, err)
		}
	}

	if d.Get("allocate_public_connection").(bool) {
		if err := client.AllocateDBPublicConnection(d.Id(), DB_DEFAULT_CONNECT_PORT); err != nil {
			return fmt.Errorf("Allocate public connection error: %v", err)
		}
	}

	return resourceAlicloudDBInstanceUpdate(d, meta)
}

func modifySecurityIps(id string, ips interface{}, meta interface{}) error {
	client := meta.(*AliyunClient)
	ipList := expandStringList(ips.([]interface{}))

	ipstr := strings.Join(ipList[:], COMMA_SEPARATED)
	// default disable connect from outside
	if ipstr == "" {
		ipstr = LOCAL_HOST_IP
	}

	if err := client.ModifyDBSecurityIps(id, ipstr); err != nil {
		return fmt.Errorf("Error modify security ips %s: %#v", ipstr, err)
	}
	return nil
}

func resourceAlicloudDBInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)
	conn := client.rdsconn
	d.Partial(true)

	if d.HasChange("db_mappings") {
		o, n := d.GetChange("db_mappings")
		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		var allDbs []string
		remove := os.Difference(ns).List()
		add := ns.Difference(os).List()

		if len(remove) > 0 && len(add) > 0 {
			return fmt.Errorf("Failure modify database, we neither support create and delete database simultaneous nor modify database attributes.")
		}

		if len(remove) > 0 {
			for _, db := range remove {
				dbm, _ := db.(map[string]interface{})
				if err := conn.DeleteDatabase(d.Id(), dbm["db_name"].(string)); err != nil {
					return fmt.Errorf("Failure delete database %s: %#v", dbm["db_name"].(string), err)
				}
			}
		}

		if len(add) > 0 {
			for _, db := range add {
				dbm, _ := db.(map[string]interface{})
				dbName := dbm["db_name"].(string)
				allDbs = append(allDbs, dbName)

				if err := client.CreateDatabaseByInfo(d.Id(), dbName, dbm["character_set_name"].(string), dbm["db_description"].(string)); err != nil {
					return fmt.Errorf("Failure create database %s: %#v", dbName, err)
				}

			}
		}

		if err := conn.WaitForAllDatabase(d.Id(), allDbs, rds.Running, 600); err != nil {
			return fmt.Errorf("Failure create database %#v", err)
		}

		if user := d.Get("master_user_name").(string); user != "" {
			for _, dbName := range allDbs {
				if err := client.GrantDBPrivilege2Account(d.Id(), user, dbName); err != nil {
					return fmt.Errorf("Failed to grant database %s readwrite privilege to account %s: %#v", dbName, user, err)
				}
			}
		}

		d.SetPartial("db_mappings")
	}

	if d.HasChange("preferred_backup_period") || d.HasChange("preferred_backup_time") || d.HasChange("backup_retention_period") {
		period := d.Get("preferred_backup_period").([]interface{})
		periodList := expandStringList(period)
		time := d.Get("preferred_backup_time").(string)
		retention := d.Get("backup_retention_period").(int)

		if time == "" || retention == 0 || len(periodList) < 1 {
			return fmt.Errorf("Both backup_time, backup_period and retention_period are required to set backup policy.")
		}

		ps := strings.Join(periodList[:], COMMA_SEPARATED)

		if err := client.ConfigDBBackup(d.Id(), time, ps, retention); err != nil {
			return fmt.Errorf("Error set backup policy: %#v", err)
		}
		d.SetPartial("preferred_backup_period")
		d.SetPartial("preferred_backup_time")
		d.SetPartial("backup_retention_period")
	}

	if d.HasChange("security_ips") {
		if err := modifySecurityIps(d.Id(), d.Get("security_ips"), meta); err != nil {
			return err
		}
		d.SetPartial("security_ips")
	}

	if d.HasChange("db_instance_class") || d.HasChange("db_instance_storage") {
		co, cn := d.GetChange("db_instance_class")
		so, sn := d.GetChange("db_instance_storage")
		classOld := co.(string)
		classNew := cn.(string)
		storageOld := so.(int)
		storageNew := sn.(int)

		// update except the first time, because we will do it in create function
		if classOld != "" && storageOld != 0 {
			chargeType := d.Get("instance_charge_type").(string)
			if chargeType == string(rds.Prepaid) {
				return fmt.Errorf("Prepaid db instance does not support modify db_instance_class or db_instance_storage")
			}

			if err := client.ModifyDBClassStorage(d.Id(), classNew, strconv.Itoa(storageNew)); err != nil {
				return fmt.Errorf("Error modify db instance class or storage error: %#v", err)
			}
		}
	}

	d.Partial(false)
	return resourceAlicloudDBInstanceRead(d, meta)
}

func resourceAlicloudDBInstanceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)
	conn := client.rdsconn

	instance, err := client.DescribeDBInstanceById(d.Id())
	if err != nil {
		if notFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error Describe DB InstanceAttribute: %#v", err)
	}

	args := rds.DescribeDatabasesArgs{
		DBInstanceId: d.Id(),
	}

	resp, err := conn.DescribeDatabases(&args)
	if err != nil {
		return err
	}
	if resp.Databases.Database == nil {
		d.SetId("")
		return nil
	}

	d.Set("db_mappings", flattenDatabaseMappings(resp.Databases.Database))

	argn := rds.DescribeDBInstanceNetInfoArgs{
		DBInstanceId: d.Id(),
	}

	resn, err := conn.DescribeDBInstanceNetInfo(&argn)
	if err != nil {
		return err
	}
	d.Set("connections", flattenDBConnections(resn.DBInstanceNetInfos.DBInstanceNetInfo))

	ips, err := client.GetSecurityIps(d.Id())
	if err != nil {
		log.Printf("Describe DB security ips error: %#v", err)
	}
	d.Set("security_ips", ips)

	d.Set("engine", instance.Engine)
	d.Set("engine_version", instance.EngineVersion)
	d.Set("db_instance_class", instance.DBInstanceClass)
	d.Set("port", instance.Port)
	d.Set("db_instance_storage", instance.DBInstanceStorage)
	d.Set("zone_id", instance.ZoneId)
	d.Set("db_instance_net_type", instance.DBInstanceNetType)
	d.Set("instance_network_type", instance.InstanceNetworkType)

	return nil
}

func resourceAlicloudDBInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).rdsconn

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		err := conn.DeleteInstance(d.Id())

		if err != nil {
			return resource.RetryableError(fmt.Errorf("DB Instance in use - trying again while it is deleted."))
		}

		args := &rds.DescribeDBInstancesArgs{
			DBInstanceId: d.Id(),
		}
		resp, err := conn.DescribeDBInstanceAttribute(args)
		if err != nil {
			return resource.NonRetryableError(err)
		} else if len(resp.Items.DBInstanceAttribute) < 1 {
			return nil
		}

		return resource.RetryableError(fmt.Errorf("DB in use - trying again while it is deleted."))
	})
}

func buildDBCreateOrderArgs(d *schema.ResourceData, meta interface{}) (*rds.CreateOrderArgs, error) {
	client := meta.(*AliyunClient)
	args := &rds.CreateOrderArgs{
		RegionId: getRegion(d, meta),
		// we does not expose this param to user,
		// because create prepaid instance progress will be stopped when set auto_pay to false,
		// then could not get instance info, cause timeout error
		AutoPay:           "true",
		EngineVersion:     d.Get("engine_version").(string),
		Engine:            rds.Engine(d.Get("engine").(string)),
		DBInstanceStorage: d.Get("db_instance_storage").(int),
		DBInstanceClass:   d.Get("db_instance_class").(string),
		Quantity:          DEFAULT_INSTANCE_COUNT,
		Resource:          rds.DefaultResource,
	}

	bussStr, err := json.Marshal(DefaultBusinessInfo)
	if err != nil {
		return nil, fmt.Errorf("Failed to translate bussiness info %#v from json to string", DefaultBusinessInfo)
	}

	args.BusinessInfo = string(bussStr)

	zoneId := d.Get("zone_id").(string)
	args.ZoneId = zoneId

	multiAZ := d.Get("multi_az").(bool)
	if multiAZ {
		if zoneId != "" {
			return nil, fmt.Errorf("You cannot set the ZoneId parameter when the MultiAZ parameter is set to true")
		}
		izs, err := client.DescribeMultiIZByRegion()
		if err != nil {
			return nil, fmt.Errorf("Get multiAZ id error")
		}

		if len(izs) < 1 {
			return nil, fmt.Errorf("Current region does not support MultiAZ.")
		}

		args.ZoneId = izs[0]
	}

	vswitchId := d.Get("vswitch_id").(string)

	networkType := d.Get("instance_network_type").(string)
	args.InstanceNetworkType = common.NetworkType(networkType)

	if vswitchId != "" {
		args.VSwitchId = vswitchId

		// check InstanceNetworkType with vswitchId
		if networkType == string(common.Classic) {
			return nil, fmt.Errorf("When fill vswitchId, you shold set instance_network_type to VPC")
		} else if networkType == "" {
			args.InstanceNetworkType = common.VPC
		}

		// get vpcId
		vpcId, err := client.GetVpcIdByVSwitchId(vswitchId)

		if err != nil {
			return nil, fmt.Errorf("VswitchId %s is not valid of current region", vswitchId)
		}
		// fill vpcId by vswitchId
		args.VPCId = vpcId

		// check vswitchId in zone
		vsw, err := client.QueryVswitchById(vpcId, vswitchId)
		if err != nil {
			return nil, fmt.Errorf("VswitchId %s is not valid of current region", vswitchId)
		}

		if zoneId == "" {
			args.ZoneId = vsw.ZoneId
		} else if vsw.ZoneId != zoneId {
			return nil, fmt.Errorf("VswitchId %s is not belong to the zone %s", vswitchId, zoneId)
		}
	}

	if v := d.Get("db_instance_net_type").(string); v != "" {
		args.DBInstanceNetType = common.NetType(v)
	}

	chargeType := d.Get("instance_charge_type").(string)
	if chargeType != "" {
		args.PayType = rds.DBPayType(chargeType)
	} else {
		args.PayType = rds.Postpaid
	}

	// if charge type is postpaid, the commodity code must set to bards
	if chargeType == string(rds.Postpaid) {
		args.CommodityCode = rds.Bards
	} else {
		args.CommodityCode = rds.Rds
	}

	period := d.Get("period").(int)
	args.UsedTime, args.TimeType = TransformPeriod2Time(period, chargeType)

	return args, nil
}
