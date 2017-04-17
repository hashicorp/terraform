package alicloud

import (
	"github.com/denverdino/aliyungo/slb"
)

func (client *AliyunClient) DescribeLoadBalancerAttribute(slbId string) (*slb.LoadBalancerType, error) {
	loadBalancer, err := client.slbconn.DescribeLoadBalancerAttribute(slbId)
	if err != nil {
		if notFoundError(err) {
			return nil, nil
		}
		return nil, err
	}

	if loadBalancer != nil {
		return loadBalancer, nil
	}

	return nil, nil
}
