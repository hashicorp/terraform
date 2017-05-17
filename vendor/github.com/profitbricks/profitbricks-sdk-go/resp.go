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