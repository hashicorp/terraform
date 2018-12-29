package aws

import (
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/iam"
)

func arnString(partition, region, service, accountId, resource string) string {
	return arn.ARN{
		Partition: partition,
		Region:    region,
		Service:   service,
		AccountID: accountId,
		Resource:  resource,
	}.String()
}

// See http://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html#arn-syntax-iam
func iamArnString(partition, accountId, resource string) string {
	return arnString(
		partition,
		"",
		iam.ServiceName,
		accountId,
		resource)
}
