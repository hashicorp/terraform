v0.1.1 (2017-03-31)
===

* [Added] Rate-limited api requests
* [Fix] Locale model
* [Added] Content type field unmarshaling

v0.1.0 (2017-03-26)
===

### Introducing resource services
Every entity resource now has its own service definition for handling api communication. With this release, we don't store `contentful client` and `space` objects inside entities anymore. Resource services now get `spaceID` as a string parameter when it is neccessary.

With the old versions, in order to create a new `ContentType`, for example, you first need to observe `Space` object. That is no longer required. The problem with the old method was that you had to make an extra api request to observe the `Space` in order to interact with rest of the resources. The following example demonstras the difference between old and new version.

```go
// prior to v0.1.0
space, err := cma.GetSpace("space-id") // this call was making an extra api call
contentTypes, err := space.ContentTypes()

// after v0.1.0
contentType := &contentful.ContentType{ ... }
spaceID := "space-id"
cma.ContentTypes.Upsert(spaceID, contentType) // now we are passing spaceID as string
```

You can access available resources as follows:

```go
cma := contentful.NewCMA("token")
cma.Spaces
cma.APIKeys
cma.Assets
cma.ContentTypes
cma.Entries
cma.Locales
cma.Webhooks
```

Every resource service exposes the following interface:

`List(spaceID string) *Collection`

`Get(spaceID, contentTypeID string)(*ContentType, error)`

`Upsert(spaceID string, ct *ContentType) error`

`Delete(spaceID string, ct *ContentType) error`

### Create resource instancas directly from their model definitions

All `New{ResourceName}`, such as `NewContentType`, `NewSpace`, functions are removed from the SDK. It turned out that it wasn't a good practice in golang. Instead, you can directly initiate resource entities directly from their models, such as:

```go
contentType := &contentful.ContentType{
    Name: "name",
    ... other fields
}
```


v0.0.3 (2017-03-22)
===
* [Added] PredefinedValues validation
* [Added] Range validation greater/less than equal to support.
* [Added] Size validation for content type field.
* [Added] Packages are vendored with `godep`.
* [Added] `version.go`.
* [Added] `entity/content_type`: regex validation for content type field.
* [Added] Validation data structures added: `MinMax`, `Regex`
* [Added] `LinkType` support for `Field` struct
* [Added] New validations: `MimeType`, `Dimension`, `FileSize`


v0.0.2 (2017-03-21)
===
* `entity/webhook`: add tests for webhook entity.
* `entity/space`: add tests for space entity.
* `errors`: add tests for error handler.
* `entity/content_type`: add test for content type entity.
* `entity/content_type`: Field validations added for link type
* `entity/content_type`: field validations added: Range, PredefinedValues, Unique


v0.0.1 (2017-03-20)
===
* `sdk`: first implementation.
* `collection`: first implementation.
* `entity/content_type`: first implementation.
* `entity/entry`: first implementation.
* `entity/query`: first implementation.
* `entity/asset`: first implementation.
* `entity/locale`: first implementation.
* `entity/space`: first implementation.
* `entity/webhook`: first implementation.
* `entity/api_key`: first implementation.
* `sdk`: basic documentation.
* `examples`: some examples for entities
