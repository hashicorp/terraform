/*
(c) Copyright [2015] Hewlett Packard Enterprise Development LP

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package ov -
package ov

import (
	//"github.com/HewlettPackard/oneview-golang/utils"
	"fmt"
	"strings"
)

func (c *OVClient) ManageI3SConnections(connections []Connection) ([]Connection, error) {

	//Find the deploy net called deploy.net
	deployNet, err := c.GetEthernetNetworkByName("deploy.net")
	if err != nil || deployNet.URI.IsNil() {
		return connections, fmt.Errorf("Could not find deployment ethernet network name: deploy.net")
	}

	//This section finds which PortIds we have available so we can apply new port ids to connections that have the boot PortIds(Which the boot connections need).
	availablePortIds := []string{"Mezz 3:1-b", "Mezz 3:1-c", "Mezz 3:1-d",
		"Mezz 3:2-b", "Mezz 3:2-c", "Mezz 3:2-d"}
	for i := 0; i < len(connections); i++ {
		for j := 0; j < len(availablePortIds); j++ {
			if connections[i].PortID == availablePortIds[j] {
				availablePortIds = append(availablePortIds[:j], availablePortIds[j+1:]...)
			}
		}
	}

	//This method finds the deployment connections if the server has them
	deployConnections := make([]Connection, 2)
	for i := 0; i < len(connections); i++ {
		// If we find the deployment connections, make them bootable
		// If a connection has our boot ports then we need to give it a new port id
		if connections[i].Name == "Deployment Network A" {
			deployConnections[0] = connections[i]
			connections[i].Boot.Priority = "Primary"
		} else if connections[i].Name == "Deployment Network B" {
			deployConnections[1] = connections[i]
			connections[i].Boot.Priority = "Secondary"
		} else if connections[i].PortID == "Mezz 3:1-a" {
			for j := 0; j < len(availablePortIds); j++ {
				if strings.Contains(availablePortIds[j], "Mezz 3:1") {
					connections[i].PortID = availablePortIds[j]
					availablePortIds = append(availablePortIds[:j], availablePortIds[j+1:]...)
					break
				}
			}
			if connections[i].PortID == "Mezz 3:1-a" {
				return connections, fmt.Errorf("Could not move connection to new portID: %s", connections[i].Name)
			}
		} else if connections[i].PortID == "Mezz 3:2-a" {
			for j := 0; j < len(availablePortIds); j++ {
				if strings.Contains(availablePortIds[j], "Mezz 3:2") {
					connections[i].PortID = availablePortIds[j]
					availablePortIds = append(availablePortIds[:j], availablePortIds[j+1:]...)
					break
				}
			}
			if connections[i].PortID == "Mezz 3:2-a" {
				return connections, fmt.Errorf("Could not move connection to new portID: %s", connections[i].Name)
			}
		}
	}

	// If we didn't find the deployment connections then we need to create them
	if deployConnections[0].NetworkURI.IsNil() {

		boot1 := BootOption{
			Priority: "Primary",
		}
		connection1 := Connection{
			ID:            connections[len(connections)-1].ID + 1,
			Name:          "Deployment Network A",
			FunctionType:  "Ethernet",
			RequestedMbps: "2500",
			NetworkURI:    deployNet.URI,
			Boot:          boot1,
			PortID:        "Mezz 3:1-a",
		}
		connections = append(connections, connection1)
	}
	if deployConnections[1].NetworkURI.IsNil() {
		boot2 := BootOption{
			Priority: "Secondary",
		}
		connection2 := Connection{
			ID:            connections[len(connections)-1].ID + 1,
			Name:          "Deployment Network B",
			FunctionType:  "Ethernet",
			RequestedMbps: "2500",
			NetworkURI:    deployNet.URI,
			Boot:          boot2,
			PortID:        "Mezz 3:2-a",
		}
		connections = append(connections, connection2)
	}

	return connections, nil

}
