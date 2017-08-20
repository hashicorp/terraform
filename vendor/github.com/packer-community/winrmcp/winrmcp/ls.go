package winrmcp

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/masterzen/winrm"
)

type FileItem struct {
	Name          string
	Path          string
	Mode          string
	LastWriteTime string
	Length        int
}

func fetchList(client *winrm.Client, remotePath string) ([]FileItem, error) {
	script := fmt.Sprintf("Get-ChildItem %s", remotePath)
	stdout, stderr, _, err := client.RunWithString("powershell -Command \""+script+" | ConvertTo-Xml -NoTypeInformation -As String\"", "")
	if err != nil {
		return nil, fmt.Errorf("Couldn't execute script %s: %v", script, err)
	}

	if stderr != "" {
		if os.Getenv("WINRMCP_DEBUG") != "" {
			log.Printf("STDERR returned: %s\n", stderr)
		}
	}

	if stdout != "" {
		doc := pslist{}
		err := xml.Unmarshal([]byte(stdout), &doc)
		if err != nil {
			return nil, fmt.Errorf("Couldn't parse results: %v", err)
		}

		return convertFileItems(doc.Objects), nil
	}

	return []FileItem{}, nil
}

func convertFileItems(objects []psobject) []FileItem {
	items := make([]FileItem, len(objects))

	for i, object := range objects {
		for _, property := range object.Properties {
			switch property.Name {
			case "Name":
				items[i].Name = property.Value
			case "Mode":
				items[i].Mode = property.Value
			case "FullName":
				items[i].Path = property.Value
			case "Length":
				if n, err := strconv.Atoi(property.Value); err == nil {
					items[i].Length = n
				}
			case "LastWriteTime":
				items[i].LastWriteTime = property.Value
			}
		}
	}

	return items
}
