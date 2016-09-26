package spotinst

import (
	"fmt"
	"net/http"
)

// AwsGroupService handles communication with the AwsGroup related
// methods of the Spotinst API.
type AwsGroupService struct {
	client *Client
}

type AwsGroup struct {
	ID          *string              `json:"id,omitempty"`
	Name        *string              `json:"name,omitempty"`
	Description *string              `json:"description,omitempty"`
	Capacity    *AwsGroupCapacity    `json:"capacity,omitempty"`
	Compute     *AwsGroupCompute     `json:"compute,omitempty"`
	Strategy    *AwsGroupStrategy    `json:"strategy,omitempty"`
	Scaling     *AwsGroupScaling     `json:"scaling,omitempty"`
	Scheduling  *AwsGroupScheduling  `json:"scheduling,omitempty"`
	Integration *AwsGroupIntegration `json:"thirdPartiesIntegration,omitempty"`
}

type AwsGroupIntegration struct {
	EC2ContainerService *AwsGroupEC2ContainerServiceIntegration `json:"ecs,omitempty"`
	ElasticBeanstalk    *AwsGroupElasticBeanstalkIntegration    `json:"elasticBeanstalk,omitempty"`
	Rancher             *AwsGroupRancherIntegration             `json:"rancher,omitempty"`
	Kubernetes          *AwsGroupKubernetesIntegration          `json:"kubernetes,omitempty"`
}

type AwsGroupRancherIntegration struct {
	MasterHost *string `json:"masterHost,omitempty"`
	AccessKey  *string `json:"accessKey,omitempty"`
	SecretKey  *string `json:"secretKey,omitempty"`
}

type AwsGroupElasticBeanstalkIntegration struct {
	EnvironmentID *string `json:"environmentId,omitempty"`
}

type AwsGroupEC2ContainerServiceIntegration struct {
	ClusterName *string `json:"clusterName,omitempty"`
}

type AwsGroupKubernetesIntegration struct {
	Server *string `json:"apiServer,omitempty"`
	Token  *string `json:"token,omitempty"`
}

type AwsGroupScheduling struct {
	Tasks []*AwsGroupScheduledTask `json:"tasks,omitempty"`
}

type AwsGroupScheduledTask struct {
	Frequency           *string `json:"frequency,omitempty"`
	CronExpression      *string `json:"cronExpression,omitempty"`
	TaskType            *string `json:"taskType,omitempty"`
	ScaleTargetCapacity *int    `json:"scaleTargetCapacity,omitempty"`
	ScaleMinCapacity    *int    `json:"scaleMinCapacity,omitempty"`
	ScaleMaxCapacity    *int    `json:"scaleMaxCapacity,omitempty"`
	BatchSizePercentage *int    `json:"batchSizePercentage,omitempty"`
	GracePeriod         *int    `json:"gracePeriod,omitempty"`
}

type AwsGroupScaling struct {
	Up   []*AwsGroupScalingPolicy `json:"up,omitempty"`
	Down []*AwsGroupScalingPolicy `json:"down,omitempty"`
}

type AwsGroupScalingPolicy struct {
	PolicyName        *string                           `json:"policyName,omitempty"`
	MetricName        *string                           `json:"metricName,omitempty"`
	Statistic         *string                           `json:"statistic,omitempty"`
	Unit              *string                           `json:"unit,omitempty"`
	Threshold         *float64                          `json:"threshold,omitempty"`
	Adjustment        *int                              `json:"adjustment,omitempty"`
	MinTargetCapacity *int                              `json:"minTargetCapacity,omitempty"`
	MaxTargetCapacity *int                              `json:"maxTargetCapacity,omitempty"`
	Namespace         *string                           `json:"namespace,omitempty"`
	EvaluationPeriods *int                              `json:"evaluationPeriods,omitempty"`
	Period            *int                              `json:"period,omitempty"`
	Cooldown          *int                              `json:"cooldown,omitempty"`
	Operator          *string                           `json:"operator,omitempty"`
	Dimensions        []*AwsGroupScalingPolicyDimension `json:"dimensions,omitempty"`
}

type AwsGroupScalingPolicyDimension struct {
	Name  *string `json:"name,omitempty"`
	Value *string `json:"value,omitempty"`
}

type AwsGroupStrategy struct {
	Risk                     *float64                  `json:"risk,omitempty"`
	OnDemandCount            *int                      `json:"onDemandCount,omitempty"`
	DrainingTimeout          *int                      `json:"drainingTimeout,omitempty"`
	AvailabilityVsCost       *string                   `json:"availabilityVsCost,omitempty"`
	UtilizeReservedInstances *bool                     `json:"utilizeReservedInstances,omitempty"`
	FallbackToOnDemand       *bool                     `json:"fallbackToOd,omitempty"`
	Signals                  []*AwsGroupStrategySignal `json:"signals"`
}

type AwsGroupStrategySignal struct {
	Name *string `json:"name"`
}

type AwsGroupCapacity struct {
	Minimum *int    `json:"minimum,omitempty"`
	Maximum *int    `json:"maximum,omitempty"`
	Target  *int    `json:"target,omitempty"`
	Unit    *string `json:"unit,omitempty"`
}

type AwsGroupCompute struct {
	Product             *string                             `json:"product,omitempty"`
	InstanceTypes       *AwsGroupComputeInstanceType        `json:"instanceTypes,omitempty"`
	LaunchSpecification *AwsGroupComputeLaunchSpecification `json:"launchSpecification,omitempty"`
	AvailabilityZones   []*AwsGroupComputeAvailabilityZone  `json:"availabilityZones,omitempty"`
	ElasticIPs          []string                            `json:"elasticIps,omitempty"`
	EBSVolumePool       []*AwsGroupComputeEBSVolume         `json:"ebsVolumePool,omitempty"`
}

type AwsGroupComputeEBSVolume struct {
	DeviceName *string  `json:"deviceName,omitempty"`
	VolumeIDs  []string `json:"volumeIds,omitempty"`
}

type AwsGroupComputeInstanceType struct {
	OnDemand *string                              `json:"ondemand,omitempty"`
	Spot     []string                             `json:"spot,omitempty"`
	Weights  []*AwsGroupComputeInstanceTypeWeight `json:"weights,omitempty"`
}

type AwsGroupComputeInstanceTypeWeight struct {
	InstanceType *string `json:"instanceType,omitempty"`
	Weight       *int    `json:"weightedCapacity,omitempty"`
}

type AwsGroupComputeAvailabilityZone struct {
	Name     *string `json:"name,omitempty"`
	SubnetID *string `json:"subnetId,omitempty"`
}

type AwsGroupComputeLaunchSpecification struct {
	LoadBalancerNames      []string                            `json:"loadBalancerNames,omitempty"`
	LoadBalancersConfig    *AwsGroupComputeLoadBalancersConfig `json:"loadBalancersConfig,omitempty"`
	SecurityGroupIDs       []string                            `json:"securityGroupIds,omitempty"`
	HealthCheckType        *string                             `json:"healthCheckType,omitempty"`
	HealthCheckGracePeriod *int                                `json:"healthCheckGracePeriod,omitempty"`
	ImageID                *string                             `json:"imageId,omitempty"`
	KeyPair                *string                             `json:"keyPair,omitempty"`
	UserData               *string                             `json:"userData,omitempty"`
	Monitoring             *bool                               `json:"monitoring,omitempty"`
	IamInstanceProfile     *AwsGroupComputeIamInstanceProfile  `json:"iamRole,omitempty"`
	BlockDevices           []*AwsGroupComputeBlockDevice       `json:"blockDeviceMappings,omitempty"`
	NetworkInterfaces      []*AwsGroupComputeNetworkInterface  `json:"networkInterfaces,omitempty"`
	Tags                   []*AwsGroupComputeTag               `json:"tags,omitempty"`
}

type AwsGroupComputeLoadBalancersConfig struct {
	LoadBalancers []*AwsGroupComputeLoadBalancer `json:"loadBalancers,omitempty"`
}

type AwsGroupComputeLoadBalancer struct {
	Name *string `json:"name,omitempty"`
	Arn  *string `json:"arn,omitempty"`
	Type *string `json:"type,omitempty"`
}

type AwsGroupComputeNetworkInterface struct {
	ID                             *string  `json:"networkInterfaceId,omitempty"`
	Description                    *string  `json:"description,omitempty"`
	DeviceIndex                    *int     `json:"deviceIndex,omitempty"`
	SecondaryPrivateIPAddressCount *int     `json:"secondaryPrivateIpAddressCount,omitempty"`
	AssociatePublicIPAddress       *bool    `json:"associatePublicIpAddress,omitempty"`
	DeleteOnTermination            *bool    `json:"deleteOnTermination,omitempty"`
	SecurityGroupsIDs              []string `json:"groups,omitempty"`
	PrivateIPAddress               *string  `json:"privateIpAddress,omitempty"`
	SubnetID                       *string  `json:"subnetId,omitempty"`
}

type AwsGroupComputeBlockDevice struct {
	DeviceName  *string             `json:"deviceName,omitempty"`
	VirtualName *string             `json:"virtualName,omitempty"`
	EBS         *AwsGroupComputeEBS `json:"ebs,omitempty"`
}

type AwsGroupComputeEBS struct {
	DeleteOnTermination *bool   `json:"deleteOnTermination,omitempty"`
	Encrypted           *bool   `json:"encrypted,omitempty"`
	SnapshotID          *string `json:"snapshotId,omitempty"`
	VolumeType          *string `json:"volumeType,omitempty"`
	VolumeSize          *int    `json:"volumeSize,omitempty"`
	IOPS                *int    `json:"iops,omitempty"`
}

type AwsGroupComputeIamInstanceProfile struct {
	Name *string `json:"name,omitempty"`
	Arn  *string `json:"arn,omitempty"`
}

type AwsGroupComputeTag struct {
	Key   *string `json:"tagKey,omitempty"`
	Value *string `json:"tagValue,omitempty"`
}

type AwsGroupResponse struct {
	Response struct {
		Errors []Error     `json:"errors"`
		Items  []*AwsGroup `json:"items"`
	} `json:"response"`
}

type groupWrapper struct {
	Group AwsGroup `json:"group"`
}

// Get an existing group.
func (s *AwsGroupService) Get(args ...string) ([]*AwsGroup, *http.Response, error) {
	var gid string
	if len(args) > 0 {
		gid = args[0]
	}

	path := fmt.Sprintf("aws/ec2/group/%s", gid)
	req, err := s.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	var retval AwsGroupResponse
	resp, err := s.client.Do(req, &retval)
	if err != nil {
		return nil, resp, err
	}

	return retval.Response.Items, resp, err
}

// Create a new group.
func (s *AwsGroupService) Create(group *AwsGroup) ([]*AwsGroup, *http.Response, error) {
	path := "aws/ec2/group"

	req, err := s.client.NewRequest("POST", path, groupWrapper{Group: *group})
	if err != nil {
		return nil, nil, err
	}

	var retval AwsGroupResponse
	resp, err := s.client.Do(req, &retval)
	if err != nil {
		return nil, resp, err
	}

	return retval.Response.Items, resp, nil
}

// Update an existing group.
func (s *AwsGroupService) Update(group *AwsGroup) ([]*AwsGroup, *http.Response, error) {
	gid := (*group).ID
	(*group).ID = nil
	path := fmt.Sprintf("aws/ec2/group/%s", *gid)

	req, err := s.client.NewRequest("PUT", path, groupWrapper{Group: *group})
	if err != nil {
		return nil, nil, err
	}

	var retval AwsGroupResponse
	resp, err := s.client.Do(req, &retval)
	if err != nil {
		return nil, resp, err
	}

	return retval.Response.Items, resp, nil
}

// Delete an existing group.
func (s *AwsGroupService) Delete(group *AwsGroup) (*http.Response, error) {
	gid := (*group).ID
	(*group).ID = nil
	path := fmt.Sprintf("aws/ec2/group/%s", *gid)

	req, err := s.client.NewRequest("DELETE", path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req, nil)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// String creates a reasonable string representation.
func (a AwsGroup) String() string {
	return Stringify(a)
}
