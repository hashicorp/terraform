package azure

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/management"
)

// testAccResourceDestroyedErrorFilter tests whether the given error is an azure ResourceNotFound
// error and properly annotates it if otherwise:
func testAccResourceDestroyedErrorFilter(resource string, err error) error {
	switch {
	case err == nil:
		return fmt.Errorf("Azure %s still exists.", resource)
	case err != nil && management.IsResourceNotFoundError(err):
		return nil
	default:
		return err
	}
}
