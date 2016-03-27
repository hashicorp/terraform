package management

import (
	"github.com/Azure/azure-sdk-for-go/core/http"
	"io/ioutil"
)

func getResponseBody(response *http.Response) ([]byte, error) {
	defer response.Body.Close()
	return ioutil.ReadAll(response.Body)
}
