package arukas

import (
	"log"
)

func removeContainer(containerID string) {
	client := NewClientWithOsExitOnErr()
	var container Container

	if err := client.Get(&container, "/containers/"+containerID); err != nil {
		client.Println(nil, "Failed to rm the container")
		log.Println(err)
		ExitCode = 1
		return
	}

	if err := client.Delete("/apps/" + container.App.ID); err != nil {
		client.Println(nil, "Failed to rm the container")
		log.Println(err)
		ExitCode = 1
		return
	}
}
