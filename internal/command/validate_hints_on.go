//go:build validatehints

package command

// validateEnableHints is a build-time flag (controlled by a build tag) which
// decides whether to enable the experimental "terraform validate -hints"
// option.
const validateEnableHints = true
