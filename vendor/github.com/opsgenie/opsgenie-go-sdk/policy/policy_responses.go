package policy

// EnablePolicyResponse holds the result data of the EnablePolicyRequest.
type EnablePolicyResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// DisablePolicyResponse holds the result data of the DisablePolicyRequest.
type DisablePolicyResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}
