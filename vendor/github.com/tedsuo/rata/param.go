package rata

import "net/http"

//  Param returns the parameter with the given name from the given request.
func Param(req *http.Request, paramName string) string {
	return req.URL.Query().Get(":" + paramName)
}
