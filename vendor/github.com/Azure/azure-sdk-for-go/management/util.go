package management

import (
	"io/ioutil"

	"github.com/Azure/azure-sdk-for-go/core/http"
)

func getResponseBody(response *http.Response) ([]byte, error) {
	defer response.Body.Close()
	return ioutil.ReadAll(response.Body)
}
