package heroku

import "testing"

func TestPipelineStage(t *testing.T) {
	valid := []string{
		"review",
		"development",
		"staging",
		"production",
	}
	for _, v := range valid {
		_, errors := validatePipelineStageName(v, "stage")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid stage: %q", v, errors)
		}
	}

	invalid := []string{
		"foobarbaz",
		"another-stage",
		"",
	}
	for _, v := range invalid {
		_, errors := validatePipelineStageName(v, "stage")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid stage", v)
		}
	}
}

func TestValidateUUID(t *testing.T) {
	valid := []string{
		"4812ccbc-2a2e-4c6c-bae4-a3d04ed51c0e",
	}
	for _, v := range valid {
		_, errors := validateUUID(v, "id")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid UUID: %q", v, errors)
		}
	}

	invalid := []string{
		"foobarbaz",
		"my-app-name",
	}
	for _, v := range invalid {
		_, errors := validateUUID(v, "id")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid UUID", v)
		}
	}
}
