package azure

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

type skuType int

const (
	DTU   skuType = 0
	VCore skuType = 1
)

type sku struct {
	Name, Tier, Family                                string
	Capacity                                          int
	MaxAllowedGB, MaxSizeGb, MinCapacity, MaxCapacity float64
	SkuType                                           skuType
}

// getDTUMaxGB: this map holds all of the DTU to 'max_size_gb' mappings based on a DTU lookup
//              note that the value can be below the returned value, except for 'basic' it's
//              value must match exactly what is returned else it will be rejected by the API
//              which will return a 'Internal Server Error'

var getDTUMaxGB = map[string]map[int]float64{
	"basic": {
		50:   4.8828125,
		100:  9.765625,
		200:  19.53125,
		300:  29.296875,
		400:  39.0625,
		800:  78.125,
		1200: 117.1875,
		1600: 156.25,
	},
	"standard": {
		50:   500,
		100:  750,
		200:  1024,
		300:  1280,
		400:  1536,
		800:  2048,
		1200: 2560,
		1600: 3072,
		2000: 3584,
		2500: 4096,
		3000: 4096,
	},
	"premium": {
		125:  1024,
		250:  1024,
		500:  1024,
		1000: 1024,
		1500: 1536,
		2000: 2048,
		2500: 2560,
		3000: 3072,
		3500: 3584,
		4000: 4096,
	},
}

// supportedDTUMaxGBValues: this map holds all of the valid 'max_size_gb' values
//                          for a DTU SKU type. If the 'max_size_gb' is anything
//                          other than the values in the map the API with throw
//                          an 'Internal Server Error'

var supportedDTUMaxGBValues = map[int]float64{
	50:   1,
	100:  1,
	150:  1,
	200:  1,
	250:  1,
	300:  1,
	400:  1,
	500:  1,
	750:  1,
	800:  1,
	1024: 1,
	1200: 1,
	1280: 1,
	1536: 1,
	1600: 1,
	1792: 1,
	2000: 1,
	2048: 1,
	2304: 1,
	2500: 1,
	2560: 1,
	2816: 1,
	3000: 1,
	3072: 1,
	3328: 1,
	3584: 1,
	3840: 1,
	4096: 1,
}

// getvCoreMaxGB: this map holds all of the vCore to 'max_size_gb' mappings based on a vCore lookup
//                note that the value can be below the returned value

var getvCoreMaxGB = map[string]map[string]map[int]float64{
	"generalpurpose": {
		"gen4": {
			1:  512,
			2:  756,
			3:  1536,
			4:  1536,
			5:  1536,
			6:  2048,
			7:  2048,
			8:  2048,
			9:  2048,
			10: 2048,
			16: 3584,
			24: 4096,
		},
		"gen5": {
			2:  512,
			4:  756,
			6:  1536,
			8:  1536,
			10: 1536,
			12: 2048,
			14: 2048,
			16: 2048,
			18: 3072,
			20: 3072,
			24: 3072,
			32: 4096,
			40: 4096,
			80: 4096,
		},
	},
	"businesscritical": {
		"gen4": {
			2:  1024,
			3:  1024,
			4:  1024,
			5:  1024,
			6:  1024,
			7:  1024,
			8:  1024,
			9:  1024,
			10: 1024,
			16: 1024,
			24: 1024,
		},
		"gen5": {
			4:  1024,
			6:  1536,
			8:  1536,
			10: 1536,
			12: 3072,
			14: 3072,
			16: 3072,
			18: 3072,
			20: 3072,
			24: 4096,
			32: 4096,
			40: 4096,
			80: 4096,
		},
	},
}

// getTierFromName: this map contains all of the valid mappings between 'name' and 'tier'
//                  the reason for this map is that the user may pass in an invalid mapping
//                  (e.g. name: "Basicpool" tier:"BusinessCritical") this map allows me
//                  to lookup the correct values in other maps even if the config file
//                  contains an invalid 'tier' attribute.

var getTierFromName = map[string]string{
	"basicpool":    "Basic",
	"standardpool": "Standard",
	"premiumpool":  "Premium",
	"gp_gen4":      "GeneralPurpose",
	"gp_gen5":      "GeneralPurpose",
	"bc_gen4":      "BusinessCritical",
	"bc_gen5":      "BusinessCritical",
}

func MSSQLElasticPoolValidateSKU(diff *schema.ResourceDiff) error {

	name := diff.Get("sku.0.name")
	tier := diff.Get("sku.0.tier")
	capacity := diff.Get("sku.0.capacity")
	family := diff.Get("sku.0.family")
	maxSizeBytes := diff.Get("max_size_bytes")
	maxSizeGb := diff.Get("max_size_gb")
	minCapacity := diff.Get("per_database_settings.0.min_capacity")
	maxCapacity := diff.Get("per_database_settings.0.max_capacity")

	s := sku{
		Name:        name.(string),
		Tier:        tier.(string),
		Family:      family.(string),
		Capacity:    capacity.(int),
		MaxSizeGb:   maxSizeGb.(float64),
		MinCapacity: minCapacity.(float64),
		MaxCapacity: maxCapacity.(float64),
		SkuType:     DTU,
	}

	// Convert Bytes to Gigabytes only if
	// 'max_size_bytes' has changed
	if diff.HasChange("max_size_bytes") {
		s.MaxSizeGb = float64(maxSizeBytes.(int) / 1024 / 1024 / 1024)
	}

	// Check to see if the name describes a vCore type SKU
	if strings.HasPrefix(strings.ToLower(s.Name), "gp_") || strings.HasPrefix(strings.ToLower(s.Name), "bc_") {
		s.SkuType = VCore
	}

	// Universal check for both DTU and vCore based SKUs
	if !nameTierIsValid(s) {
		return fmt.Errorf("Mismatch between SKU name '%s' and tier '%s', expected 'tier' to be '%s'", s.Name, s.Tier, getTierFromName[strings.ToLower(s.Name)])
	}

	// Verify that Family is valid
	if s.SkuType == DTU && s.Family != "" {
		return fmt.Errorf("Invalid attribute 'family'(%s) for service tier '%s', remove the 'family' attribute from the configuration file", s.Family, s.Tier)
	} else if s.SkuType == VCore && !nameContainsFamily(s) {
		return fmt.Errorf("Mismatch between SKU name '%s' and family '%s', expected '%s'", s.Name, s.Family, getFamilyFromName(s))
	}

	//get max GB and do validation based on SKU type
	if s.SkuType == DTU {
		s.MaxAllowedGB = getDTUMaxGB[strings.ToLower(s.Tier)][s.Capacity]
		return doDTUSKUValidation(s)
	} else {
		s.MaxAllowedGB = getvCoreMaxGB[strings.ToLower(s.Tier)][strings.ToLower(s.Family)][s.Capacity]
		return doVCoreSKUValidation(s)
	}
}

func nameContainsFamily(s sku) bool {
	if s.Family == "" {
		return false
	}

	return strings.Contains(strings.ToLower(s.Name), strings.ToLower(s.Family))
}

func nameTierIsValid(s sku) bool {
	if strings.EqualFold(s.Name, "BasicPool") && !strings.EqualFold(s.Tier, "Basic") ||
		strings.EqualFold(s.Name, "StandardPool") && !strings.EqualFold(s.Tier, "Standard") ||
		strings.EqualFold(s.Name, "PremiumPool") && !strings.EqualFold(s.Tier, "Premium") ||
		strings.HasPrefix(strings.ToLower(s.Name), "gp_") && !strings.EqualFold(s.Tier, "GeneralPurpose") ||
		strings.HasPrefix(strings.ToLower(s.Name), "bc_") && !strings.EqualFold(s.Tier, "BusinessCritical") {
		return false
	}

	return true
}

func getFamilyFromName(s sku) string {
	if !strings.HasPrefix(strings.ToLower(s.Name), "gp_") && !strings.HasPrefix(strings.ToLower(s.Name), "bc_") {
		return ""
	}

	nameFamily := s.Name[3:]
	retFamily := "Gen4" // Default

	if strings.EqualFold(nameFamily, "Gen5") {
		retFamily = "Gen5"
	}

	return retFamily
}

func getDTUCapacityErrorMsg(s sku) string {
	m := getDTUMaxGB[strings.ToLower(s.Tier)]
	stub := fmt.Sprintf("service tier '%s' must have a 'capacity'(%d) of ", s.Tier, s.Capacity)
	return buildErrorString(stub, m) + " DTUs"
}

func getVCoreCapacityErrorMsg(s sku) string {
	m := getvCoreMaxGB[strings.ToLower(s.Tier)][strings.ToLower(s.Family)]
	stub := fmt.Sprintf("service tier '%s' %s must have a 'capacity'(%d) of ", s.Tier, s.Family, s.Capacity)
	return buildErrorString(stub, m) + " vCores"
}

func getDTUNotValidSizeErrorMsg(s sku) string {
	m := supportedDTUMaxGBValues
	stub := fmt.Sprintf("'max_size_gb'(%d) is not a valid value for service tier '%s', 'max_size_gb' must have a value of ", int(s.MaxSizeGb), s.Tier)
	return buildErrorString(stub, m) + " GB"
}

func buildErrorString(stub string, m map[int]float64) string {
	var a []int

	// copy the keys into another map
	p := make([]int, 0, len(m))
	for k := range m {
		p = append(p, k)
	}

	// copy the values of the map of keys into a slice of ints
	for v := range p {
		a = append(a, p[v])
	}

	// sort the slice to get them in order
	sort.Ints(a)

	// build the error message
	for i := range a {
		if i < len(a)-1 {
			stub += fmt.Sprintf("%d, ", a[i])
		} else {
			stub += fmt.Sprintf("or %d", a[i])
		}
	}

	return stub
}

func doDTUSKUValidation(s sku) error {

	if s.MaxAllowedGB == 0 {
		return fmt.Errorf(getDTUCapacityErrorMsg(s))
	}

	if strings.EqualFold(s.Name, "BasicPool") {
		// Basic SKU does not let you pick your max_size_GB they are fixed values
		if s.MaxSizeGb != s.MaxAllowedGB {
			return fmt.Errorf("service tier 'Basic' with a 'capacity' of %d must have a 'max_size_gb' of %.7f GB, got %.7f GB", s.Capacity, s.MaxAllowedGB, s.MaxSizeGb)
		}
	} else {
		// All other DTU based SKUs
		if s.MaxSizeGb > s.MaxAllowedGB {
			return fmt.Errorf("service tier '%s' with a 'capacity' of %d must have a 'max_size_gb' no greater than %d GB, got %d GB", s.Tier, s.Capacity, int(s.MaxAllowedGB), int(s.MaxSizeGb))
		}

		if int(s.MaxSizeGb) < 50 {
			return fmt.Errorf("service tier '%s', must have a 'max_size_gb' value equal to or greater than 50 GB, got %d GB", s.Tier, int(s.MaxSizeGb))
		}

		// Check to see if the max_size_gb value is valid for this SKU type and capacity
		if supportedDTUMaxGBValues[int(s.MaxSizeGb)] != 1 {
			return fmt.Errorf(getDTUNotValidSizeErrorMsg(s))
		}
	}

	// All Other DTU based SKU Checks
	if s.MinCapacity != math.Trunc(s.MinCapacity) {
		return fmt.Errorf("service tier '%s' must have whole numbers as their 'minCapacity'", s.Tier)
	}

	if s.MaxCapacity != math.Trunc(s.MaxCapacity) {
		return fmt.Errorf("service tier '%s' must have whole numbers as their 'maxCapacity'", s.Tier)
	}

	return nil
}

func doVCoreSKUValidation(s sku) error {

	if s.MaxAllowedGB == 0 {
		return fmt.Errorf(getVCoreCapacityErrorMsg(s))
	}

	if s.MaxSizeGb > s.MaxAllowedGB {
		return fmt.Errorf("service tier '%s' %s with a 'capacity' of %d vCores must have a 'max_size_gb' between 5 GB and %d GB, got %d GB", s.Tier, s.Family, s.Capacity, int(s.MaxAllowedGB), int(s.MaxSizeGb))
	}

	if int(s.MaxSizeGb) < 5 {
		return fmt.Errorf("service tier '%s' must have a 'max_size_gb' value equal to or greater than 5 GB, got %d GB", s.Tier, int(s.MaxSizeGb))
	}

	if s.MaxSizeGb != math.Trunc(s.MaxSizeGb) {
		return fmt.Errorf("'max_size_gb' must be a whole number, got %f GB", s.MaxSizeGb)
	}

	if s.MaxCapacity > float64(s.Capacity) {
		return fmt.Errorf("service tier '%s' perDatabaseSettings 'maxCapacity'(%d) must not be higher than the SKUs 'capacity'(%d) value", s.Tier, int(s.MaxCapacity), s.Capacity)
	}

	if s.MinCapacity > s.MaxCapacity {
		return fmt.Errorf("perDatabaseSettings 'maxCapacity'(%d) must be greater than or equal to the perDatabaseSettings 'minCapacity'(%d) value", int(s.MaxCapacity), int(s.MinCapacity))
	}

	return nil
}
