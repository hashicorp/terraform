package spotinst

// Logger specifies the interface for all log operations.
type Logger interface {
	Printf(format string, args ...interface{})
}
