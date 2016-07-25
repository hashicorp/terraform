package dvs

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"golang.org/x/net/context"
)

var _testGovmomiClient *govmomi.Client

func getGovmomiClient(meta interface{}) (*govmomi.Client, error) {
	client, casted := meta.(*govmomi.Client)
	if !casted {
		return nil, fmt.Errorf("%+v is not castable as govmomi.Client", meta)
	}
	return client, nil
}

// parse ID to components (DVS)
func parseDVSID(id string) (out *dvsID, err error) {
	out = &dvsID{}
	// _, err = fmt.Sscanf(id, dvs_name_format, &out.datacenter, &out.name)
	r := re_dvs.FindStringSubmatch(id)
	if r == nil {
		return nil, fmt.Errorf("Cannot match id [%s] with regexp [%s]", id, re_dvs)
	}
	out.datacenter = r[1]
	out.path = r[2]
	return
}

// parse ID to components (DVPG)
func parseDVPGID(id string) (out *dvPGID, err error) {
	out = &dvPGID{}
	r := re_dvpg.FindStringSubmatch(id)
	if r == nil {
		return nil, fmt.Errorf("Cannot match id [%s] with regexp [%s]", id, re_dvs)
	}
	out.datacenter = r[1]
	out.switchName = r[2]
	out.name = r[3]
	return
}

// parse ID to components (MapHostDVS)
func parseMapHostDVSID(id string) (out *mapHostDVSID, err error) {
	out = &mapHostDVSID{}
	r := re_maphostdvs.FindStringSubmatch(id)
	if r == nil {
		return nil, fmt.Errorf("Cannot match id [%s] with regexp [%s]", id, re_dvs)
	}
	out.datacenter = r[1]
	out.switchName = r[2]
	out.hostName = r[3]
	return
}

// parse ID to components (MapHostDVS)
func parseMapVMDVPGID(id string) (out *mapVMDVPGID, err error) {
	out = &mapVMDVPGID{}
	r := re_mapvmdvpg.FindStringSubmatch(id)
	if r == nil {
		return nil, fmt.Errorf("Cannot match id [%s] with regexp [%s]", id, re_dvs)
	}
	out.datacenter = r[1]
	out.switchName = r[2]
	out.portgroupName = r[3]
	out.vmName = r[4]
	//_, err = fmt.Sscanf(id, mapvmdvpg_name_format, &out.datacenter, &out.switchName, &out.portgroupName, &out.vmName)
	return
}

func getDCAndFolders(c *govmomi.Client, datacenter string) (*object.Datacenter, *object.DatacenterFolders, error) {
	dvso := dvs{
		datacenter: datacenter,
	}
	return dvso.getDCAndFolders(c)
}

func getTestGovmomiClient() (*govmomi.Client, error) {
	if _testGovmomiClient == nil {
		u, err := url.Parse("https://" + os.Getenv("VSPHERE_URL") + "/sdk")
		if err != nil {
			return nil, fmt.Errorf("Cannot parse VSPHERE_URL")
		}
		u.User = url.UserPassword(os.Getenv("VSPHERE_USER"), os.Getenv("VSPHERE_PASSWORD"))

		_testGovmomiClient, err = govmomi.NewClient(context.TODO(), u, true)
		if err != nil {
			return nil, err
		}
	}
	return _testGovmomiClient, nil
}

func changeFolder(c *govmomi.Client, datacenter, objtype, folderPath string) (*object.Folder, error) {
	var folderObj *object.Folder
	var folderRef object.Reference
	var err error
	if len(folderPath) > 0 {
		si := object.NewSearchIndex(c.Client)
		folderRef, err = si.FindByInventoryPath(
			context.TODO(), fmt.Sprintf("%v/%v/%v", datacenter, objtype, folderPath))
		if err != nil {
			err = fmt.Errorf("Error reading folder %s: %s", folderPath, err)
		} else if folderRef == nil {
			err = fmt.Errorf("Cannot find folder %s", folderPath)
		} else {
			folderObj = folderRef.(*object.Folder)
		}
	}
	return folderObj, err
}

func dirname(path string) string {
	s := strings.Split(path, "/")
	sslice := s[0 : len(s)-1]
	out := strings.Join(sslice, "/")

	return out
}

func dirAndFile(path string) (string, string) {
	s := strings.Split(path, "/")
	if len(s) == 1 {
		return "", path
	}
	sslice := s[0 : len(s)-1]
	folderPath := strings.Join(sslice, "/")
	filePath := s[len(s)-1]
	return folderPath, filePath

}

func vmPath(folder string, name string) string {
	var path string
	if len(folder) > 0 {
		path += folder + "/"
	}
	return path + name
}
