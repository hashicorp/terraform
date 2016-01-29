package chef

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestDataBagsService_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"bag1":"http://localhost/data/bag1", "bag2":"http://localhost/data/bag2"}`)
	})

	databags, err := client.DataBags.List()
	if err != nil {
		t.Errorf("DataBags.List returned error: %v", err)
	}

	want := &DataBagListResult{"bag1": "http://localhost/data/bag1", "bag2": "http://localhost/data/bag2"}
	if !reflect.DeepEqual(databags, want) {
		t.Errorf("DataBags.List returned %+v, want %+v", databags, want)
	}
}

func TestDataBagsService_Create(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"uri": "http://localhost/data/newdatabag"}`)
	})

	databag := &DataBag{Name: "newdatabag"}
	response, err := client.DataBags.Create(databag)
	if err != nil {
		t.Errorf("DataBags.Create returned error: %v", err)
	}

	want := &DataBagCreateResult{URI: "http://localhost/data/newdatabag"}
	if !reflect.DeepEqual(response, want) {
		t.Errorf("DataBags.Create returned %+v, want %+v", response, want)
	}
}

func TestDataBagsService_Delete(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/data/databag", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"name": "databag", "json_class": "Chef::DataBag", "chef_type": "data_bag"}`)
	})

	response, err := client.DataBags.Delete("databag")
	if err != nil {
		t.Errorf("DataBags.Delete returned error: %v", err)
	}

	want := &DataBag{
		Name:      "databag",
		JsonClass: "Chef::DataBag",
		ChefType:  "data_bag",
	}

	if !reflect.DeepEqual(response, want) {
		t.Errorf("DataBags.Delete returned %+v, want %+v", response, want)
	}
}

func TestDataBagsService_ListItems(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/data/bag1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"item1":"http://localhost/data/bag1/item1", "item2":"http://localhost/data/bag1/item2"}`)
	})

	databags, err := client.DataBags.ListItems("bag1")
	if err != nil {
		t.Errorf("DataBags.ListItems returned error: %v", err)
	}

	want := &DataBagListResult{"item1": "http://localhost/data/bag1/item1", "item2": "http://localhost/data/bag1/item2"}
	if !reflect.DeepEqual(databags, want) {
		t.Errorf("DataBags.ListItems returned %+v, want %+v", databags, want)
	}
}

func TestDataBagsService_GetItem(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/data/bag1/item1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"id":"item1", "stuff":"things"}`)
	})

	_, err := client.DataBags.GetItem("bag1", "item1")
	if err != nil {
		t.Errorf("DataBags.GetItem returned error: %v", err)
	}
}

func TestDataBagsService_CreateItem(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/data/bag1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, ``)
	})

	dbi := map[string]string{
		"id":  "item1",
		"foo": "test123",
	}

	err := client.DataBags.CreateItem("bag1", dbi)
	if err != nil {
		t.Errorf("DataBags.CreateItem returned error: %v", err)
	}
}

func TestDataBagsService_DeleteItem(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/data/bag1/item1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, ``)
	})

	err := client.DataBags.DeleteItem("bag1", "item1")
	if err != nil {
		t.Errorf("DataBags.DeleteItem returned error: %v", err)
	}
}

func TestDataBagsService_UpdateItem(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/data/bag1/item1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, ``)
	})

	dbi := map[string]string{
		"id":  "item1",
		"foo": "test123",
	}

	err := client.DataBags.UpdateItem("bag1", "item1", dbi)
	if err != nil {
		t.Errorf("DataBags.UpdateItem returned error: %v", err)
	}
}

func TestDataBagsService_DataBagListResultString(t *testing.T) {
	e := &DataBagListResult{"bag1": "http://localhost/data/bag1", "bag2": "http://localhost/data/bag2"}
	want := "bag1 => http://localhost/data/bag1\nbag2 => http://localhost/data/bag2\n"
	if e.String() != want {
		t.Errorf("DataBagListResult.String returned:\n%+v\nwant:\n%+v\n", e.String(), want)
	}
}
