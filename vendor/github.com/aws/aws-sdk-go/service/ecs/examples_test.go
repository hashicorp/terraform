// THIS FILE IS AUTOMATICALLY GENERATED. DO NOT EDIT.

package ecs_test

import (
	"bytes"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
)

var _ time.Duration
var _ bytes.Buffer

func ExampleECS_CreateCluster() {
	svc := ecs.New(session.New())

	params := &ecs.CreateClusterInput{
		ClusterName: aws.String("String"),
	}
	resp, err := svc.CreateCluster(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_CreateService() {
	svc := ecs.New(session.New())

	params := &ecs.CreateServiceInput{
		DesiredCount:   aws.Int64(1),         // Required
		ServiceName:    aws.String("String"), // Required
		TaskDefinition: aws.String("String"), // Required
		ClientToken:    aws.String("String"),
		Cluster:        aws.String("String"),
		DeploymentConfiguration: &ecs.DeploymentConfiguration{
			MaximumPercent:        aws.Int64(1),
			MinimumHealthyPercent: aws.Int64(1),
		},
		LoadBalancers: []*ecs.LoadBalancer{
			{ // Required
				ContainerName:    aws.String("String"),
				ContainerPort:    aws.Int64(1),
				LoadBalancerName: aws.String("String"),
			},
			// More values...
		},
		Role: aws.String("String"),
	}
	resp, err := svc.CreateService(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_DeleteCluster() {
	svc := ecs.New(session.New())

	params := &ecs.DeleteClusterInput{
		Cluster: aws.String("String"), // Required
	}
	resp, err := svc.DeleteCluster(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_DeleteService() {
	svc := ecs.New(session.New())

	params := &ecs.DeleteServiceInput{
		Service: aws.String("String"), // Required
		Cluster: aws.String("String"),
	}
	resp, err := svc.DeleteService(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_DeregisterContainerInstance() {
	svc := ecs.New(session.New())

	params := &ecs.DeregisterContainerInstanceInput{
		ContainerInstance: aws.String("String"), // Required
		Cluster:           aws.String("String"),
		Force:             aws.Bool(true),
	}
	resp, err := svc.DeregisterContainerInstance(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_DeregisterTaskDefinition() {
	svc := ecs.New(session.New())

	params := &ecs.DeregisterTaskDefinitionInput{
		TaskDefinition: aws.String("String"), // Required
	}
	resp, err := svc.DeregisterTaskDefinition(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_DescribeClusters() {
	svc := ecs.New(session.New())

	params := &ecs.DescribeClustersInput{
		Clusters: []*string{
			aws.String("String"), // Required
			// More values...
		},
	}
	resp, err := svc.DescribeClusters(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_DescribeContainerInstances() {
	svc := ecs.New(session.New())

	params := &ecs.DescribeContainerInstancesInput{
		ContainerInstances: []*string{ // Required
			aws.String("String"), // Required
			// More values...
		},
		Cluster: aws.String("String"),
	}
	resp, err := svc.DescribeContainerInstances(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_DescribeServices() {
	svc := ecs.New(session.New())

	params := &ecs.DescribeServicesInput{
		Services: []*string{ // Required
			aws.String("String"), // Required
			// More values...
		},
		Cluster: aws.String("String"),
	}
	resp, err := svc.DescribeServices(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_DescribeTaskDefinition() {
	svc := ecs.New(session.New())

	params := &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String("String"), // Required
	}
	resp, err := svc.DescribeTaskDefinition(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_DescribeTasks() {
	svc := ecs.New(session.New())

	params := &ecs.DescribeTasksInput{
		Tasks: []*string{ // Required
			aws.String("String"), // Required
			// More values...
		},
		Cluster: aws.String("String"),
	}
	resp, err := svc.DescribeTasks(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_DiscoverPollEndpoint() {
	svc := ecs.New(session.New())

	params := &ecs.DiscoverPollEndpointInput{
		Cluster:           aws.String("String"),
		ContainerInstance: aws.String("String"),
	}
	resp, err := svc.DiscoverPollEndpoint(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_ListClusters() {
	svc := ecs.New(session.New())

	params := &ecs.ListClustersInput{
		MaxResults: aws.Int64(1),
		NextToken:  aws.String("String"),
	}
	resp, err := svc.ListClusters(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_ListContainerInstances() {
	svc := ecs.New(session.New())

	params := &ecs.ListContainerInstancesInput{
		Cluster:    aws.String("String"),
		MaxResults: aws.Int64(1),
		NextToken:  aws.String("String"),
	}
	resp, err := svc.ListContainerInstances(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_ListServices() {
	svc := ecs.New(session.New())

	params := &ecs.ListServicesInput{
		Cluster:    aws.String("String"),
		MaxResults: aws.Int64(1),
		NextToken:  aws.String("String"),
	}
	resp, err := svc.ListServices(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_ListTaskDefinitionFamilies() {
	svc := ecs.New(session.New())

	params := &ecs.ListTaskDefinitionFamiliesInput{
		FamilyPrefix: aws.String("String"),
		MaxResults:   aws.Int64(1),
		NextToken:    aws.String("String"),
	}
	resp, err := svc.ListTaskDefinitionFamilies(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_ListTaskDefinitions() {
	svc := ecs.New(session.New())

	params := &ecs.ListTaskDefinitionsInput{
		FamilyPrefix: aws.String("String"),
		MaxResults:   aws.Int64(1),
		NextToken:    aws.String("String"),
		Sort:         aws.String("SortOrder"),
		Status:       aws.String("TaskDefinitionStatus"),
	}
	resp, err := svc.ListTaskDefinitions(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_ListTasks() {
	svc := ecs.New(session.New())

	params := &ecs.ListTasksInput{
		Cluster:           aws.String("String"),
		ContainerInstance: aws.String("String"),
		DesiredStatus:     aws.String("DesiredStatus"),
		Family:            aws.String("String"),
		MaxResults:        aws.Int64(1),
		NextToken:         aws.String("String"),
		ServiceName:       aws.String("String"),
		StartedBy:         aws.String("String"),
	}
	resp, err := svc.ListTasks(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_RegisterContainerInstance() {
	svc := ecs.New(session.New())

	params := &ecs.RegisterContainerInstanceInput{
		Attributes: []*ecs.Attribute{
			{ // Required
				Name:  aws.String("String"), // Required
				Value: aws.String("String"),
			},
			// More values...
		},
		Cluster:                           aws.String("String"),
		ContainerInstanceArn:              aws.String("String"),
		InstanceIdentityDocument:          aws.String("String"),
		InstanceIdentityDocumentSignature: aws.String("String"),
		TotalResources: []*ecs.Resource{
			{ // Required
				DoubleValue:  aws.Float64(1.0),
				IntegerValue: aws.Int64(1),
				LongValue:    aws.Int64(1),
				Name:         aws.String("String"),
				StringSetValue: []*string{
					aws.String("String"), // Required
					// More values...
				},
				Type: aws.String("String"),
			},
			// More values...
		},
		VersionInfo: &ecs.VersionInfo{
			AgentHash:     aws.String("String"),
			AgentVersion:  aws.String("String"),
			DockerVersion: aws.String("String"),
		},
	}
	resp, err := svc.RegisterContainerInstance(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_RegisterTaskDefinition() {
	svc := ecs.New(session.New())

	params := &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{ // Required
			{ // Required
				Command: []*string{
					aws.String("String"), // Required
					// More values...
				},
				Cpu:               aws.Int64(1),
				DisableNetworking: aws.Bool(true),
				DnsSearchDomains: []*string{
					aws.String("String"), // Required
					// More values...
				},
				DnsServers: []*string{
					aws.String("String"), // Required
					// More values...
				},
				DockerLabels: map[string]*string{
					"Key": aws.String("String"), // Required
					// More values...
				},
				DockerSecurityOptions: []*string{
					aws.String("String"), // Required
					// More values...
				},
				EntryPoint: []*string{
					aws.String("String"), // Required
					// More values...
				},
				Environment: []*ecs.KeyValuePair{
					{ // Required
						Name:  aws.String("String"),
						Value: aws.String("String"),
					},
					// More values...
				},
				Essential: aws.Bool(true),
				ExtraHosts: []*ecs.HostEntry{
					{ // Required
						Hostname:  aws.String("String"), // Required
						IpAddress: aws.String("String"), // Required
					},
					// More values...
				},
				Hostname: aws.String("String"),
				Image:    aws.String("String"),
				Links: []*string{
					aws.String("String"), // Required
					// More values...
				},
				LogConfiguration: &ecs.LogConfiguration{
					LogDriver: aws.String("LogDriver"), // Required
					Options: map[string]*string{
						"Key": aws.String("String"), // Required
						// More values...
					},
				},
				Memory: aws.Int64(1),
				MountPoints: []*ecs.MountPoint{
					{ // Required
						ContainerPath: aws.String("String"),
						ReadOnly:      aws.Bool(true),
						SourceVolume:  aws.String("String"),
					},
					// More values...
				},
				Name: aws.String("String"),
				PortMappings: []*ecs.PortMapping{
					{ // Required
						ContainerPort: aws.Int64(1),
						HostPort:      aws.Int64(1),
						Protocol:      aws.String("TransportProtocol"),
					},
					// More values...
				},
				Privileged:             aws.Bool(true),
				ReadonlyRootFilesystem: aws.Bool(true),
				Ulimits: []*ecs.Ulimit{
					{ // Required
						HardLimit: aws.Int64(1),             // Required
						Name:      aws.String("UlimitName"), // Required
						SoftLimit: aws.Int64(1),             // Required
					},
					// More values...
				},
				User: aws.String("String"),
				VolumesFrom: []*ecs.VolumeFrom{
					{ // Required
						ReadOnly:        aws.Bool(true),
						SourceContainer: aws.String("String"),
					},
					// More values...
				},
				WorkingDirectory: aws.String("String"),
			},
			// More values...
		},
		Family: aws.String("String"), // Required
		Volumes: []*ecs.Volume{
			{ // Required
				Host: &ecs.HostVolumeProperties{
					SourcePath: aws.String("String"),
				},
				Name: aws.String("String"),
			},
			// More values...
		},
	}
	resp, err := svc.RegisterTaskDefinition(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_RunTask() {
	svc := ecs.New(session.New())

	params := &ecs.RunTaskInput{
		TaskDefinition: aws.String("String"), // Required
		Cluster:        aws.String("String"),
		Count:          aws.Int64(1),
		Overrides: &ecs.TaskOverride{
			ContainerOverrides: []*ecs.ContainerOverride{
				{ // Required
					Command: []*string{
						aws.String("String"), // Required
						// More values...
					},
					Environment: []*ecs.KeyValuePair{
						{ // Required
							Name:  aws.String("String"),
							Value: aws.String("String"),
						},
						// More values...
					},
					Name: aws.String("String"),
				},
				// More values...
			},
		},
		StartedBy: aws.String("String"),
	}
	resp, err := svc.RunTask(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_StartTask() {
	svc := ecs.New(session.New())

	params := &ecs.StartTaskInput{
		ContainerInstances: []*string{ // Required
			aws.String("String"), // Required
			// More values...
		},
		TaskDefinition: aws.String("String"), // Required
		Cluster:        aws.String("String"),
		Overrides: &ecs.TaskOverride{
			ContainerOverrides: []*ecs.ContainerOverride{
				{ // Required
					Command: []*string{
						aws.String("String"), // Required
						// More values...
					},
					Environment: []*ecs.KeyValuePair{
						{ // Required
							Name:  aws.String("String"),
							Value: aws.String("String"),
						},
						// More values...
					},
					Name: aws.String("String"),
				},
				// More values...
			},
		},
		StartedBy: aws.String("String"),
	}
	resp, err := svc.StartTask(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_StopTask() {
	svc := ecs.New(session.New())

	params := &ecs.StopTaskInput{
		Task:    aws.String("String"), // Required
		Cluster: aws.String("String"),
		Reason:  aws.String("String"),
	}
	resp, err := svc.StopTask(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_SubmitContainerStateChange() {
	svc := ecs.New(session.New())

	params := &ecs.SubmitContainerStateChangeInput{
		Cluster:       aws.String("String"),
		ContainerName: aws.String("String"),
		ExitCode:      aws.Int64(1),
		NetworkBindings: []*ecs.NetworkBinding{
			{ // Required
				BindIP:        aws.String("String"),
				ContainerPort: aws.Int64(1),
				HostPort:      aws.Int64(1),
				Protocol:      aws.String("TransportProtocol"),
			},
			// More values...
		},
		Reason: aws.String("String"),
		Status: aws.String("String"),
		Task:   aws.String("String"),
	}
	resp, err := svc.SubmitContainerStateChange(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_SubmitTaskStateChange() {
	svc := ecs.New(session.New())

	params := &ecs.SubmitTaskStateChangeInput{
		Cluster: aws.String("String"),
		Reason:  aws.String("String"),
		Status:  aws.String("String"),
		Task:    aws.String("String"),
	}
	resp, err := svc.SubmitTaskStateChange(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_UpdateContainerAgent() {
	svc := ecs.New(session.New())

	params := &ecs.UpdateContainerAgentInput{
		ContainerInstance: aws.String("String"), // Required
		Cluster:           aws.String("String"),
	}
	resp, err := svc.UpdateContainerAgent(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func ExampleECS_UpdateService() {
	svc := ecs.New(session.New())

	params := &ecs.UpdateServiceInput{
		Service: aws.String("String"), // Required
		Cluster: aws.String("String"),
		DeploymentConfiguration: &ecs.DeploymentConfiguration{
			MaximumPercent:        aws.Int64(1),
			MinimumHealthyPercent: aws.Int64(1),
		},
		DesiredCount:   aws.Int64(1),
		TaskDefinition: aws.String("String"),
	}
	resp, err := svc.UpdateService(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}
