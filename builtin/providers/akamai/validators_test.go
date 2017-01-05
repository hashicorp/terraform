package akamai

import (
    "testing"
)

func TestValidateContractId(t *testing.T) {
    validIdentifiers := []string{
        "ctr_1-1TJZFW",
    }
    for _, v := range validIdentifiers {
        _, errors := validateContractId(v, "contract_id")
        if len(errors) != 0 {
            t.Fatalf("%q should be a valid contract ID: %q", v, errors)
        }
    }

    invalidIdentifiers := []string{
        "1-1TJZFW",
        "1TJZFW",
        "ctr_",
        "ctr_1-1TJZF",
    }
    for _, v := range invalidIdentifiers {
        _, errors := validateContractId(v, "contract_id")
        if len(errors) == 0 {
            t.Fatalf("%q should be an invalid contract ID", v)
        }
    }
}

func TestValidateGroupId(t *testing.T) {
    validIdentifiers := []string{
        "grp_15166",
    }
    for _, v := range validIdentifiers {
        _, errors := validateGroupId(v, "group_id")
        if len(errors) != 0 {
            t.Fatalf("%q should be a valid group ID: %q", v, errors)
        }
    }

    invalidIdentifiers := []string{
        "15166",
        "grp_",
        "grp_1516",
        "grp_ABCDEF",
    }

    for _, v := range invalidIdentifiers {
        _, errors := validateGroupId(v, "group_id")
        if len(errors) == 0 {
            t.Fatalf("%q should be an invalid group ID", v)
        }
    }
}
