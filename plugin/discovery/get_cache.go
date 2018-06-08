package discovery

// PluginCache is an interface implemented by objects that are able to maintain
// a cache of plugins.
type PluginCache interface {
	// CachedPluginPath returns a path where the requested plugin is already
	// cached, or an empty string if the requested plugin is not yet cached.
	CachedPluginPath(kind string, name string, version Version) string

	// InstallDir returns the directory that new plugins should be installed into
	// in order to populate the cache. This directory should be used as the
	// first argument to getter.Get when downloading plugins with go-getter.
	//
	// After installing into this directory, use CachedPluginPath to obtain the
	// path where the plugin was installed.
	InstallDir() string
}

// NewLocalPluginCache returns a PluginCache that caches plugins in a
// given local directory.
func NewLocalPluginCache(dir string) PluginCache {
	return &pluginCache{
		Dir: dir,
	}
}

type pluginCache struct {
	Dir string
}

func (c *pluginCache) CachedPluginPath(kind string, name string, version Version) string {
	allPlugins := FindPlugins(kind, []string{c.Dir})
	plugins := allPlugins.WithName(name).WithVersion(version)

	if plugins.Count() == 0 {
		// nothing cached
		return ""
	}

	// There should generally be only one plugin here; if there's more than
	// one match for some reason then we'll just choose one arbitrarily.
	plugin := plugins.Newest()
	return plugin.Path
}

func (c *pluginCache) InstallDir() string {
	return c.Dir
}
