package arukas

func createAndRunContainer(name string, image string, instances int, mem int, envs []string, ports []string, cmd string, appName string) {
	client := NewClientWithOsExitOnErr()
	var appSet AppSet

	// create an app
	newApp := App{Name: appName}

	var parsedEnvs Envs
	var parsedPorts Ports

	if len(envs) > 0 {
		var err error
		parsedEnvs, err = ParseEnv(envs)
		if err != nil {
			client.Println(nil, err)
			ExitCode = 1
			return
		}
	}

	if len(ports) > 0 {
		var err error
		parsedPorts, err = ParsePort(ports)
		if err != nil {
			client.Println(nil, err)
			ExitCode = 1
			return
		}
	}

	newContainer := Container{
		Envs:      parsedEnvs,
		Ports:     parsedPorts,
		ImageName: image,
		Mem:       mem,
		Instances: instances,
		Cmd:       cmd,
		Name:      name,
	}

	newAppSet := AppSet{
		App:       newApp,
		Container: newContainer,
	}

	if err := client.Post(&appSet, "/app-sets", newAppSet); err != nil {
		client.Println(nil, err)
		ExitCode = 1
		return
	}

	startContainer(appSet.Container.ID, true)

	client.Println(nil, "ID", "IMAGE", "CREATED", "STATUS", "NAME", "ENDPOINT")
	client.Println(nil, appSet.Container.ID, appSet.Container.ImageName, appSet.Container.CreatedAt.String(),
		appSet.Container.StatusText, appSet.Container.Name, appSet.Container.Endpoint)
}
