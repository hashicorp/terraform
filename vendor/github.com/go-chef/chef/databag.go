package chef

import "fmt"

// DataBagService is the service for interacting with the chef server data endpoint
type DataBagService struct {
	client *Client
}

// DataBagItem is a data bag item
type DataBagItem interface{}

// DataBag is a data bag
type DataBag struct {
	Name      string `json:"name"`
	JsonClass string `json:"json_class"`
	ChefType  string `json:"chef_type"`
}

type DataBagCreateResult struct {
	URI string `json:"uri"`
}

// DataBagListResult is the list of data bags returned by chef-api when listing
// http://docs.getchef.com/api_chef_server.html#data
type DataBagListResult map[string]string

// String makes DataBagListResult implement the string result
func (d DataBagListResult) String() (out string) {
	for k, v := range d {
		out += fmt.Sprintf("%s => %s\n", k, v)
	}
	return out
}

// List returns a list of databags on the server
//   Chef API Docs: http://docs.getchef.com/api_chef_server.html#id18
func (d *DataBagService) List() (data *DataBagListResult, err error) {
	path := fmt.Sprintf("data")
	err = d.client.magicRequestDecoder("GET", path, nil, &data)
	return
}

// Create adds a data bag to the server
//   Chef API Docs: http://docs.getchef.com/api_chef_server.html#id19
func (d *DataBagService) Create(databag *DataBag) (result *DataBagCreateResult, err error) {
	body, err := JSONReader(databag)
	if err != nil {
		return
	}

	err = d.client.magicRequestDecoder("POST", "data", body, &result)
	return
}

// Delete removes a data bag from the server
//   Chef API Docs: ????????????????
func (d *DataBagService) Delete(name string) (result *DataBag, err error) {
	path := fmt.Sprintf("data/%s", name)
	err = d.client.magicRequestDecoder("DELETE", path, nil, &result)
	return
}

// ListItems gets a list of the items in a data bag.
//   Chef API Docs: http://docs.getchef.com/api_chef_server.html#id20
func (d *DataBagService) ListItems(name string) (data *DataBagListResult, err error) {
	path := fmt.Sprintf("data/%s", name)
	err = d.client.magicRequestDecoder("GET", path, nil, &data)
	return
}

// CreateItem adds an item to a data bag
//   Chef API Docs: http://docs.getchef.com/api_chef_server.html#id21
func (d *DataBagService) CreateItem(databagName string, databagItem DataBagItem) (err error) {
	body, err := JSONReader(databagItem)
	if err != nil {
		return
	}
	path := fmt.Sprintf("data/%s", databagName)
	return d.client.magicRequestDecoder("POST", path, body, nil)
}

// DeleteItem deletes an item from a data bag
//   Chef API Docs: http://docs.getchef.com/api_chef_server.html#id22
func (d *DataBagService) DeleteItem(databagName, databagItem string) (err error) {
	path := fmt.Sprintf("data/%s/%s", databagName, databagItem)
	err = d.client.magicRequestDecoder("DELETE", path, nil, nil)
	return
}

// GetItem gets an item from a data bag
//   Chef API Docs: http://docs.getchef.com/api_chef_server.html#id23
func (d *DataBagService) GetItem(databagName, databagItem string) (item DataBagItem, err error) {
	path := fmt.Sprintf("data/%s/%s", databagName, databagItem)
	err = d.client.magicRequestDecoder("GET", path, nil, &item)
	return
}

// UpdateItem updates an item in a data bag
//    Chef API Docs: http://docs.getchef.com/api_chef_server.html#id24
func (d *DataBagService) UpdateItem(databagName, databagItemId string, databagItem DataBagItem) (err error) {
	body, err := JSONReader(databagItem)
	if err != nil {
		return
	}
	path := fmt.Sprintf("data/%s/%s", databagName, databagItemId)
	return d.client.magicRequestDecoder("PUT", path, body, nil)
}
