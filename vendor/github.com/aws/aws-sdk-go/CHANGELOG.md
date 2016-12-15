Release v1.6.3 (2016-12-14)
===

Service Client Updates
---
* `service/batch`: Adds new service
  * AWS Batch is a batch computing service that lets customers define queues and compute environments and then submit work as batch jobs.
* `service/databasemigrationservice`: Updates service API and documentation
  * Adds support for SSL enabled Oracle endpoints and task modification.
* `service/elasticbeanstalk`: Updates service documentation
* `aws/endpoints`: Updated Regions and Endpoints metadata.
* `service/cloudwatchlogs`: Updates service API and documentation
  * Add support for associating LogGroups with AWSTagris tags
* `service/marketplacecommerceanalytics`: Updates service API and documentation
  * Add new enum to DataSetType: sales_compensation_billed_revenue
* `service/rds`: Updates service documentation
  * Doc-only Update for RDS: New versions available in CreateDBInstance
* `service/sts`: Updates service documentation
  * Adding Code Snippet Examples for SDKs for STS

SDK Bug Fixes
---
* `aws/request`: Fix retrying timeout requests (#981)
  * Fixes: Requests Retrying is broken if the error was caused due to a client timeout #947
* `aws/request`: Fix for Go 1.8 request incorrectly sent with body (#991)
  * Fixes: service/route53: ListHostedZones hangs and then fails with go1.8 #984
* private/protocol/rest: Use RawPath instead of Opaque (#993)
  * Fixes: HTTP2 request failing with REST protocol services, e.g AWS X-Ray
* private/model/api: Generate REST-JSON JSONVersion correctly (#998)
  * Fixes: REST-JSON protocol service code missing JSONVersion metadata.

Release v1.6.2 (2016-12-08)
===

Service Client Updates
---
* `service/cloudfront`: Add lambda function associations to cache behaviors
* `service/codepipeline`: This is a doc-only update request to incorporate some recent minor revisions to the doc content.
* `service/rds`: Updates service API and documentation
* `service/wafregional`: With this new feature, customers can use AWS WAF directly on Application Load Balancers in a VPC within available regions to protect their websites and web services from malicious attacks such as SQL injection, Cross Site Scripting, bad bots, etc.

Release v1.6.1 (2016-12-07)
===

Service Client Updates
---
* `service/config`: Updates service API
* `service/s3`: Updates service API
* `service/sqs`: Updates service API and documentation

Release v1.6.0 (2016-12-06)
===

Service Client Updates
---
* `service/config`: Updates service API and documentation
* `service/ec2`: Updates service API
* `service/sts`: Updates service API, documentation, and examples

SDK Bug Fixes
---
* private/protocol/xml/xmlutil: Fix SDK XML unmarshaler #975
  * Fixes GetBucketACL Grantee required type always nil. #916

SDK Feature
---
* aws/endpoints: Add endpoint metadata to SDK #961
  * Adds Region and Endpoint metadata to the SDK. This allows you to enumerate regions and endpoint metadata based on a defined model embedded in the SDK.

Release v1.5.13 (2016-12-01)
===

Service Client Updates
---
* `service/apigateway`: Updates service API and documentation
* `service/appstream`: Adds new service
* `service/codebuild`: Adds new service
* `service/directconnect`: Updates service API and documentation
* `service/ec2`: Adds new service
* `service/elasticbeanstalk`: Updates service API and documentation
* `service/health`: Adds new service
* `service/lambda`: Updates service API and documentation
* `service/opsworkscm`: Adds new service
* `service/pinpoint`: Adds new service
* `service/shield`: Adds new service
* `service/ssm`: Updates service API and documentation
* `service/states`: Adds new service
* `service/xray`: Adds new service

Release v1.5.12 (2016-11-30)
===

Service Client Updates
---
* `service/lightsail`: Adds new service
* `service/polly`: Adds new service
* `service/rekognition`: Adds new service
* `service/snowball`: Updates service API and documentation

Release v1.5.11 (2016-11-29)
===

Service Client Updates
---
`service/s3`: Updates service API and documentation

Release v1.5.10 (2016-11-22)
===

Service Client Updates
---
* `service/cloudformation`: Updates service API and documentation
* `service/glacier`: Updates service API, documentation, and examples
* `service/route53`: Updates service API and documentation
* `service/s3`: Updates service API and documentation

SDK Bug Fixes
---
* `private/protocol/xml/xmlutil`: Fixes xml marshaler to unmarshal properly
into tagged fields 
[#916](https://github.com/aws/aws-sdk-go/issues/916)

Release v1.5.9 (2016-11-22)
===

Service Client Updates
---
* `service/cloudtrail`: Updates service API and documentation
* `service/ecs`: Updates service API and documentation

Release v1.5.8 (2016-11-18)
===

Service Client Updates
---
* `service/application-autoscaling`: Updates service API and documentation
* `service/elasticmapreduce`: Updates service API and documentation
* `service/elastictranscoder`: Updates service API, documentation, and examples
* `service/gamelift`: Updates service API and documentation
* `service/lambda`: Updates service API and documentation

Release v1.5.7 (2016-11-18)
===

Service Client Updates
---
* `service/apigateway`: Updates service API and documentation
* `service/meteringmarketplace`: Updates service API and documentation
* `service/monitoring`: Updates service API and documentation
* `service/sqs`: Updates service API, documentation, and examples

Release v1.5.6 (2016-11-16)
===

Service Client Updates
---
`service/route53`: Updates service API and documentation
`service/servicecatalog`: Updates service API and documentation

Release v1.5.5 (2016-11-15)
===

Service Client Updates
---
* `service/ds`: Updates service API and documentation
* `service/elasticache`: Updates service API and documentation
* `service/kinesis`: Updates service API and documentation

Release v1.5.4 (2016-11-15)
===

Service Client Updates
---
* `service/cognito-idp`: Updates service API and documentation

Release v1.5.3 (2016-11-11)
===

Service Client Updates
---
* `service/cloudformation`: Updates service documentation and examples
* `service/logs`: Updates service API and documentation

Release v1.5.2 (2016-11-03)
===

Service Client Updates
---
* `service/directconnect`: Updates service API and documentation

Release v1.5.1 (2016-11-02)
===

Service Client Updates
---
* `service/email`: Updates service API and documentation

Release v1.5.0 (2016-11-01)
===

Service Client Updates
---
* `service/cloudformation`: Updates service API and documentation
* `service/ecr`: Updates service paginators

SDK Feature Updates
---
* `private/model/api`: Add generated setters for API parameters (#918)
  * Adds setters to the SDK's API parameter types, and are a convenience method that reduce the need to use `aws.String` and like utility. 

Release v1.4.22 (2016-10-25)
===

Service Client Updates
---
* `service/elasticloadbalancingv2`: Updates service documentation.
* `service/autoscaling`: Updates service documentation.

Release v1.4.21 (2016-10-24)
===

Service Client Updates
---
* `service/sms`: AWS Server Migration Service (SMS) is an agentless service which makes it easier and faster for you to migrate thousands of on-premises workloads to AWS. AWS SMS allows you to automate, schedule, and track incremental replications of live server volumes, making it easier for you to coordinate large-scale server migrations.
* `service/ecs`: Updates documentation.

SDK Feature Updates
---
* `private/models/api`: Improve code generation of documentation.

Release v1.4.20 (2016-10-20)
===

Service Client Updates
---
* `service/budgets`: Adds new service, AWS Budgets.
* `service/waf`: Updates service documentation.

Release v1.4.19 (2016-10-18)
===

Service Client Updates
---
* `service/cloudfront`: Updates service API and documentation.
  * Ability to use Amazon CloudFront to deliver your content both via IPv6 and IPv4 using HTTP/HTTPS.
* `service/configservice`: Update service API and documentation.
* `service/iot`: Updates service API and documentation.
* `service/kinesisanalytics`: Updates service API and documentation.
  * Whenever Amazon Kinesis Analytics is not able to detect schema for the given streaming source on DiscoverInputSchema API, we would return the raw records that was sampled to detect the schema.
* `service/rds`: Updates service API and documentation.
  * Amazon Aurora integrates with other AWS services to allow you to extend your Aurora DB cluster to utilize other capabilities in the AWS cloud. Permission to access other AWS services is granted by creating an IAM role with the necessary permissions, and then associating the role with your DB cluster.

SDK Feature Updates
---
* `service/dynamodb/dynamodbattribute`: Add UnmarshalListOfMaps #897
  * Adds support for unmarshaling a list of maps. This is useful for unmarshaling the DynamoDB AttributeValue list of maps returned by APIs like Query and Scan.

Release v1.4.18 (2016-10-17)
===

Service Model Updates
---
* `service/route53`: Updates service API and documentation.

Release v1.4.17
===

Service Model Updates
---
* `service/acm`: Update service API, and documentation.
  * This change allows users to import third-party SSL/TLS certificates into ACM.
* `service/elasticbeanstalk`: Update service API, documentation, and pagination.
  * Elastic Beanstalk DescribeApplicationVersions API is being updated to support pagination.
* `service/gamelift`: Update service API, and documentation.
  * New APIs to protect game developer resource (builds, alias, fleets, instances, game sessions and player sessions) against abuse.

SDK Features
---
* `service/s3`: Add support for accelerate with dualstack [#887](https://github.com/aws/aws-sdk-go/issues/887)

Release v1.4.16 (2016-10-13)
===

Service Model Updates
---
* `service/ecr`: Update Amazon EC2 Container Registry service model
  * DescribeImages is a new api used to expose image metadata which today includes image size and image creation timestamp.
* `service/elasticache`: Update Amazon ElastiCache service model
  * Elasticache is launching a new major engine release of Redis, 3.2 (providing stability updates and new command sets over 2.8), as well as ElasticSupport for enabling Redis Cluster in 3.2, which provides support for multiple node groups to horizontally scale data, as well as superior engine failover capabilities 

SDK Bug Fixes
---
* `aws/session`: Skip shared config on read errors [#883](https://github.com/aws/aws-sdk-go/issues/883)
* `aws/signer/v4`: Add support for URL.EscapedPath to signer [#885](https://github.com/aws/aws-sdk-go/issues/885)

SDK Features
---
* `private/model/api`: Add docs for errors to API operations [#881](https://github.com/aws/aws-sdk-go/issues/881)
* `private/model/api`: Improve field and waiter doc strings [#879](https://github.com/aws/aws-sdk-go/issues/879)
* `service/dynamodb/dynamodbattribute`: Allow multiple struct tag elements [#886](https://github.com/aws/aws-sdk-go/issues/886)
* Add build tags to internal SDK tools [#880](https://github.com/aws/aws-sdk-go/issues/880)

Release v1.4.15 (2016-10-06)
===

Service Model Updates
---
* `service/cognitoidentityprovider`: Update Amazon Cognito Identity Provider service model
* `service/devicefarm`: Update AWS Device Farm documentation
* `service/opsworks`: Update AWS OpsWorks service model
* `service/s3`: Update Amazon Simple Storage Service model
* `service/waf`: Update AWS WAF service model

SDK Bug Fixes
---
* `aws/request`: Fix HTTP Request Body race condition [#874](https://github.com/aws/aws-sdk-go/issues/874)

SDK Feature Updates
---
* `aws/ec2metadata`: Add support for EC2 User Data [#872](https://github.com/aws/aws-sdk-go/issues/872)
* `aws/signer/v4`: Remove logic determining if request needs to be resigned [#876](https://github.com/aws/aws-sdk-go/issues/876)

Release v1.4.14 (2016-09-29)
===
* `service/ec2`:  api, documentation, and paginators updates.
* `service/s3`:  api and documentation updates.

Release v1.4.13 (2016-09-27)
===
* `service/codepipeline`:  documentation updates.
* `service/cloudformation`:  api and documentation updates.
* `service/kms`:  documentation updates.
* `service/elasticfilesystem`:  documentation updates.
* `service/snowball`:  documentation updates.
