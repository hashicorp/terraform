package alicloud

import (
	"testing"
	"github.com/denverdino/aliyungo/ecs"
)

func TestAccVSwitch(t *testing.T) {
	vswitchClient := NewTestClient()
	vswitchArgs := &ecs.CreateVSwitchArgs{
		VpcId:     "vpc-2zenkmf95pz761l67ol7z",
		ZoneId:    "cn-beijing-b",
		CidrBlock: "10.1.1.0/24",
	}
	vswitchId, err := vswitchClient.CreateVSwitch(vswitchArgs)

	if err != nil {
		t.Fatalf("Failed to create vswitch: %s \n", err)
	} else {
		t.Logf("vswitch id is %s \n", vswitchId)
	}

}
