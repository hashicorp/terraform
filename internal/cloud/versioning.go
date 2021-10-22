package cloud

// This simple map exists to translate TFP-API-Version strings to the TFE release where it was
// introduced, to provide actionable feedback on features that may be unsupported by the TFE
// installation but present in this version of Terraform.
//
// The cloud package here, introduced in Terraform 1.1.0, requires a minimum of 2.5 (v202201-1)
// The TFP-API-Version header that this refers to was introduced in 2.3 (v202006-1), so an absent
// header can be considered < 2.3.
// var apiToMinimumTFEVersion = map[string]string{
// 	"2.5": "v202201-1",
// }
