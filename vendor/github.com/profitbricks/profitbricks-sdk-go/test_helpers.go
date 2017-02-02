package profitbricks

import (
	"fmt"
	"time"
	"strings"
	"os"
)

func mkdcid(name string) string {
	request := Datacenter{
		Properties: DatacenterProperties{
			Name:        name,
			Description: "description",
			Location:    "us/las",
		},
	}
	dc := CreateDatacenter(request)
	fmt.Println("===========================")
	fmt.Println("Created a DC " + name)
	fmt.Println("Created a DC id " + dc.Id)
	fmt.Println(dc.StatusCode)
	fmt.Println("===========================")
	return dc.Id
}

func mksrvid(srv_dcid string) string {
	var req = Server{
		Properties: ServerProperties{
			Name:  "GO SDK test",
			Ram:   1024,
			Cores: 2,
		},
	}
	srv := CreateServer(srv_dcid, req)
	fmt.Println("===========================")
	fmt.Println("Created a server " + srv.Id)
	fmt.Println(srv.StatusCode)
	fmt.Println("===========================")

	waitTillProvisioned(srv.Headers.Get("Location"))
	return srv.Id
}

func mknic(lbal_dcid, serverid string) string {
	var request = Nic{
		Properties: NicProperties{
			Name: "GO SDK Original Nic",
			Lan:  1,
		},
	}

	resp := CreateNic(lbal_dcid, serverid, request)
	fmt.Println("===========================")
	fmt.Println("created a nic at server " + serverid)

	fmt.Println("created a nic with id " + resp.Id)
	fmt.Println(resp.StatusCode)
	fmt.Println("===========================")
	waitTillProvisioned(resp.Headers.Get("Location"))
	return resp.Id
}

func waitTillProvisioned(path string) {
	waitCount := 120
	fmt.Println(path)
	for i := 0; i < waitCount; i++ {
		request := GetRequestStatus(path)
		if request.Metadata.Status == "DONE" {
			break
		}
		time.Sleep(1 * time.Second)
		i++
	}
}

func getImageId(location string, imageName string, imageType string) string {
	if imageName == "" {
		return ""
	}

	SetAuth(os.Getenv("PROFITBRICKS_USERNAME"), os.Getenv("PROFITBRICKS_PASSWORD"))

	images := ListImages()
	if images.StatusCode > 299 {
		fmt.Printf("Error while fetching the list of images %s", images.Response)
	}

	if len(images.Items) > 0 {
		for _, i := range images.Items {
			imgName := ""
			if i.Properties.Name != "" {
				imgName = i.Properties.Name
			}

			if imageType == "SSD" {
				imageType = "HDD"
			}
			if imgName != "" && strings.Contains(strings.ToLower(imgName), strings.ToLower(imageName)) && i.Properties.ImageType == imageType && i.Properties.Location == location && i.Properties.Public == true {
				return i.Id
			}
		}
	}
	return ""
}
