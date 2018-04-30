package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"
)

// PathObjectCacheKey is like PathCacheKey but includes an additional name
// to be included in the key, for module-namespaced objects.
//
// The result of this function is guaranteed unique for any distinct pair
// of path and name, but is not guaranteed to be in any particular format
// and in particular should never be shown to end-users.
func PathObjectCacheKey(path addrs.ModuleInstance, objectName string) string {
	return fmt.Sprintf("%s|%s", path.String(), objectName)
}
