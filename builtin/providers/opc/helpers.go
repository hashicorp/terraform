package opc

import (
	"sort"

	"github.com/hashicorp/terraform/helper/schema"
)

// Helper function to get a string list from the schema, and alpha-sort it
func getStringList(d *schema.ResourceData, key string) []string {
	if _, ok := d.GetOk(key); !ok {
		return nil
	}
	l := d.Get(key).([]interface{})
	res := make([]string, len(l))
	for i, v := range l {
		res[i] = v.(string)
	}
	sort.Strings(res)
	return res
}

// Helper function to set a string list in the schema, in an alpha-sorted order.
func setStringList(d *schema.ResourceData, key string, value []string) error {
	sort.Strings(value)
	return d.Set(key, value)
}

// Helper function to get an int list from the schema, and numerically sort it
func getIntList(d *schema.ResourceData, key string) []int {
	if _, ok := d.GetOk(key); !ok {
		return nil
	}

	l := d.Get(key).([]interface{})
	res := make([]int, len(l))
	for i, v := range l {
		res[i] = v.(int)
	}
	sort.Ints(res)
	return res
}

func setIntList(d *schema.ResourceData, key string, value []int) error {
	sort.Ints(value)
	return d.Set(key, value)
}
