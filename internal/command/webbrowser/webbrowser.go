package webbrowser

// Launcher is an object that knows how to open a given URL in a new tab in
// some suitable browser on the current system.
//
// Launching of browsers is a very target-platform-sensitive activity, so
// this interface serves as an abstraction over many possible implementations
// which can be selected based on what is appropriate for a specific situation.
type Launcher interface {
	// OpenURL opens the given URL in a web browser.
	//
	// Depending on the circumstances and on the target platform, this may or
	// may not cause the browser to take input focus. Because of this
	// uncertainty, any caller of this method must be sure to include some
	// language in its UI output to let the user know that a browser tab has
	// opened somewhere, so that they can go and find it if the focus didn't
	// switch automatically.
	OpenURL(url string) error
}
