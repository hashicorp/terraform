package aws

import (
	"testing"
)

func TestAccAWSConfig(t *testing.T) {
	testCases := map[string]map[string]func(t *testing.T){
		"Config": {
			"basic":        testAccConfigConfigRule_basic,
			"ownerAws":     testAccConfigConfigRule_ownerAws,
			"customlambda": testAccConfigConfigRule_customlambda,
			"importAws":    testAccConfigConfigRule_importAws,
			"importLambda": testAccConfigConfigRule_importLambda,
		},
		"ConfigurationRecorderStatus": {
			"basic":        testAccConfigConfigurationRecorderStatus_basic,
			"startEnabled": testAccConfigConfigurationRecorderStatus_startEnabled,
			"importBasic":  testAccConfigConfigurationRecorderStatus_importBasic,
		},
		"ConfigurationRecorder": {
			"basic":       testAccConfigConfigurationRecorder_basic,
			"allParams":   testAccConfigConfigurationRecorder_allParams,
			"importBasic": testAccConfigConfigurationRecorder_importBasic,
		},
		"DeliveryChannel": {
			"basic":       testAccConfigDeliveryChannel_basic,
			"allParams":   testAccConfigDeliveryChannel_allParams,
			"importBasic": testAccConfigDeliveryChannel_importBasic,
		},
	}

	for group, m := range testCases {
		m := m
		t.Run(group, func(t *testing.T) {
			for name, tc := range m {
				tc := tc
				t.Run(name, func(t *testing.T) {
					tc(t)
				})
			}
		})
	}
}
