package plugin

import (
	"fmt"
	"path"
	"runtime"

	"github.com/hashicorp/terraform/internal/tfdiags"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// grpcErr extracts some known error types and formats them into better
// representations for core. This must only be called from plugin methods.
// Since we don't use RPC status errors for the plugin protocol, these do not
// contain any useful details, and we can return some text that at least
// indicates the plugin call and possible error condition.
func grpcErr(err error) (diags tfdiags.Diagnostics) {
	if err == nil {
		return
	}

	// extract the method name from the caller.
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		logger.Error("unknown grpc call", "error", err)
		return diags.Append(err)
	}

	f := runtime.FuncForPC(pc)

	// Function names will contain the full import path. Take the last
	// segment, which will let users know which method was being called.
	_, requestName := path.Split(f.Name())

	// Here we can at least correlate the error in the logs to a particular binary.
	logger.Error(requestName, "error", err)

	// TODO: while this expands the error codes into somewhat better messages,
	// this still does not easily link the error to an actual user-recognizable
	// plugin. The grpc plugin does not know its configured name, and the
	// errors are in a list of diagnostics, making it hard for the caller to
	// annotate the returned errors.
	switch status.Code(err) {
	case codes.Unavailable:
		// This case is when the plugin has stopped running for some reason,
		// and is usually the result of a crash.
		diags = diags.Append(tfdiags.WholeContainingBody(
			tfdiags.Error,
			"Plugin did not respond",
			fmt.Sprintf("The plugin encountered an error, and failed to respond to the %s call. "+
				"The plugin logs may contain more details.", requestName),
		))
	case codes.Canceled:
		diags = diags.Append(tfdiags.WholeContainingBody(
			tfdiags.Error,
			"Request cancelled",
			fmt.Sprintf("The %s request was cancelled.", requestName),
		))
	case codes.Unimplemented:
		diags = diags.Append(tfdiags.WholeContainingBody(
			tfdiags.Error,
			"Unsupported plugin method",
			fmt.Sprintf("The %s method is not supported by this plugin.", requestName),
		))
	default:
		diags = diags.Append(tfdiags.WholeContainingBody(
			tfdiags.Error,
			"Plugin error",
			fmt.Sprintf("The plugin returned an unexpected error from %s: %v", requestName, err),
		))
	}
	return
}
