package arukas

import (
	"log"
)

func startContainer(containerID string, quiet bool) {
	client := NewClientWithOsExitOnErr()

	if err := client.Post(nil, "/containers/"+containerID+"/power", nil); err != nil {
		client.Println(nil, "Failed to start the container")
		log.Print(err)
		ExitCode = 1
	} else {
		if !quiet {
			client.Println(nil, "Starting...")
		}
	}
}
