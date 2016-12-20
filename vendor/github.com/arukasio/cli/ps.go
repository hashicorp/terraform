package arukas

import (
	"log"
)

func listContainers(listAll bool, quiet bool) {
	var parsedContainer []Container
	var filteredContainer []Container
	client := NewClientWithOsExitOnErr()

	if err := client.Get(&parsedContainer, "/containers"); err != nil {
		log.Println(err)
		ExitCode = 1
		return
	}

	if listAll {
		filteredContainer = parsedContainer
	} else {
		for _, container := range parsedContainer {
			if container.StatusText == "running" {
				filteredContainer = append(filteredContainer, container)
			}
		}
	}

	if quiet {
		for _, container := range filteredContainer {
			client.Println(nil, container.ID)
		}
	} else {
		client.PrintHeaderln(nil, "CONTAINER ID", "IMAGE", "COMMAND", "CREATED", "STATUS", "NAME", "ENDPOINT")
		for _, container := range filteredContainer {
			client.Println(nil, container.ID, container.ImageName, container.Cmd, container.CreatedAt.String(),
				container.StatusText, container.Name, container.Endpoint)
		}
	}
}
