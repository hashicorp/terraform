---
layout: "functions"
page_title: "keys function"
sidebar_current: "docs-funcs-collection-keys"
description: |-
  The keys function returns a list of the keys in a given map.
---

# `keys` Function

`keys` takes a map and returns a list containing the keys from that map.

The keys are returned in lexicographical order, ensuring that the result will
be identical as long as the keys in the map don't change.

## Examples

```
> keys({a=1, c=2, d=3})
[
  "a",
  "c",
  "d",
]
```

## Related Functions

* [`values`](./values.html) returns a list of the _values_ from a map.
