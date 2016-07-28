package helpers

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"golang.org/x/net/context"
)

// GetDatacenter gets datacenter object - meant for internal use
func GetDatacenter(c *govmomi.Client, dc string) (*object.Datacenter, error) {
	finder := find.NewFinder(c.Client, true)
	if dc != "" {
		d, err := finder.Datacenter(context.TODO(), dc)
		return d, err
	}
	d, err := finder.DefaultDatacenter(context.TODO())
	return d, err
}

// WaitForTaskEnd waits for a vSphere task to end
func WaitForTaskEnd(task *object.Task, message string) error {
	//time.Sleep(time.Second * 5)
	if err := task.Wait(context.TODO()); err != nil {
		taskmo := mo.Task{}
		task.Properties(context.TODO(), task.Reference(), []string{"info"}, &taskmo)
		return fmt.Errorf("[%T] → "+message, err, err)
	}
	return nil

}

// SortedStringMap outputs a map[string]interface{} sorted by key
func SortedStringMap(in map[string]interface{}) string {

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
func JoinStringer(values []fmt.Stringer, sep string) string {
	var data = make([]string, len(values))
	for i, v := range values {
		data[i] = v.String()
	}
	return strings.Join(data, sep)
}
