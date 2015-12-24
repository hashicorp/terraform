package aws

import (
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
)

func makeAwsStringList(in []interface{}) []*string {
	ret := make([]*string, len(in), len(in))
	for i := 0; i < len(in); i++ {
		ret[i] = aws.String(in[i].(string))
	}
	return ret
}

func makeAwsStringSet(in *schema.Set) []*string {
	inList := in.List()
	ret := make([]*string, len(inList), len(inList))
	for i := 0; i < len(ret); i++ {
		ret[i] = aws.String(inList[i].(string))
	}
	return ret
}

func unwrapAwsStringList(in []*string) []string {
	ret := make([]string, len(in), len(in))
	for i := 0; i < len(in); i++ {
		if in[i] != nil {
			ret[i] = *in[i]
		}
	}
	return ret
}
