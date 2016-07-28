package vsphere

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"encoding/json"
	"io/ioutil"
	"log"
	"net/url"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"golang.org/x/net/context"

	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/types"
)

// import "github.com/davecgh/go-spew/spew"
// import "github.com/hashicorp/terraform/builtin/providers/vsphere/helpers"

const cacheIndexFile = ".vsphereobjindex.json"

// GetGovmomiClient gets a Govmomi client from the meta passed by Terraform
func getGovmomiClient(meta interface{}) (*govmomi.Client, error) {
	client, casted := meta.(*govmomi.Client)
	if !casted {
		return nil, fmt.Errorf("%+v is not castable as govmomi.Client", meta)
	}
	return client, nil
}

// GetDVSByUUID finds a DVS object from its UUID
func getDVSByUUID(c *govmomi.Client, uuid string) (*object.DistributedVirtualSwitch, error) {

	q := types.QueryDvsByUuid{
		This: *c.ServiceContent.DvSwitchManager,
		Uuid: uuid,
	}

	l, err := methods.QueryDvsByUuid(context.TODO(), c, &q)
	if err != nil {
		log.Printf("[DEBUG] ! oops panic getting [%s] %s", uuid, err.Error())
		return nil, err
	}
	r := object.NewDistributedVirtualSwitch(c.Client, *l.Returnval)
	r.InventoryPath, err = getObjectPathFromManagedObjects(c, r.Reference())
	if err != nil {
		log.Printf("[DEBUG] ! oops could not get path for [%s] %s", uuid, err.Error())
		return nil, err
	}
	return r, nil
}

// GetObjectPathFromManagedObjects gets a path for a given Managed Object reference
func getObjectPathFromManagedObjects(client *govmomi.Client, objID types.ManagedObjectReference) (string, error) {
	_, inv, err := buildManagedObjectsIndexes(client, "")
	if err != nil {
		return "", fmt.Errorf("Could not get path for reference %+v", objID)
	}
	res, ok := inv[objID.Value]
	if !ok {
		return "", fmt.Errorf("Could not get path for reference %+v", objID)
	}
	return res[0], nil
}

type bmores struct {
	Ret map[string]types.ManagedObjectReference
	Inv map[string][]string
}

var _bmo bmores

// BuildManagedObjectsIndexes builds indexes between objects and their paths
// it is highly costly and to be called once only. It memoizes its results
func buildManagedObjectsIndexes(client *govmomi.Client, path string) (map[string]types.ManagedObjectReference, map[string][]string, error) {
	if _bmo.Ret == nil {
		if file, err := ioutil.ReadFile(cacheIndexFile); err == nil {
			json.Unmarshal(file, &_bmo)
		} else {
			log.Printf("[DEBUG] no cached data in cacheIndexFile. Building cache. Will be slow.")
		}
	}
	if path == "" && _bmo.Ret != nil {
		return _bmo.Ret, _bmo.Inv, nil
	}
	ret := map[string]types.ManagedObjectReference{}
	inv := map[string][]string{}
	f := find.NewFinder(client.Client, true)
	paths, err := f.ManagedObjectListChildren(context.TODO(), path)
	if err != nil {
		return nil, nil, err
	}
	for _, p := range paths {
		ret[p.Path] = p.Object.Reference()
		d, ok := inv[p.Object.Reference().Value]
		if !ok {
			inv[p.Object.Reference().Value] = []string{p.Path}
			// log.Printf("[DEBUG] Path: %s, Type: %s", p.Path, p.Object.Reference().Type)
			if p.Path == path || (p.Object.Reference().Type != "Datacenter" && p.Object.Reference().Type != "Folder") {
				continue
			}
			kids1, invs1, err1 := buildManagedObjectsIndexes(client, p.Path)
			if err1 != nil {
				continue
			}
			mapMerge2(&ret, &kids1)
			mapMerge1(&inv, &invs1)
		} else {
			inv[p.Object.Reference().Value] = append(d, p.Path)
		}
	}
	if path == "" {
		_bmo = bmores{
			Ret: ret,
			Inv: inv,
		}
		clearVSphereInventoryCache()
		if file, err := os.Create(cacheIndexFile); err == nil {
			b, err := json.MarshalIndent(_bmo, "", " ")
			if err != nil {
				log.Printf("[ERROR] Oops! Error in marshalling %+v", err)
				panic("Cannot mashall data - something very wrong is going on")
			}
			file.Write(b)
			file.Close()
		}
	}
	return ret, inv, nil
}

var _testGovmomiClient *govmomi.Client

// GetTestGovmomiClient builds a Govmomi client for unit tests
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

// MapMerge1 merges two map[string][]string
func mapMerge1(m, m2 *map[string][]string) error {
	for k, v := range *m2 {
		v2, ok := (*m)[k]
		if !ok {
			(*m)[k] = v
		} else {
			(*m)[k] = append(v2, v...)
		}
	}
	return nil
}

// MapMerge2 merges two map[string]types.ManagedObjectReference
func mapMerge2(m, m2 *map[string]types.ManagedObjectReference) error {
	for k, v := range *m2 {
		_, ok := (*m)[k]
		if !ok {
			(*m)[k] = v
		}
	}
	return nil
}

// Dirname extracts the directory name from a path
func dirname(path string) string {
	s := strings.Split(path, "/")
	sslice := s[0 : len(s)-1]
	out := strings.Join(sslice, "/")

	return out
}

// ClearVSphereInventoryCache clears the vSphere inventory cache
func clearVSphereInventoryCache() {
	os.Remove(cacheIndexFile)
	_bmo = bmores{}
}

// RemoveFirstPartsOfPath removes the header part of a vSphere inventory path
func removeFirstPartsOfPath(path string) string {
	s := strings.Split(path, "/")
	log.Printf("[DEBUG] split: %+v", s)
	if len(s) < 3 {
		return path
	}
	log.Printf("[DEBUG] split: %+v", s)
	return strings.Join(s[3:], "/")
}

// GetDatacenter gets datacenter object - meant for internal use
func getDatacenter(c *govmomi.Client, dc string) (*object.Datacenter, error) {
	finder := find.NewFinder(c.Client, true)
	if dc != "" {
		d, err := finder.Datacenter(context.TODO(), dc)
		return d, err
	}
	d, err := finder.DefaultDatacenter(context.TODO())
	return d, err
}

// WaitForTaskEnd waits for a vSphere task to end
func waitForTaskEnd(task *object.Task, message string) error {
	//time.Sleep(time.Second * 5)
	if err := task.Wait(context.TODO()); err != nil {
		taskmo := mo.Task{}
		task.Properties(context.TODO(), task.Reference(), []string{"info"}, &taskmo)
		return fmt.Errorf("[%T] → "+message, err, err)
	}
	return nil

}

// SortedStringMap outputs a map[string]interface{} sorted by key
func sortedStringMap(in map[string]interface{}) string {

	var keys []string
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out string
	for _, k := range keys {
		out = fmt.Sprintf("%s%s: %+v\t", out, k, in[k])
	}
	return out
}

// JoinStringer joins fmt.Stringer elements like strings.Join
func joinStringer(values []fmt.Stringer, sep string) string {
	var data = make([]string, len(values))
	for i, v := range values {
		data[i] = v.String()
	}
	return strings.Join(data, sep)
}

// testing helpers

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("VSPHERE_USER"); v == "" {
		t.Fatal("VSPHERE_USER must be set for acceptance tests")
	}

	if v := os.Getenv("VSPHERE_PASSWORD"); v == "" {
		t.Fatal("VSPHERE_PASSWORD must be set for acceptance tests")
	}

	if v := os.Getenv("VSPHERE_SERVER"); v == "" {
		t.Fatal("VSPHERE_SERVER must be set for acceptance tests")
	}
}

// dvs helpers

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

func getDCAndFolders(c *govmomi.Client, datacenter string) (*object.Datacenter, *object.DatacenterFolders, error) {
	dvso := dvs{
		datacenter: datacenter,
	}
	return dvso.getDCAndFolders(c)
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

/** // disabled (untested)

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

// */
