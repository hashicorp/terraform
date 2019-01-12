package aws

import (
	"net/url"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/datasync"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSyncParseLocationURI(uri string) (string, error) {
	parsedURL, err := url.ParseRequestURI(uri)

	if err != nil {
		return "", err
	}

	return parsedURL.Path, nil
}

func expandDataSyncEc2Config(l []interface{}) *datasync.Ec2Config {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	ec2Config := &datasync.Ec2Config{
		SecurityGroupArns: expandStringSet(m["security_group_arns"].(*schema.Set)),
		SubnetArn:         aws.String(m["subnet_arn"].(string)),
	}

	return ec2Config
}

func expandDataSyncOnPremConfig(l []interface{}) *datasync.OnPremConfig {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	onPremConfig := &datasync.OnPremConfig{
		AgentArns: expandStringSet(m["agent_arns"].(*schema.Set)),
	}

	return onPremConfig
}

func expandDataSyncOptions(l []interface{}) *datasync.Options {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	options := &datasync.Options{
		Atime:                aws.String(m["atime"].(string)),
		Gid:                  aws.String(m["gid"].(string)),
		Mtime:                aws.String(m["mtime"].(string)),
		PreserveDeletedFiles: aws.String(m["preserve_deleted_files"].(string)),
		PreserveDevices:      aws.String(m["preserve_devices"].(string)),
		PosixPermissions:     aws.String(m["posix_permissions"].(string)),
		Uid:                  aws.String(m["uid"].(string)),
		VerifyMode:           aws.String(m["verify_mode"].(string)),
	}

	if v, ok := m["bytes_per_second"]; ok && v.(int) > 0 {
		options.BytesPerSecond = aws.Int64(int64(v.(int)))
	}

	return options
}

func expandDataSyncS3Config(l []interface{}) *datasync.S3Config {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	s3Config := &datasync.S3Config{
		BucketAccessRoleArn: aws.String(m["bucket_access_role_arn"].(string)),
	}

	return s3Config
}

func flattenDataSyncEc2Config(ec2Config *datasync.Ec2Config) []interface{} {
	if ec2Config == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"security_group_arns": schema.NewSet(schema.HashString, flattenStringList(ec2Config.SecurityGroupArns)),
		"subnet_arn":          aws.StringValue(ec2Config.SubnetArn),
	}

	return []interface{}{m}
}

func flattenDataSyncOnPremConfig(onPremConfig *datasync.OnPremConfig) []interface{} {
	if onPremConfig == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"agent_arns": schema.NewSet(schema.HashString, flattenStringList(onPremConfig.AgentArns)),
	}

	return []interface{}{m}
}

func flattenDataSyncOptions(options *datasync.Options) []interface{} {
	if options == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"atime":                  aws.StringValue(options.Atime),
		"bytes_per_second":       aws.Int64Value(options.BytesPerSecond),
		"gid":                    aws.StringValue(options.Gid),
		"mtime":                  aws.StringValue(options.Mtime),
		"posix_permissions":      aws.StringValue(options.PosixPermissions),
		"preserve_deleted_files": aws.StringValue(options.PreserveDeletedFiles),
		"preserve_devices":       aws.StringValue(options.PreserveDevices),
		"uid":                    aws.StringValue(options.Uid),
		"verify_mode":            aws.StringValue(options.VerifyMode),
	}

	return []interface{}{m}
}

func flattenDataSyncS3Config(s3Config *datasync.S3Config) []interface{} {
	if s3Config == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"bucket_access_role_arn": aws.StringValue(s3Config.BucketAccessRoleArn),
	}

	return []interface{}{m}
}
