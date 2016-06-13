package cassandra

import (
	"testing"
)

func TestValidateReplicationClass(t *testing.T) {
	validNames := []string{
		ReplicationStrategySimple,
		ReplicationStrategyNetworkTopology,
	}
	for _, v := range validNames {
		_, errors := validateReplicationClass(v, "replication_class")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid replication_class: %q", v, errors)
		}
	}

	invalidNames := []string{
		"anything else",
	}

	for _, v := range invalidNames {
		_, errors := validateReplicationClass(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid replication_class", v)
		}
	}
}
