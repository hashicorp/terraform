package vsphere

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

// import "github.com/davecgh/go-spew/spew"
// import "github.com/hashicorp/terraform/builtin/providers/vsphere/helpers"

const cacheIndexFile = ".vsphereobjindex.json"

func getGovmomiClient(meta interface{}) (*govmomi.Client, error) {
	client, casted := meta.(*govmomi.Client)
	if !casted {
		return nil, fmt.Errorf("%+v is not castable as govmomi.Client", meta)
	}
	return client, nil
}

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
	r.InventoryPath, err = GetObjectPathFromManagedObjects(c, r.Reference())
	if err != nil {
		log.Printf("[DEBUG] ! oops could not get path for [%s] %s", uuid, err.Error())
		return nil, err
	}
	return r, nil
}

// GetObjectPathFromManagedObjects gets a path for a given Managed Object reference
func GetObjectPathFromManagedObjects(client *govmomi.Client, objID types.ManagedObjectReference) (string, error) {
	_, inv, err := BuildManagedObjectsIndexes(client, "")
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
func BuildManagedObjectsIndexes(client *govmomi.Client, path string) (map[string]types.ManagedObjectReference, map[string][]string, error) {
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
			kids1, invs1, err1 := BuildManagedObjectsIndexes(client, p.Path)
			if err1 != nil {
				continue
			}
			MapMerge2(&ret, &kids1)
			MapMerge1(&inv, &invs1)
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

func MapMerge1(m, m2 *map[string][]string) error {
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

func MapMerge2(m, m2 *map[string]types.ManagedObjectReference) error {
	for k, v := range *m2 {
		_, ok := (*m)[k]
		if !ok {
			(*m)[k] = v
		}
	}
	return nil
}

func dirname(path string) string {
	s := strings.Split(path, "/")
	sslice := s[0 : len(s)-1]
	out := strings.Join(sslice, "/")

	return out
}

func clearVSphereInventoryCache() {
	os.Remove(cacheIndexFile)
	_bmo = bmores{}
}

func removefirstpartsofpath(path string) string {
	s := strings.Split(path, "/")
	log.Printf("[DEBUG] split: %+v", s)
	if len(s) < 3 {
		return path
	}
	log.Printf("[DEBUG] split: %+v", s)
	return strings.Join(s[3:], "/")
}
