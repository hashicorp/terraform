package alicloud

import (
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/rds"
	"strings"
)

//
//       _______________                      _______________                       _______________
//       |              | ______param______\  |              |  _____request_____\  |              |
//       |   Business   |                     |    Service   |                      |    SDK/API   |
//       |              | __________________  |              |  __________________  |              |
//       |______________| \    (obj, err)     |______________|  \ (status, cont)    |______________|
//                           |                                    |
//                           |A. {instance, nil}                  |a. {200, content}
//                           |B. {nil, error}                     |b. {200, nil}
//                      					  |c. {4xx, nil}
//
// The API return 200 for resource not found.
// When getInstance is empty, then throw InstanceNotfound error.
// That the business layer only need to check error.
func (client *AliyunClient) DescribeDBInstanceById(id string) (instance *rds.DBInstanceAttribute, err error) {
	arrtArgs := rds.DescribeDBInstancesArgs{
		DBInstanceId: id,
	}
	resp, err := client.rdsconn.DescribeDBInstanceAttribute(&arrtArgs)
	if err != nil {
		return nil, err
	}

	attr := resp.Items.DBInstanceAttribute

	if len(attr) <= 0 {
		return nil, GetNotFoundErrorFromString("DB instance not found")
	}

	return &attr[0], nil
}

func (client *AliyunClient) CreateAccountByInfo(instanceId, username, pwd string) error {
	conn := client.rdsconn
	args := rds.CreateAccountArgs{
		DBInstanceId:    instanceId,
		AccountName:     username,
		AccountPassword: pwd,
	}

	if _, err := conn.CreateAccount(&args); err != nil {
		return err
	}

	if err := conn.WaitForAccount(instanceId, username, rds.Available, 200); err != nil {
		return err
	}
	return nil
}

func (client *AliyunClient) CreateDatabaseByInfo(instanceId, dbName, charset, desp string) error {
	conn := client.rdsconn
	args := rds.CreateDatabaseArgs{
		DBInstanceId:     instanceId,
		DBName:           dbName,
		CharacterSetName: charset,
		DBDescription:    desp,
	}
	_, err := conn.CreateDatabase(&args)
	return err
}

func (client *AliyunClient) DescribeDatabaseByName(instanceId, dbName string) (ds []rds.Database, err error) {
	conn := client.rdsconn
	args := rds.DescribeDatabasesArgs{
		DBInstanceId: instanceId,
		DBName:       dbName,
	}

	resp, err := conn.DescribeDatabases(&args)
	if err != nil {
		return nil, err
	}

	return resp.Databases.Database, nil
}

func (client *AliyunClient) GrantDBPrivilege2Account(instanceId, username, dbName string) error {
	conn := client.rdsconn
	pargs := rds.GrantAccountPrivilegeArgs{
		DBInstanceId:     instanceId,
		AccountName:      username,
		DBName:           dbName,
		AccountPrivilege: rds.ReadWrite,
	}
	if _, err := conn.GrantAccountPrivilege(&pargs); err != nil {
		return err
	}

	if err := conn.WaitForAccountPrivilege(instanceId, username, dbName, rds.ReadWrite, 200); err != nil {
		return err
	}
	return nil
}

func (client *AliyunClient) AllocateDBPublicConnection(instanceId, port string) error {
	conn := client.rdsconn
	args := rds.AllocateInstancePublicConnectionArgs{
		DBInstanceId:           instanceId,
		ConnectionStringPrefix: instanceId + "o",
		Port: port,
	}

	if _, err := conn.AllocateInstancePublicConnection(&args); err != nil {
		return err
	}

	if err := conn.WaitForPublicConnection(instanceId, 600); err != nil {
		return err
	}
	return nil
}

func (client *AliyunClient) ConfigDBBackup(instanceId, backupTime, backupPeriod string, retentionPeriod int) error {
	bargs := rds.BackupPolicy{
		PreferredBackupTime:   backupTime,
		PreferredBackupPeriod: backupPeriod,
		BackupRetentionPeriod: retentionPeriod,
	}
	args := rds.ModifyBackupPolicyArgs{
		DBInstanceId: instanceId,
		BackupPolicy: bargs,
	}

	if _, err := client.rdsconn.ModifyBackupPolicy(&args); err != nil {
		return err
	}

	if err := client.rdsconn.WaitForInstance(instanceId, rds.Running, 600); err != nil {
		return err
	}
	return nil
}

func (client *AliyunClient) ModifyDBSecurityIps(instanceId, ips string) error {
	sargs := rds.DBInstanceIPArray{
		SecurityIps: ips,
	}

	args := rds.ModifySecurityIpsArgs{
		DBInstanceId:      instanceId,
		DBInstanceIPArray: sargs,
	}

	if _, err := client.rdsconn.ModifySecurityIps(&args); err != nil {
		return err
	}

	if err := client.rdsconn.WaitForInstance(instanceId, rds.Running, 600); err != nil {
		return err
	}
	return nil
}

func (client *AliyunClient) DescribeDBSecurityIps(instanceId string) (ips []rds.DBInstanceIPList, err error) {
	args := rds.DescribeDBInstanceIPsArgs{
		DBInstanceId: instanceId,
	}

	resp, err := client.rdsconn.DescribeDBInstanceIPs(&args)
	if err != nil {
		return nil, err
	}
	return resp.Items.DBInstanceIPArray, nil
}

func (client *AliyunClient) GetSecurityIps(instanceId string) ([]string, error) {
	arr, err := client.DescribeDBSecurityIps(instanceId)
	if err != nil {
		return nil, err
	}
	var ips, separator string
	for _, ip := range arr {
		ips += separator + ip.SecurityIPList
		separator = COMMA_SEPARATED
	}
	return strings.Split(ips, COMMA_SEPARATED), nil
}

func (client *AliyunClient) ModifyDBClassStorage(instanceId, class, storage string) error {
	conn := client.rdsconn
	args := rds.ModifyDBInstanceSpecArgs{
		DBInstanceId:      instanceId,
		PayType:           rds.Postpaid,
		DBInstanceClass:   class,
		DBInstanceStorage: storage,
	}

	if _, err := conn.ModifyDBInstanceSpec(&args); err != nil {
		return err
	}

	if err := conn.WaitForInstance(instanceId, rds.Running, 600); err != nil {
		return err
	}
	return nil
}

// turn period to TimeType
func TransformPeriod2Time(period int, chargeType string) (ut int, tt common.TimeType) {
	if chargeType == string(rds.Postpaid) {
		return 1, common.Day
	}

	if period >= 1 && period <= 9 {
		return period, common.Month
	}

	if period == 12 {
		return 1, common.Year
	}

	if period == 24 {
		return 2, common.Year
	}
	return 0, common.Day

}

// turn TimeType to Period
func TransformTime2Period(ut int, tt common.TimeType) (period int) {
	if tt == common.Year {
		return 12 * ut
	}

	return ut

}

// Flattens an array of databases into a []map[string]interface{}
func flattenDatabaseMappings(list []rds.Database) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))
	for _, i := range list {
		l := map[string]interface{}{
			"db_name":            i.DBName,
			"character_set_name": i.CharacterSetName,
			"db_description":     i.DBDescription,
		}
		result = append(result, l)
	}
	return result
}

func flattenDBBackup(list []rds.BackupPolicy) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))
	for _, i := range list {
		l := map[string]interface{}{
			"preferred_backup_period": i.PreferredBackupPeriod,
			"preferred_backup_time":   i.PreferredBackupTime,
			"backup_retention_period": i.LogBackupRetentionPeriod,
		}
		result = append(result, l)
	}
	return result
}

func flattenDBSecurityIPs(list []rds.DBInstanceIPList) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))
	for _, i := range list {
		l := map[string]interface{}{
			"security_ips": i.SecurityIPList,
		}
		result = append(result, l)
	}
	return result
}

// Flattens an array of databases connection into a []map[string]interface{}
func flattenDBConnections(list []rds.DBInstanceNetInfo) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))
	for _, i := range list {
		l := map[string]interface{}{
			"connection_string": i.ConnectionString,
			"ip_type":           i.IPType,
			"ip_address":        i.IPAddress,
		}
		result = append(result, l)
	}
	return result
}
