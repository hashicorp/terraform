package profitbricks

import "net/http"
import "fmt"
import (
	"encoding/json"
)

func MkJson(i interface{}) string {
	jason, err := json.MarshalIndent(&i, "", "    ")
	if err != nil {
		panic(err)
	}
	//	fmt.Println(string(jason))
	return string(jason)
}

// MetaData is a map for metadata returned in a Resp.Body
type StringMap map[string]string

type StringIfaceMap map[string]interface{}

type StringCollectionMap map[string]Collection

// Resp is the struct returned by all Rest request functions
type Resp struct {
	Req        *http.Request
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// PrintHeaders prints the http headers as k,v pairs
func (r *Resp) PrintHeaders() {
	for key, value := range r.Headers {
		fmt.Println(key, " : ", value[0])
	}

}

type Id_Type_Href struct {
	Id   string `json:"id"`
	Type string `json:"type"`
	Href string `json:"href"`
}

type MetaData StringIfaceMap

type Instance struct {
	Id_Type_Href
	MetaData   StringMap           `json:"metaData,omitempty"`
	Properties StringIfaceMap      `json:"properties,omitempty"`
	Entities   StringCollectionMap `json:"entities,omitempty"`
	Resp       Resp                `json:"-"`
}

type Collection struct {
	Id_Type_Href
	Items []Instance `json:"items,omitempty"`
	Resp  Resp       `json:"-"`
}
