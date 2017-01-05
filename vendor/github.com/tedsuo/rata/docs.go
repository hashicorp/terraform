/*
Package rata provides three things: Routes, a Router, and a RequestGenerator.

Routes are structs that define which Method and Path each associated http handler
should respond to. Unlike many router implementations, the routes and the handlers
are defined separately.  This allows for the routes to be reused in multiple contexts.
For example, a proxy server and a backend server can be created by having one set of
Routes, but two sets of Handlers (one handler that proxies, another that serves the
request). Likewise, your client code can use the routes with the RequestGenerator to
create requests that use the same routes.  Then, if the routes change, unit tests in
the client and proxy service will warn you of the problem.  This contract helps components
stay in sync while relying less on integration tests.

For example, let's imagine that you want to implement a "pet" resource that allows
you to view, create, update, and delete which pets people own.  Also, you would
like to include the owner_id and pet_id as part of the URL path.

First off, the routes might look like this:
  petRoutes := rata.Routes{
    {Name: "get_pet",    Method: "GET",    Path: "/people/:owner_id/pets/:pet_id"},
    {Name: "create_pet", Method: "POST",   Path: "/people/:owner_id/pets"},
    {Name: "update_pet", Method: "PUT",    Path: "/people/:owner_id/pets/:pet_id"},
    {Name: "delete_pet", Method: "DELETE", Path: "/people/:owner_id/pets/:pet_id"},
  }


On the server, create a matching set of http handlers, one for each route:
  handlers := rata.Handlers{
    "get_pet":    newGetPetHandler(),
    "create_pet": newCreatePetHandler(),
    "update_pet": newUpdatePetHandler(),
    "delete_pet": newDeletePetHandler()
  }

You can create a router by mixing the routes and handlers together:
  router, err := rata.NewRouter(petRoutes, handlers)
  if err != nil {
    panic(err)
  }

The router is just an http.Handler, so it can be used to create a server in the usual fashion:
  server := httptest.NewServer(router)

The handlers can obtain parameters derived from the URL path:

  ownerId := rata.Param(request, "owner_id")

Meanwhile, on the client side, you can create a request generator:
  requestGenerator := rata.NewRequestGenerator(server.URL, petRoutes)

You can use the request generator to ensure you are creating a valid request:
  req, err := requestGenerator.CreateRequest("get_pet", rata.Params{"owner_id": "123", "pet_id": "5"}, nil)

The generated request can be used like any other http.Request object:
  res, err := http.DefaultClient.Do(req)
*/
package rata
