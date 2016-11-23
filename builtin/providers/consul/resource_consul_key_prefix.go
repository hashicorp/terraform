package consul

import (
	"fmt"
	"sort"
	"strings"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceConsulKeyPrefix() *schema.Resource {
	return &schema.Resource{
		Create: resourceConsulKeyPrefixCreate,
		Update: resourceConsulKeyPrefixUpdate,
		Read:   resourceConsulKeyPrefixRead,
		Delete: resourceConsulKeyPrefixDelete,

		Schema: map[string]*schema.Schema{
			"datacenter": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"token": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"path_prefix": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"subkeys": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"file": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				StateFunc: func(v interface{}) string {
					switch v.(type) {
					case string:
						return serializeKvMap(fileToKvMap(v.(string)))
					default:
						return ""
					}
				},
			},
		},
	}
}

func resourceConsulKeyPrefixCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	kv := client.KV()
	token := d.Get("token").(string)
	dc, err := getDC(d, client)
	if err != nil {
		return err
	}

	keyClient := newKeyClient(kv, dc, token)

	pathPrefix := d.Get("path_prefix").(string)
	subKeys := map[string]string{}

	for k, vI := range d.Get("subkeys").(map[string]interface{}) {
		subKeys[k] = vI.(string)
	}

	file := d.Get("file").(string)
	kvMap := fileToKvMap(file)
	for k, v := range kvMap {
		subKeys[k] = v
	}

	// To reduce the impact of mistakes, we will only "create" a prefix that
	// is currently empty. This way we are less likely to accidentally
	// conflict with other mechanisms managing the same prefix.
	currentSubKeys, err := keyClient.GetUnderPrefix(pathPrefix)
	if err != nil {
		return err
	}
	if len(currentSubKeys) > 0 {
		return fmt.Errorf(
			"%d keys already exist under %s; delete them before managing this prefix with Terraform",
			len(currentSubKeys), pathPrefix,
		)
	}

	// Ideally we'd use d.Partial(true) here so we can correctly record
	// a partial write, but that mechanism doesn't work for individual map
	// members, so we record that the resource was created before we
	// do anything and that way we can recover from errors by doing an
	// Update on subsequent runs, rather than re-attempting Create with
	// some keys possibly already present.
	d.SetId(pathPrefix)

	// Store the datacenter on this resource, which can be helpful for reference
	// in case it was read from the provider
	d.Set("datacenter", dc)

	// Now we can just write in all the initial values, since we can expect
	// that nothing should need deleting yet, as long as there isn't some
	// other program racing us to write values... which we'll catch on a
	// subsequent Read.
	for k, v := range subKeys {
		fullPath := pathPrefix + k
		err := keyClient.Put(fullPath, v)
		if err != nil {
			return fmt.Errorf("error while writing %s: %s", fullPath, err)
		}
	}

	return nil
}

func resourceConsulKeyPrefixUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	kv := client.KV()
	token := d.Get("token").(string)
	dc, err := getDC(d, client)
	if err != nil {
		return err
	}

	keyClient := newKeyClient(kv, dc, token)

	pathPrefix := d.Id()

	if d.HasChange("file") {
		o, n := d.GetChange("file")

		oKvMap := fileToKvMap(o.(string))
		nKvMap := fileToKvMap(n.(string))

		// We will check what keys have to be created or removed
		toPut, toRemove := diffKvFile(oKvMap, nKvMap)

		for k, v := range toPut {
			fullPath := pathPrefix + k
			if err := keyClient.Put(fullPath, v); err != nil {
				return err
			}
		}
		for _, k := range toRemove {
			fullPath := pathPrefix + k
			if err := keyClient.Delete(fullPath); err != nil {
				return err
			}
		}
	}

	if d.HasChange("subkeys") {
		o, n := d.GetChange("subkeys")
		if o == nil {
			o = map[string]interface{}{}
		}
		if n == nil {
			n = map[string]interface{}{}
		}

		om := o.(map[string]interface{})
		nm := n.(map[string]interface{})

		// First we'll write all of the stuff in the "new map" nm,
		// and then we'll delete any keys that appear in the "old map" om
		// and do not also appear in nm. This ordering means that if a subkey
		// name is changed we will briefly have both the old and new names in
		// Consul, as opposed to briefly having neither.

		// Again, we'd ideally use d.Partial(true) here but it doesn't work
		// for maps and so we'll just rely on a subsequent Read to tidy up
		// after a partial write.

		// Write new and changed keys
		for k, vI := range nm {
			v := vI.(string)
			fullPath := pathPrefix + k
			err := keyClient.Put(fullPath, v)
			if err != nil {
				return fmt.Errorf("error while writing %s: %s", fullPath, err)
			}
		}

		// Remove deleted keys
		for k, _ := range om {
			if _, exists := nm[k]; exists {
				continue
			}
			fullPath := pathPrefix + k
			err := keyClient.Delete(fullPath)
			if err != nil {
				return fmt.Errorf("error while deleting %s: %s", fullPath, err)
			}
		}

	}

	// Store the datacenter on this resource, which can be helpful for reference
	// in case it was read from the provider
	d.Set("datacenter", dc)

	return nil
}

func resourceConsulKeyPrefixRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	kv := client.KV()
	token := d.Get("token").(string)
	dc, err := getDC(d, client)
	if err != nil {
		return err
	}

	keyClient := newKeyClient(kv, dc, token)

	pathPrefix := d.Id()

	subKeys, err := keyClient.GetUnderPrefix(pathPrefix)
	if err != nil {
		return err
	}

	oFile := d.Get("file").(string)
	oKvMap := fileToKvMap(oFile)
	nKvMap := map[string]string{}
	for k, v := range subKeys {
		if _, ok := oKvMap[k]; ok == true {
			nKvMap[k] = v
			delete(subKeys, k)
		}
	}

	d.Set("subkeys", subKeys)
	d.Set("file", serializeKvMap(nKvMap))

	// Store the datacenter on this resource, which can be helpful for reference
	// in case it was read from the provider
	d.Set("datacenter", dc)

	return nil
}

func resourceConsulKeyPrefixDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	kv := client.KV()
	token := d.Get("token").(string)
	dc, err := getDC(d, client)
	if err != nil {
		return err
	}

	keyClient := newKeyClient(kv, dc, token)

	pathPrefix := d.Id()

	// Delete everything under our prefix, since the entire set of keys under
	// the given prefix is considered to be managed exclusively by Terraform.
	err = keyClient.DeleteUnderPrefix(pathPrefix)
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

// convert lines into a map of kv
func fileToKvMap(file string) map[string]string {
	variables := map[string]string{}

	lines := strings.Split(file, "\n")
	for _, line := range lines {
		kv := strings.SplitN(line, "=", 2)

		if len(kv) < 2 {
			continue
		}

		k, v := kv[0], kv[1]
		variables[k] = v
	}

	return variables
}

func diffKvFile(oldKv map[string]string, newKv map[string]string) (map[string]string, []string) {
	toPut, toRemove := map[string]string{}, []string{}

	for k, v := range newKv {
		if o, ok := oldKv[k]; ok == false || o != v {
			toPut[k] = v
		}
	}

	for k, _ := range oldKv {
		if _, ok := newKv[k]; ok == false {
			toRemove = append(toRemove, k)
		}
	}

	return toPut, toRemove
}

// Sort by alphabetical order and write a file
func serializeKvMap(kv map[string]string) string {
	var keys []string
	for k, _ := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	str := ""
	for _, k := range keys {
		str += k + "=" + kv[k] + "\n"
	}
	return str
}
