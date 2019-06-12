
package endpoints

import (
	"encoding/json"
	"fmt"
	"sync"
)

const endpointsJson =`{
	"products": [
		{
			"code": "ecs",
			"document_id": "25484",
			"location_service_code": "ecs",
			"regional_endpoints": [
				{
					"region": "cn-shanghai",
					"endpoint": "ecs-cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "eu-west-1",
					"endpoint": "ecs.eu-west-1.aliyuncs.com"
				},
				{
					"region": "cn-huhehaote",
					"endpoint": "ecs.cn-huhehaote.aliyuncs.com"
				},
				{
					"region": "me-east-1",
					"endpoint": "ecs.me-east-1.aliyuncs.com"
				},
				{
					"region": "ap-southeast-3",
					"endpoint": "ecs.ap-southeast-3.aliyuncs.com"
				},
				{
					"region": "ap-southeast-2",
					"endpoint": "ecs.ap-southeast-2.aliyuncs.com"
				},
				{
					"region": "ap-south-1",
					"endpoint": "ecs.ap-south-1.aliyuncs.com"
				},
				{
					"region": "cn-beijing",
					"endpoint": "ecs-cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "ecs-cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "cn-shenzhen",
					"endpoint": "ecs-cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "ap-northeast-1",
					"endpoint": "ecs.ap-northeast-1.aliyuncs.com"
				},
				{
					"region": "ap-southeast-5",
					"endpoint": "ecs.ap-southeast-5.aliyuncs.com"
				},
				{
					"region": "eu-central-1",
					"endpoint": "ecs.eu-central-1.aliyuncs.com"
				},
				{
					"region": "cn-zhangjiakou",
					"endpoint": "ecs.cn-zhangjiakou.aliyuncs.com"
				},
				{
					"region": "cn-qingdao",
					"endpoint": "ecs-cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "cn-hongkong",
					"endpoint": "ecs-cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "ap-southeast-1",
					"endpoint": "ecs-cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "us-west-1",
					"endpoint": "ecs-cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "us-east-1",
					"endpoint": "ecs-cn-hangzhou.aliyuncs.com"
				}
			],
			"global_endpoint": "ecs-cn-hangzhou.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "chatbot",
			"document_id": "60760",
			"location_service_code": "beebot",
			"regional_endpoints": [
				{
					"region": "cn-shanghai",
					"endpoint": "chatbot.cn-shanghai.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "chatbot.cn-hangzhou.aliyuncs.com"
				}
			],
			"global_endpoint": "",
			"regional_endpoint_pattern": "chatbot.[RegionId].aliyuncs.com"
		},
		{
			"code": "alidns",
			"document_id": "29739",
			"location_service_code": "alidns",
			"regional_endpoints": [
				{
					"region": "cn-hangzhou",
					"endpoint": "alidns.aliyuncs.com"
				}
			],
			"global_endpoint": "alidns.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "itaas",
			"document_id": "55759",
			"location_service_code": "itaas",
			"regional_endpoints": null,
			"global_endpoint": "itaas.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "csb",
			"document_id": "64837",
			"location_service_code": "csb",
			"regional_endpoints": [
				{
					"region": "cn-hangzhou",
					"endpoint": "csb.cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "cn-beijing",
					"endpoint": "csb.cn-beijing.aliyuncs.com"
				}
			],
			"global_endpoint": "",
			"regional_endpoint_pattern": "csb.[RegionId].aliyuncs.com"
		},
		{
			"code": "slb",
			"document_id": "27565",
			"location_service_code": "slb",
			"regional_endpoints": [
				{
					"region": "cn-hongkong",
					"endpoint": "slb.aliyuncs.com"
				},
				{
					"region": "me-east-1",
					"endpoint": "slb.me-east-1.aliyuncs.com"
				},
				{
					"region": "ap-southeast-5",
					"endpoint": "slb.ap-southeast-5.aliyuncs.com"
				},
				{
					"region": "ap-southeast-2",
					"endpoint": "slb.ap-southeast-2.aliyuncs.com"
				},
				{
					"region": "ap-south-1",
					"endpoint": "slb.ap-south-1.aliyuncs.com"
				},
				{
					"region": "eu-central-1",
					"endpoint": "slb.eu-central-1.aliyuncs.com"
				},
				{
					"region": "cn-shanghai",
					"endpoint": "slb.aliyuncs.com"
				},
				{
					"region": "eu-west-1",
					"endpoint": "slb.eu-west-1.aliyuncs.com"
				},
				{
					"region": "cn-huhehaote",
					"endpoint": "slb.cn-huhehaote.aliyuncs.com"
				},
				{
					"region": "us-west-1",
					"endpoint": "slb.aliyuncs.com"
				},
				{
					"region": "cn-zhangjiakou",
					"endpoint": "slb.cn-zhangjiakou.aliyuncs.com"
				},
				{
					"region": "cn-qingdao",
					"endpoint": "slb.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "slb.aliyuncs.com"
				},
				{
					"region": "cn-shenzhen",
					"endpoint": "slb.aliyuncs.com"
				},
				{
					"region": "us-east-1",
					"endpoint": "slb.aliyuncs.com"
				},
				{
					"region": "ap-southeast-3",
					"endpoint": "slb.ap-southeast-3.aliyuncs.com"
				},
				{
					"region": "cn-beijing",
					"endpoint": "slb.aliyuncs.com"
				},
				{
					"region": "ap-southeast-1",
					"endpoint": "slb.aliyuncs.com"
				},
				{
					"region": "ap-northeast-1",
					"endpoint": "slb.ap-northeast-1.aliyuncs.com"
				}
			],
			"global_endpoint": "slb.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "cloudwf",
			"document_id": "58111",
			"location_service_code": "cloudwf",
			"regional_endpoints": null,
			"global_endpoint": "cloudwf.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "cloudphoto",
			"document_id": "59902",
			"location_service_code": "cloudphoto",
			"regional_endpoints": [
				{
					"region": "cn-shanghai",
					"endpoint": "cloudphoto.cn-shanghai.aliyuncs.com"
				}
			],
			"global_endpoint": "",
			"regional_endpoint_pattern": "cloudphoto.[RegionId].aliyuncs.com"
		},
		{
			"code": "dds",
			"document_id": "61715",
			"location_service_code": "dds",
			"regional_endpoints": [
				{
					"region": "ap-southeast-5",
					"endpoint": "mongodb.ap-southeast-5.aliyuncs.com"
				},
				{
					"region": "cn-qingdao",
					"endpoint": "mongodb.aliyuncs.com"
				},
				{
					"region": "cn-hongkong",
					"endpoint": "mongodb.aliyuncs.com"
				},
				{
					"region": "eu-west-1",
					"endpoint": "mongodb.eu-west-1.aliyuncs.com"
				},
				{
					"region": "us-west-1",
					"endpoint": "mongodb.aliyuncs.com"
				},
				{
					"region": "us-east-1",
					"endpoint": "mongodb.aliyuncs.com"
				},
				{
					"region": "me-east-1",
					"endpoint": "mongodb.me-east-1.aliyuncs.com"
				},
				{
					"region": "cn-zhangjiakou",
					"endpoint": "mongodb.cn-zhangjiakou.aliyuncs.com"
				},
				{
					"region": "cn-shanghai",
					"endpoint": "mongodb.aliyuncs.com"
				},
				{
					"region": "cn-shenzhen",
					"endpoint": "mongodb.aliyuncs.com"
				},
				{
					"region": "ap-northeast-1",
					"endpoint": "mongodb.ap-northeast-1.aliyuncs.com"
				},
				{
					"region": "ap-southeast-1",
					"endpoint": "mongodb.aliyuncs.com"
				},
				{
					"region": "ap-southeast-2",
					"endpoint": "mongodb.ap-southeast-2.aliyuncs.com"
				},
				{
					"region": "ap-southeast-3",
					"endpoint": "mongodb.ap-southeast-3.aliyuncs.com"
				},
				{
					"region": "ap-south-1",
					"endpoint": "mongodb.ap-south-1.aliyuncs.com"
				},
				{
					"region": "eu-central-1",
					"endpoint": "mongodb.eu-central-1.aliyuncs.com"
				},
				{
					"region": "cn-beijing",
					"endpoint": "mongodb.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "mongodb.aliyuncs.com"
				},
				{
					"region": "cn-huhehaote",
					"endpoint": "mongodb.cn-huhehaote.aliyuncs.com"
				}
			],
			"global_endpoint": "mongodb.aliyuncs.com",
			"regional_endpoint_pattern": "mongodb.[RegionId].aliyuncs.com"
		},
		{
			"code": "dm",
			"document_id": "29434",
			"location_service_code": "dm",
			"regional_endpoints": [
				{
					"region": "ap-southeast-2",
					"endpoint": "dm.ap-southeast-2.aliyuncs.com"
				}
			],
			"global_endpoint": "dm.aliyuncs.com",
			"regional_endpoint_pattern": "dm.[RegionId].aliyuncs.com"
		},
		{
			"code": "ons",
			"document_id": "44416",
			"location_service_code": "ons",
			"regional_endpoints": [
				{
					"region": "cn-zhangjiakou",
					"endpoint": "ons.cn-zhangjiakou.aliyuncs.com"
				},
				{
					"region": "us-west-1",
					"endpoint": "ons.us-west-1.aliyuncs.com"
				},
				{
					"region": "me-east-1",
					"endpoint": "ons.me-east-1.aliyuncs.com"
				},
				{
					"region": "us-east-1",
					"endpoint": "ons.us-east-1.aliyuncs.com"
				},
				{
					"region": "ap-northeast-1",
					"endpoint": "ons.ap-northeast-1.aliyuncs.com"
				},
				{
					"region": "ap-southeast-2",
					"endpoint": "ons.ap-southeast-2.aliyuncs.com"
				},
				{
					"region": "ap-southeast-1",
					"endpoint": "ons.ap-southeast-1.aliyuncs.com"
				},
				{
					"region": "cn-shanghai",
					"endpoint": "ons.cn-shanghai.aliyuncs.com"
				},
				{
					"region": "cn-shenzhen",
					"endpoint": "ons.cn-shenzhen.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "ons.cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "ap-south-1",
					"endpoint": "ons.cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "eu-central-1",
					"endpoint": "ons.eu-central-1.aliyuncs.com"
				},
				{
					"region": "eu-west-1",
					"endpoint": "ons.eu-west-1.aliyuncs.com"
				},
				{
					"region": "cn-beijing",
					"endpoint": "ons.cn-beijing.aliyuncs.com"
				},
				{
					"region": "ap-southeast-3",
					"endpoint": "ons.ap-southeast-3.aliyuncs.com"
				},
				{
					"region": "cn-huhehaote",
					"endpoint": "ons.cn-huhehaote.aliyuncs.com"
				},
				{
					"region": "cn-hongkong",
					"endpoint": "ons.cn-hongkong.aliyuncs.com"
				},
				{
					"region": "cn-qingdao",
					"endpoint": "ons.cn-qingdao.aliyuncs.com"
				}
			],
			"global_endpoint": "",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "polardb",
			"document_id": "58764",
			"location_service_code": "polardb",
			"regional_endpoints": [
				{
					"region": "cn-qingdao",
					"endpoint": "polardb.aliyuncs.com"
				},
				{
					"region": "cn-beijing",
					"endpoint": "polardb.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "polardb.aliyuncs.com"
				},
				{
					"region": "cn-shanghai",
					"endpoint": "polardb.aliyuncs.com"
				},
				{
					"region": "cn-shenzhen",
					"endpoint": "polardb.aliyuncs.com"
				},
				{
					"region": "cn-huhehaote",
					"endpoint": "polardb.cn-huhehaote.aliyuncs.com"
				},
				{
					"region": "ap-southeast-5",
					"endpoint": "polardb.ap-southeast-5.aliyuncs.com"
				},
				{
					"region": "ap-south-1",
					"endpoint": "polardb.ap-south-1.aliyuncs.com"
				},
				{
					"region": "cn-hongkong",
					"endpoint": "polardb.aliyuncs.com"
				}
			],
			"global_endpoint": "",
			"regional_endpoint_pattern": "polardb.aliyuncs.com"
		},
		{
			"code": "batchcompute",
			"document_id": "44717",
			"location_service_code": "batchcompute",
			"regional_endpoints": [
				{
					"region": "us-west-1",
					"endpoint": "batchcompute.us-west-1.aliyuncs.com"
				},
				{
					"region": "cn-beijing",
					"endpoint": "batchcompute.cn-beijing.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "batchcompute.cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "cn-shanghai",
					"endpoint": "batchcompute.cn-shanghai.aliyuncs.com"
				},
				{
					"region": "ap-southeast-1",
					"endpoint": "batchcompute.ap-southeast-1.aliyuncs.com"
				},
				{
					"region": "cn-huhehaote",
					"endpoint": "batchcompute.cn-huhehaote.aliyuncs.com"
				},
				{
					"region": "cn-qingdao",
					"endpoint": "batchcompute.cn-qingdao.aliyuncs.com"
				},
				{
					"region": "cn-zhangjiakou",
					"endpoint": "batchcompute.cn-zhangjiakou.aliyuncs.com"
				},
				{
					"region": "cn-shenzhen",
					"endpoint": "batchcompute.cn-shenzhen.aliyuncs.com"
				}
			],
			"global_endpoint": "",
			"regional_endpoint_pattern": "batchcompute.[RegionId].aliyuncs.com"
		},
		{
			"code": "cloudauth",
			"document_id": "60687",
			"location_service_code": "cloudauth",
			"regional_endpoints": [
				{
					"region": "cn-hangzhou",
					"endpoint": "cloudauth.aliyuncs.com"
				}
			],
			"global_endpoint": "cloudauth.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "vod",
			"document_id": "60574",
			"location_service_code": "vod",
			"regional_endpoints": [
				{
					"region": "cn-beijing",
					"endpoint": "vod.cn-shanghai.aliyuncs.com"
				},
				{
					"region": "ap-southeast-1",
					"endpoint": "vod.ap-southeast-1.aliyuncs.com"
				},
				{
					"region": "eu-central-1",
					"endpoint": "vod.eu-central-1.aliyuncs.com"
				},
				{
					"region": "cn-shanghai",
					"endpoint": "vod.cn-shanghai.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "vod.cn-shanghai.aliyuncs.com"
				},
				{
					"region": "cn-shenzhen",
					"endpoint": "vod.cn-shanghai.aliyuncs.com"
				}
			],
			"global_endpoint": "",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "ram",
			"document_id": "28672",
			"location_service_code": "ram",
			"regional_endpoints": null,
			"global_endpoint": "ram.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "ess",
			"document_id": "25925",
			"location_service_code": "ess",
			"regional_endpoints": [
				{
					"region": "me-east-1",
					"endpoint": "ess.me-east-1.aliyuncs.com"
				},
				{
					"region": "ap-northeast-1",
					"endpoint": "ess.ap-northeast-1.aliyuncs.com"
				},
				{
					"region": "ap-south-1",
					"endpoint": "ess.ap-south-1.aliyuncs.com"
				},
				{
					"region": "eu-central-1",
					"endpoint": "ess.eu-central-1.aliyuncs.com"
				},
				{
					"region": "cn-shanghai",
					"endpoint": "ess.aliyuncs.com"
				},
				{
					"region": "ap-southeast-1",
					"endpoint": "ess.aliyuncs.com"
				},
				{
					"region": "cn-huhehaote",
					"endpoint": "ess.cn-huhehaote.aliyuncs.com"
				},
				{
					"region": "ap-southeast-2",
					"endpoint": "ess.ap-southeast-2.aliyuncs.com"
				},
				{
					"region": "cn-beijing",
					"endpoint": "ess.aliyuncs.com"
				},
				{
					"region": "cn-hongkong",
					"endpoint": "ess.aliyuncs.com"
				},
				{
					"region": "us-west-1",
					"endpoint": "ess.aliyuncs.com"
				},
				{
					"region": "us-east-1",
					"endpoint": "ess.aliyuncs.com"
				},
				{
					"region": "ap-southeast-5",
					"endpoint": "ess.ap-southeast-5.aliyuncs.com"
				},
				{
					"region": "cn-qingdao",
					"endpoint": "ess.aliyuncs.com"
				},
				{
					"region": "ap-southeast-3",
					"endpoint": "ess.ap-southeast-3.aliyuncs.com"
				},
				{
					"region": "cn-zhangjiakou",
					"endpoint": "ess.cn-zhangjiakou.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "ess.aliyuncs.com"
				},
				{
					"region": "cn-shenzhen",
					"endpoint": "ess.aliyuncs.com"
				},
				{
					"region": "eu-west-1",
					"endpoint": "ess.eu-west-1.aliyuncs.com"
				}
			],
			"global_endpoint": "ess.aliyuncs.com",
			"regional_endpoint_pattern": "ess.[RegionId].aliyuncs.com"
		},
		{
			"code": "live",
			"document_id": "48207",
			"location_service_code": "live",
			"regional_endpoints": [
				{
					"region": "cn-beijing",
					"endpoint": "live.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "live.aliyuncs.com"
				},
				{
					"region": "ap-northeast-1",
					"endpoint": "live.aliyuncs.com"
				},
				{
					"region": "cn-shanghai",
					"endpoint": "live.aliyuncs.com"
				},
				{
					"region": "ap-southeast-1",
					"endpoint": "live.aliyuncs.com"
				},
				{
					"region": "eu-central-1",
					"endpoint": "live.aliyuncs.com"
				},
				{
					"region": "cn-shenzhen",
					"endpoint": "live.aliyuncs.com"
				}
			],
			"global_endpoint": "live.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "hpc",
			"document_id": "35201",
			"location_service_code": "hpc",
			"regional_endpoints": [
				{
					"region": "cn-hangzhou",
					"endpoint": "hpc.aliyuncs.com"
				},
				{
					"region": "cn-beijing",
					"endpoint": "hpc.aliyuncs.com"
				}
			],
			"global_endpoint": "hpc.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "rds",
			"document_id": "26223",
			"location_service_code": "rds",
			"regional_endpoints": [
				{
					"region": "me-east-1",
					"endpoint": "rds.me-east-1.aliyuncs.com"
				},
				{
					"region": "ap-south-1",
					"endpoint": "rds.ap-south-1.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "rds.aliyuncs.com"
				},
				{
					"region": "cn-shenzhen",
					"endpoint": "rds.aliyuncs.com"
				},
				{
					"region": "ap-southeast-1",
					"endpoint": "rds.aliyuncs.com"
				},
				{
					"region": "ap-southeast-3",
					"endpoint": "rds.ap-southeast-3.aliyuncs.com"
				},
				{
					"region": "ap-southeast-2",
					"endpoint": "rds.ap-southeast-2.aliyuncs.com"
				},
				{
					"region": "cn-zhangjiakou",
					"endpoint": "rds.cn-zhangjiakou.aliyuncs.com"
				},
				{
					"region": "cn-qingdao",
					"endpoint": "rds.aliyuncs.com"
				},
				{
					"region": "us-west-1",
					"endpoint": "rds.aliyuncs.com"
				},
				{
					"region": "us-east-1",
					"endpoint": "rds.aliyuncs.com"
				},
				{
					"region": "ap-southeast-5",
					"endpoint": "rds.ap-southeast-5.aliyuncs.com"
				},
				{
					"region": "eu-central-1",
					"endpoint": "rds.eu-central-1.aliyuncs.com"
				},
				{
					"region": "cn-beijing",
					"endpoint": "rds.aliyuncs.com"
				},
				{
					"region": "cn-shanghai",
					"endpoint": "rds.aliyuncs.com"
				},
				{
					"region": "eu-west-1",
					"endpoint": "rds.eu-west-1.aliyuncs.com"
				},
				{
					"region": "cn-huhehaote",
					"endpoint": "rds.cn-huhehaote.aliyuncs.com"
				},
				{
					"region": "ap-northeast-1",
					"endpoint": "rds.ap-northeast-1.aliyuncs.com"
				},
				{
					"region": "cn-hongkong",
					"endpoint": "rds.aliyuncs.com"
				}
			],
			"global_endpoint": "rds.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "cloudapi",
			"document_id": "43590",
			"location_service_code": "apigateway",
			"regional_endpoints": [
				{
					"region": "cn-beijing",
					"endpoint": "apigateway.cn-beijing.aliyuncs.com"
				},
				{
					"region": "ap-southeast-2",
					"endpoint": "apigateway.ap-southeast-2.aliyuncs.com"
				},
				{
					"region": "ap-south-1",
					"endpoint": "apigateway.ap-south-1.aliyuncs.com"
				},
				{
					"region": "us-east-1",
					"endpoint": "apigateway.us-east-1.aliyuncs.com"
				},
				{
					"region": "cn-shanghai",
					"endpoint": "apigateway.cn-shanghai.aliyuncs.com"
				},
				{
					"region": "us-west-1",
					"endpoint": "apigateway.us-west-1.aliyuncs.com"
				},
				{
					"region": "ap-southeast-1",
					"endpoint": "apigateway.ap-southeast-1.aliyuncs.com"
				},
				{
					"region": "eu-central-1",
					"endpoint": "apigateway.eu-central-1.aliyuncs.com"
				},
				{
					"region": "cn-qingdao",
					"endpoint": "apigateway.cn-qingdao.aliyuncs.com"
				},
				{
					"region": "cn-zhangjiakou",
					"endpoint": "apigateway.cn-zhangjiakou.aliyuncs.com"
				},
				{
					"region": "cn-huhehaote",
					"endpoint": "apigateway.cn-huhehaote.aliyuncs.com"
				},
				{
					"region": "eu-west-1",
					"endpoint": "apigateway.eu-west-1.aliyuncs.com"
				},
				{
					"region": "me-east-1",
					"endpoint": "apigateway.me-east-1.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "apigateway.cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "ap-northeast-1",
					"endpoint": "apigateway.ap-northeast-1.aliyuncs.com"
				},
				{
					"region": "ap-southeast-5",
					"endpoint": "apigateway.ap-southeast-5.aliyuncs.com"
				},
				{
					"region": "cn-hongkong",
					"endpoint": "apigateway.cn-hongkong.aliyuncs.com"
				},
				{
					"region": "cn-shenzhen",
					"endpoint": "apigateway.cn-shenzhen.aliyuncs.com"
				},
				{
					"region": "ap-southeast-3",
					"endpoint": "apigateway.ap-southeast-3.aliyuncs.com"
				}
			],
			"global_endpoint": "",
			"regional_endpoint_pattern": "apigateway.[RegionId].aliyuncs.com"
		},
		{
			"code": "sas-api",
			"document_id": "28498",
			"location_service_code": "sas",
			"regional_endpoints": [
				{
					"region": "cn-hangzhou",
					"endpoint": "sas.aliyuncs.com"
				}
			],
			"global_endpoint": "",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "cs",
			"document_id": "26043",
			"location_service_code": "cs",
			"regional_endpoints": null,
			"global_endpoint": "cs.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "jaq",
			"document_id": "35037",
			"location_service_code": "jaq",
			"regional_endpoints": null,
			"global_endpoint": "jaq.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "r-kvstore",
			"document_id": "60831",
			"location_service_code": "redisa",
			"regional_endpoints": [
				{
					"region": "cn-huhehaote",
					"endpoint": "r-kvstore.cn-huhehaote.aliyuncs.com"
				},
				{
					"region": "cn-zhangjiakou",
					"endpoint": "r-kvstore.cn-zhangjiakou.aliyuncs.com"
				},
				{
					"region": "cn-beijing",
					"endpoint": "r-kvstore.aliyuncs.com"
				},
				{
					"region": "cn-shanghai",
					"endpoint": "r-kvstore.aliyuncs.com"
				},
				{
					"region": "ap-south-1",
					"endpoint": "r-kvstore.ap-south-1.aliyuncs.com"
				},
				{
					"region": "eu-central-1",
					"endpoint": "r-kvstore.eu-central-1.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "r-kvstore.aliyuncs.com"
				},
				{
					"region": "cn-shenzhen",
					"endpoint": "r-kvstore.aliyuncs.com"
				},
				{
					"region": "me-east-1",
					"endpoint": "r-kvstore.me-east-1.aliyuncs.com"
				},
				{
					"region": "ap-northeast-1",
					"endpoint": "r-kvstore.ap-northeast-1.aliyuncs.com"
				},
				{
					"region": "cn-hongkong",
					"endpoint": "r-kvstore.cn-hongkong.aliyuncs.com"
				},
				{
					"region": "ap-southeast-2",
					"endpoint": "r-kvstore.ap-southeast-2.aliyuncs.com"
				},
				{
					"region": "eu-west-1",
					"endpoint": "r-kvstore.eu-west-1.aliyuncs.com"
				},
				{
					"region": "ap-southeast-5",
					"endpoint": "r-kvstore.ap-southeast-5.aliyuncs.com"
				},
				{
					"region": "us-west-1",
					"endpoint": "r-kvstore.aliyuncs.com"
				},
				{
					"region": "ap-southeast-1",
					"endpoint": "r-kvstore.ap-southeast-1.aliyuncs.com"
				},
				{
					"region": "ap-southeast-3",
					"endpoint": "r-kvstore.ap-southeast-3.aliyuncs.com"
				},
				{
					"region": "cn-qingdao",
					"endpoint": "r-kvstore.aliyuncs.com"
				},
				{
					"region": "us-east-1",
					"endpoint": "r-kvstore.aliyuncs.com"
				}
			],
			"global_endpoint": "r-kvstore.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "drds",
			"document_id": "51111",
			"location_service_code": "drds",
			"regional_endpoints": [
				{
					"region": "ap-southeast-1",
					"endpoint": "drds.ap-southeast-1.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "drds.cn-hangzhou.aliyuncs.com"
				}
			],
			"global_endpoint": "drds.aliyuncs.com",
			"regional_endpoint_pattern": "drds.aliyuncs.com"
		},
		{
			"code": "waf",
			"document_id": "62847",
			"location_service_code": "waf",
			"regional_endpoints": [
				{
					"region": "cn-hangzhou",
					"endpoint": "wafopenapi.cn-hangzhou.aliyuncs.com"
				}
			],
			"global_endpoint": "",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "sts",
			"document_id": "28756",
			"location_service_code": "sts",
			"regional_endpoints": null,
			"global_endpoint": "sts.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "cr",
			"document_id": "60716",
			"location_service_code": "cr",
			"regional_endpoints": null,
			"global_endpoint": "cr.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "arms",
			"document_id": "42924",
			"location_service_code": "arms",
			"regional_endpoints": [
				{
					"region": "cn-hangzhou",
					"endpoint": "arms.cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "cn-shanghai",
					"endpoint": "arms.cn-shanghai.aliyuncs.com"
				},
				{
					"region": "cn-hongkong",
					"endpoint": "arms.cn-hongkong.aliyuncs.com"
				},
				{
					"region": "ap-southeast-1",
					"endpoint": "arms.ap-southeast-1.aliyuncs.com"
				},
				{
					"region": "cn-shenzhen",
					"endpoint": "arms.cn-shenzhen.aliyuncs.com"
				},
				{
					"region": "cn-qingdao",
					"endpoint": "arms.cn-qingdao.aliyuncs.com"
				},
				{
					"region": "cn-beijing",
					"endpoint": "arms.cn-beijing.aliyuncs.com"
				}
			],
			"global_endpoint": "",
			"regional_endpoint_pattern": "arms.[RegionId].aliyuncs.com"
		},
		{
			"code": "iot",
			"document_id": "30557",
			"location_service_code": "iot",
			"regional_endpoints": [
				{
					"region": "us-east-1",
					"endpoint": "iot.us-east-1.aliyuncs.com"
				},
				{
					"region": "ap-northeast-1",
					"endpoint": "iot.ap-northeast-1.aliyuncs.com"
				},
				{
					"region": "us-west-1",
					"endpoint": "iot.us-west-1.aliyuncs.com"
				},
				{
					"region": "eu-central-1",
					"endpoint": "iot.eu-central-1.aliyuncs.com"
				},
				{
					"region": "cn-shanghai",
					"endpoint": "iot.cn-shanghai.aliyuncs.com"
				},
				{
					"region": "ap-southeast-1",
					"endpoint": "iot.ap-southeast-1.aliyuncs.com"
				}
			],
			"global_endpoint": "",
			"regional_endpoint_pattern": "iot.[RegionId].aliyuncs.com"
		},
		{
			"code": "vpc",
			"document_id": "34962",
			"location_service_code": "vpc",
			"regional_endpoints": [
				{
					"region": "us-west-1",
					"endpoint": "vpc.aliyuncs.com"
				},
				{
					"region": "us-east-1",
					"endpoint": "vpc.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "vpc.aliyuncs.com"
				},
				{
					"region": "cn-shenzhen",
					"endpoint": "vpc.aliyuncs.com"
				},
				{
					"region": "ap-southeast-1",
					"endpoint": "vpc.aliyuncs.com"
				},
				{
					"region": "cn-huhehaote",
					"endpoint": "vpc.cn-huhehaote.aliyuncs.com"
				},
				{
					"region": "me-east-1",
					"endpoint": "vpc.me-east-1.aliyuncs.com"
				},
				{
					"region": "ap-northeast-1",
					"endpoint": "vpc.ap-northeast-1.aliyuncs.com"
				},
				{
					"region": "ap-southeast-3",
					"endpoint": "vpc.ap-southeast-3.aliyuncs.com"
				},
				{
					"region": "eu-central-1",
					"endpoint": "vpc.eu-central-1.aliyuncs.com"
				},
				{
					"region": "ap-southeast-5",
					"endpoint": "vpc.ap-southeast-5.aliyuncs.com"
				},
				{
					"region": "ap-south-1",
					"endpoint": "vpc.ap-south-1.aliyuncs.com"
				},
				{
					"region": "cn-zhangjiakou",
					"endpoint": "vpc.cn-zhangjiakou.aliyuncs.com"
				},
				{
					"region": "cn-beijing",
					"endpoint": "vpc.aliyuncs.com"
				},
				{
					"region": "ap-southeast-2",
					"endpoint": "vpc.ap-southeast-2.aliyuncs.com"
				},
				{
					"region": "cn-qingdao",
					"endpoint": "vpc.aliyuncs.com"
				},
				{
					"region": "cn-shanghai",
					"endpoint": "vpc.aliyuncs.com"
				},
				{
					"region": "cn-hongkong",
					"endpoint": "vpc.aliyuncs.com"
				},
				{
					"region": "eu-west-1",
					"endpoint": "vpc.eu-west-1.aliyuncs.com"
				}
			],
			"global_endpoint": "vpc.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "aegis",
			"document_id": "28449",
			"location_service_code": "vipaegis",
			"regional_endpoints": [
				{
					"region": "ap-southeast-3",
					"endpoint": "aegis.ap-southeast-3.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "aegis.cn-hangzhou.aliyuncs.com"
				}
			],
			"global_endpoint": "aegis.cn-hangzhou.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "domain",
			"document_id": "42875",
			"location_service_code": "domain",
			"regional_endpoints": [
				{
					"region": "cn-hangzhou",
					"endpoint": "domain.aliyuncs.com"
				},
				{
					"region": "ap-southeast-1",
					"endpoint": "domain-intl.aliyuncs.com"
				}
			],
			"global_endpoint": "domain.aliyuncs.com",
			"regional_endpoint_pattern": "domain.aliyuncs.com"
		},
		{
			"code": "cdn",
			"document_id": "27148",
			"location_service_code": "cdn",
			"regional_endpoints": [
				{
					"region": "cn-hangzhou",
					"endpoint": "cdn.aliyuncs.com"
				}
			],
			"global_endpoint": "cdn.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "qualitycheck",
			"document_id": "50807",
			"location_service_code": "qualitycheck",
			"regional_endpoints": [
				{
					"region": "cn-hangzhou",
					"endpoint": "qualitycheck.cn-hangzhou.aliyuncs.com"
				}
			],
			"global_endpoint": "",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "emr",
			"document_id": "28140",
			"location_service_code": "emr",
			"regional_endpoints": [
				{
					"region": "us-east-1",
					"endpoint": "emr.us-east-1.aliyuncs.com"
				},
				{
					"region": "ap-southeast-5",
					"endpoint": "emr.ap-southeast-5.aliyuncs.com"
				},
				{
					"region": "eu-central-1",
					"endpoint": "emr.eu-central-1.aliyuncs.com"
				},
				{
					"region": "eu-west-1",
					"endpoint": "emr.eu-west-1.aliyuncs.com"
				},
				{
					"region": "us-west-1",
					"endpoint": "emr.aliyuncs.com"
				},
				{
					"region": "ap-southeast-1",
					"endpoint": "emr.aliyuncs.com"
				},
				{
					"region": "ap-south-1",
					"endpoint": "emr.ap-south-1.aliyuncs.com"
				},
				{
					"region": "me-east-1",
					"endpoint": "emr.me-east-1.aliyuncs.com"
				},
				{
					"region": "cn-beijing",
					"endpoint": "emr.aliyuncs.com"
				},
				{
					"region": "cn-shanghai",
					"endpoint": "emr.aliyuncs.com"
				},
				{
					"region": "cn-hongkong",
					"endpoint": "emr.cn-hongkong.aliyuncs.com"
				},
				{
					"region": "cn-huhehaote",
					"endpoint": "emr.cn-huhehaote.aliyuncs.com"
				},
				{
					"region": "ap-northeast-1",
					"endpoint": "emr.ap-northeast-1.aliyuncs.com"
				},
				{
					"region": "ap-southeast-3",
					"endpoint": "emr.ap-southeast-3.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "emr.aliyuncs.com"
				},
				{
					"region": "ap-southeast-2",
					"endpoint": "emr.ap-southeast-2.aliyuncs.com"
				},
				{
					"region": "cn-zhangjiakou",
					"endpoint": "emr.cn-zhangjiakou.aliyuncs.com"
				},
				{
					"region": "cn-qingdao",
					"endpoint": "emr.cn-qingdao.aliyuncs.com"
				},
				{
					"region": "cn-shenzhen",
					"endpoint": "emr.aliyuncs.com"
				}
			],
			"global_endpoint": "emr.aliyuncs.com",
			"regional_endpoint_pattern": "emr.[RegionId].aliyuncs.com"
		},
		{
			"code": "httpdns",
			"document_id": "52679",
			"location_service_code": "httpdns",
			"regional_endpoints": null,
			"global_endpoint": "httpdns-api.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "push",
			"document_id": "30074",
			"location_service_code": "push",
			"regional_endpoints": null,
			"global_endpoint": "cloudpush.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "cms",
			"document_id": "28615",
			"location_service_code": "cms",
			"regional_endpoints": [
				{
					"region": "cn-qingdao",
					"endpoint": "metrics.cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "metrics.cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "eu-west-1",
					"endpoint": "metrics.eu-west-1.aliyuncs.com"
				},
				{
					"region": "eu-central-1",
					"endpoint": "metrics.cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "ap-northeast-1",
					"endpoint": "metrics.ap-northeast-1.aliyuncs.com"
				},
				{
					"region": "ap-south-1",
					"endpoint": "metrics.ap-south-1.aliyuncs.com"
				},
				{
					"region": "cn-beijing",
					"endpoint": "metrics.cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "cn-shenzhen",
					"endpoint": "metrics.cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "ap-southeast-1",
					"endpoint": "metrics.cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "ap-southeast-2",
					"endpoint": "metrics.cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "ap-southeast-5",
					"endpoint": "metrics.ap-southeast-5.aliyuncs.com"
				},
				{
					"region": "cn-huhehaote",
					"endpoint": "metrics.cn-huhehaote.aliyuncs.com"
				},
				{
					"region": "cn-zhangjiakou",
					"endpoint": "metrics.cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "me-east-1",
					"endpoint": "metrics.cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "ap-southeast-3",
					"endpoint": "metrics.ap-southeast-3.aliyuncs.com"
				},
				{
					"region": "cn-shanghai",
					"endpoint": "metrics.cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "cn-hongkong",
					"endpoint": "metrics.cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "us-west-1",
					"endpoint": "metrics.cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "us-east-1",
					"endpoint": "metrics.cn-hangzhou.aliyuncs.com"
				}
			],
			"global_endpoint": "metrics.cn-hangzhou.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "nas",
			"document_id": "62598",
			"location_service_code": "nas",
			"regional_endpoints": [
				{
					"region": "ap-southeast-5",
					"endpoint": "nas.ap-southeast-5.aliyuncs.com"
				},
				{
					"region": "ap-south-1",
					"endpoint": "nas.ap-south-1.aliyuncs.com"
				},
				{
					"region": "us-west-1",
					"endpoint": "nas.us-west-1.aliyuncs.com"
				},
				{
					"region": "ap-southeast-3",
					"endpoint": "nas.ap-southeast-3.aliyuncs.com"
				},
				{
					"region": "cn-zhangjiakou",
					"endpoint": "nas.cn-zhangjiakou.aliyuncs.com"
				},
				{
					"region": "ap-northeast-1",
					"endpoint": "nas.ap-northeast-1.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "nas.cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "cn-qingdao",
					"endpoint": "nas.cn-qingdao.aliyuncs.com"
				},
				{
					"region": "cn-beijing",
					"endpoint": "nas.cn-beijing.aliyuncs.com"
				},
				{
					"region": "ap-southeast-2",
					"endpoint": "nas.ap-southeast-2.aliyuncs.com"
				},
				{
					"region": "cn-shenzhen",
					"endpoint": "nas.cn-shenzhen.aliyuncs.com"
				},
				{
					"region": "eu-central-1",
					"endpoint": "nas.eu-central-1.aliyuncs.com"
				},
				{
					"region": "cn-huhehaote",
					"endpoint": "nas.cn-huhehaote.aliyuncs.com"
				},
				{
					"region": "cn-shanghai",
					"endpoint": "nas.cn-shanghai.aliyuncs.com"
				},
				{
					"region": "cn-hongkong",
					"endpoint": "nas.cn-hongkong.aliyuncs.com"
				},
				{
					"region": "ap-southeast-1",
					"endpoint": "nas.ap-southeast-1.aliyuncs.com"
				},
				{
					"region": "us-east-1",
					"endpoint": "nas.us-east-1.aliyuncs.com"
				}
			],
			"global_endpoint": "",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "cds",
			"document_id": "62887",
			"location_service_code": "codepipeline",
			"regional_endpoints": [
				{
					"region": "cn-beijing",
					"endpoint": "cds.cn-beijing.aliyuncs.com"
				}
			],
			"global_endpoint": "cds.cn-beijing.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "green",
			"document_id": "28427",
			"location_service_code": "green",
			"regional_endpoints": [
				{
					"region": "us-west-1",
					"endpoint": "green.us-west-1.aliyuncs.com"
				},
				{
					"region": "cn-beijing",
					"endpoint": "green.cn-beijing.aliyuncs.com"
				},
				{
					"region": "ap-southeast-1",
					"endpoint": "green.ap-southeast-1.aliyuncs.com"
				},
				{
					"region": "cn-shanghai",
					"endpoint": "green.cn-shanghai.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "green.cn-hangzhou.aliyuncs.com"
				}
			],
			"global_endpoint": "green.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "ccc",
			"document_id": "63027",
			"location_service_code": "ccc",
			"regional_endpoints": [
				{
					"region": "cn-shanghai",
					"endpoint": "ccc.cn-shanghai.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "ccc.cn-hangzhou.aliyuncs.com"
				}
			],
			"global_endpoint": "",
			"regional_endpoint_pattern": "ccc.[RegionId].aliyuncs.com"
		},
		{
			"code": "ros",
			"document_id": "28899",
			"location_service_code": "ros",
			"regional_endpoints": [
				{
					"region": "cn-hangzhou",
					"endpoint": "ros.aliyuncs.com"
				}
			],
			"global_endpoint": "ros.aliyuncs.com",
			"regional_endpoint_pattern": ""
		},
		{
			"code": "mts",
			"document_id": "29212",
			"location_service_code": "mts",
			"regional_endpoints": [
				{
					"region": "ap-northeast-1",
					"endpoint": "mts.ap-northeast-1.aliyuncs.com"
				},
				{
					"region": "cn-shanghai",
					"endpoint": "mts.cn-shanghai.aliyuncs.com"
				},
				{
					"region": "cn-hongkong",
					"endpoint": "mts.cn-hongkong.aliyuncs.com"
				},
				{
					"region": "cn-shenzhen",
					"endpoint": "mts.cn-shenzhen.aliyuncs.com"
				},
				{
					"region": "us-west-1",
					"endpoint": "mts.us-west-1.aliyuncs.com"
				},
				{
					"region": "cn-zhangjiakou",
					"endpoint": "mts.cn-zhangjiakou.aliyuncs.com"
				},
				{
					"region": "eu-west-1",
					"endpoint": "mts.eu-west-1.aliyuncs.com"
				},
				{
					"region": "ap-south-1",
					"endpoint": "mts.ap-south-1.aliyuncs.com"
				},
				{
					"region": "cn-beijing",
					"endpoint": "mts.cn-beijing.aliyuncs.com"
				},
				{
					"region": "cn-hangzhou",
					"endpoint": "mts.cn-hangzhou.aliyuncs.com"
				},
				{
					"region": "ap-southeast-1",
					"endpoint": "mts.ap-southeast-1.aliyuncs.com"
				},
				{
					"region": "eu-central-1",
					"endpoint": "mts.eu-central-1.aliyuncs.com"
				}
			],
			"global_endpoint": "",
			"regional_endpoint_pattern": ""
		}
	]
}`
var initOnce sync.Once
var data interface{}

func getEndpointConfigData() interface{} {
	initOnce.Do(func() {
		err := json.Unmarshal([]byte(endpointsJson), &data)
		if err != nil {
			panic(fmt.Sprintf("init endpoint config data failed. %s", err))
		}
	})
	return data
}
