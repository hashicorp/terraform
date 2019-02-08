/*
Package clientconfig provides convienent functions for creating OpenStack
clients. It is based on the Python os-client-config library.

See https://docs.openstack.org/os-client-config/latest for details.

Example to Create a Provider Client From clouds.yaml

	opts := &clientconfig.ClientOpts{
		Name: "hawaii",
	}

	pClient, err := clientconfig.AuthenticatedClient(opts)
	if err != nil {
		panic(err)
	}


Example to Manually Create a Provider Client

	opts := &clientconfig.ClientOpts{
		AuthInfo: &clientconfig.AuthInfo{
			AuthURL:     "https://hi.example.com:5000/v3",
			Username:    "jdoe",
			Password:    "password",
			ProjectName: "Some Project",
			DomainName:  "default",
		},
	}

	pClient, err := clientconfig.AuthenticatedClient(opts)
	if err != nil {
		panic(err)
	}


Example to Create a Service Client from clouds.yaml

	opts := &clientconfig.ClientOpts{
		Name: "hawaii",
	}

	computeClient, err := clientconfig.NewServiceClient("compute", opts)
	if err != nil {
		panic(err)
	}

*/
package clientconfig
