package routing

import "github.com/tedsuo/rata"

var CCRoutes = rata.Routes{
	{Name: "apps", Method: "GET", Path: "/v3/apps"},
}

var UAARoutes = rata.Routes{
	{Name: "refresh_token", Method: "POST", Path: "/oauth/token"},
}
