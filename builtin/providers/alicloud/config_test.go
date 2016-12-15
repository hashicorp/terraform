package alicloud

import "github.com/denverdino/aliyungo/ecs"

//Modify with your Access Key Id and Access Key Secret

const (
	TestAccessKeyId = "****"
	TestAccessKeySecret = "****"
	TestInstanceId = "MY_TEST_INSTANCE_ID"
	TestSecurityGroupId = "MY_TEST_SECURITY_GROUP_ID"
	TestImageId = "MY_TEST_IMAGE_ID"
	TestAccountId = "MY_TEST_ACCOUNT_ID" //Get from https://account.console.aliyun.com

	TestIAmRich = true
	TestQuick = false
)

var testClient *ecs.Client

func NewTestClient() *ecs.Client {
	if testClient == nil {
		testClient = ecs.NewClient(TestAccessKeyId, TestAccessKeySecret)
	}
	return testClient
}

var testDebugClient *ecs.Client

func NewTestClientForDebug() *ecs.Client {
	if testDebugClient == nil {
		testDebugClient = ecs.NewClient(TestAccessKeyId, TestAccessKeySecret)
		testDebugClient.SetDebug(true)
	}
	return testDebugClient
}
