package statefile

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/tfdiags"
)

func readStateV3(src []byte) (*File, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	sV3 := &stateV3{}
	err := json.Unmarshal(src, sV3)
	if err != nil {
		diags = diags.Append(jsonUnmarshalDiags(err))
		return nil, diags
	}

	file, prepDiags := prepareStateV3(sV3)
	diags = diags.Append(prepDiags)
	return file, diags
}

func prepareStateV3(sV3 *stateV3) (*File, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	sV4, err := upgradeStateV3ToV4(sV3)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			upgradeFailed,
			fmt.Sprintf("Error upgrading state file format from version 3 to version 4: %s.", err),
		))
		return nil, diags
	}

	file, prepDiags := prepareStateV4(sV4)
	diags = diags.Append(prepDiags)
	return file, diags
}

// stateV2 is a representation of the legacy JSON state format version 3.
//
// It is only used to read version 3 JSON files prior to upgrading them to
// the current format.
//
// The differences between version 2 and version 3 are only in the data and
// not in the structure, so stateV3 actually shares the same structs as
// stateV2. Type stateV3 represents that the data within is formatted as
// expected by the V3 format, rather than the V2 format.
type stateV3 stateV2
