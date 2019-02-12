Release v1.0.1
==============

**Changes**

* Added support for absolute etree Path queries. An absolute path begins with
  `/` or `//` and begins its search from the element's document root.
* Added [`GetPath`](https://godoc.org/github.com/beevik/etree#Element.GetPath)
  and [`GetRelativePath`](https://godoc.org/github.com/beevik/etree#Element.GetRelativePath)
  functions to the [`Element`](https://godoc.org/github.com/beevik/etree#Element)
  type.

**Breaking changes**

* A path starting with `//` is now interpreted as an absolute path.
  Previously, it was interpreted as a relative path starting from the element
  whose
  [`FindElement`](https://godoc.org/github.com/beevik/etree#Element.FindElement)
  method was called.  To remain compatible with this release, all paths
  prefixed with `//` should be prefixed with `.//` when called from any
  element other than the document's root.


Release v1.0.0
==============

Initial release.
