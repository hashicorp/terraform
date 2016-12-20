package arukas

import (
	"log"
)

func stopContainer(stopContainerID string) {
	client := NewClientWithOsExitOnErr()

	if err := client.Delete("/containers/" + stopContainerID + "/power"); err != nil {
		client.Println(nil, "Failed to stop the container")
		log.Println(err)
		ExitCode = 1
	} else {
		client.Println(nil, "Stopping...")
	}
}
