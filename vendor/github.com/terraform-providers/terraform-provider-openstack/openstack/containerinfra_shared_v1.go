package openstack

import (
	"fmt"
	"os"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/containerinfra/v1/clusters"
	"github.com/gophercloud/gophercloud/openstack/containerinfra/v1/clustertemplates"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func expandContainerInfraV1LabelsMap(v map[string]interface{}) (map[string]string, error) {
	m := make(map[string]string)
	for key, val := range v {
		labelValue, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("label %s value should be string", key)
		}
		m[key] = labelValue
	}
	return m, nil
}

func expandContainerInfraV1LabelsString(v map[string]interface{}) (string, error) {
	var formattedLabels string
	for key, val := range v {
		labelValue, ok := val.(string)
		if !ok {
			return "", fmt.Errorf("label %s value should be string", key)
		}
		formattedLabels = strings.Join([]string{
			formattedLabels,
			fmt.Sprintf("%s=%s", key, labelValue),
		}, ",")
	}
	formattedLabels = strings.Trim(formattedLabels, ",")

	return formattedLabels, nil
}

func containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts []clustertemplates.UpdateOptsBuilder, attribute, value string) []clustertemplates.UpdateOptsBuilder {
	if value == "" {
		updateOpts = append(updateOpts, clustertemplates.UpdateOpts{
			Op:   clustertemplates.RemoveOp,
			Path: strings.Join([]string{"/", attribute}, ""),
		})
	} else {
		updateOpts = append(updateOpts, clustertemplates.UpdateOpts{
			Op:    clustertemplates.ReplaceOp,
			Path:  strings.Join([]string{"/", attribute}, ""),
			Value: value,
		})
	}
	return updateOpts
}

// ContainerInfraClusterV1StateRefreshFunc returns a resource.StateRefreshFunc
// that is used to watch a container infra Cluster.
func containerInfraClusterV1StateRefreshFunc(client *gophercloud.ServiceClient, clusterID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		c, err := clusters.Get(client, clusterID).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				return c, "DELETE_COMPLETE", nil
			}
			return nil, "", err
		}

		errorStatuses := []string{
			"CREATE_FAILED",
			"UPDATE_FAILED",
			"DELETE_FAILED",
			"RESUME_FAILED",
			"ROLLBACK_FAILED",
		}
		for _, errorStatus := range errorStatuses {
			if c.Status == errorStatus {
				err = fmt.Errorf("openstack_containerinfra_cluster_v1 is in an error state: %s", c.StatusReason)
				return c, c.Status, err
			}
		}

		return c, c.Status, nil
	}
}

// containerInfraClusterV1Flavor will determine the flavor for a container infra
// cluster based on either what was set in the configuration or environment
// variable.
func containerInfraClusterV1Flavor(d *schema.ResourceData) (string, error) {
	if flavor := d.Get("flavor").(string); flavor != "" {
		return flavor, nil
	}
	// Try the OS_MAGNUM_FLAVOR environment variable
	if v := os.Getenv("OS_MAGNUM_FLAVOR"); v != "" {
		return v, nil
	}

	return "", nil
}

// containerInfraClusterV1Flavor will determine the master flavor for a
// container infra cluster based on either what was set in the configuration
// or environment variable.
func containerInfraClusterV1MasterFlavor(d *schema.ResourceData) (string, error) {
	if flavor := d.Get("master_flavor").(string); flavor != "" {
		return flavor, nil
	}

	// Try the OS_MAGNUM_FLAVOR environment variable
	if v := os.Getenv("OS_MAGNUM_MASTER_FLAVOR"); v != "" {
		return v, nil
	}

	return "", nil
}
