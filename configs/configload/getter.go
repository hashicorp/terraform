package configload

import (
	"log"
	"path/filepath"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	getter "github.com/hashicorp/go-getter"
)

// We configure our own go-getter detector and getter sets here, because
// the set of sources we support is part of Terraform's documentation and
// so we don't want any new sources introduced in go-getter to sneak in here
// and work even though they aren't documented. This also insulates us from
// any meddling that might be done by other go-getter callers linked into our
// executable.

var goGetterDetectors = []getter.Detector{
	new(getter.GitHubDetector),
	new(getter.BitBucketDetector),
	new(getter.S3Detector),
	new(getter.FileDetector),
}

var goGetterNoDetectors = []getter.Detector{}

var goGetterDecompressors = map[string]getter.Decompressor{
	"bz2": new(getter.Bzip2Decompressor),
	"gz":  new(getter.GzipDecompressor),
	"xz":  new(getter.XzDecompressor),
	"zip": new(getter.ZipDecompressor),

	"tar.bz2":  new(getter.TarBzip2Decompressor),
	"tar.tbz2": new(getter.TarBzip2Decompressor),

	"tar.gz": new(getter.TarGzipDecompressor),
	"tgz":    new(getter.TarGzipDecompressor),

	"tar.xz": new(getter.TarXzDecompressor),
	"txz":    new(getter.TarXzDecompressor),
}

var goGetterGetters = map[string]getter.Getter{
	"file":  new(getter.FileGetter),
	"git":   new(getter.GitGetter),
	"hg":    new(getter.HgGetter),
	"s3":    new(getter.S3Getter),
	"http":  getterHTTPGetter,
	"https": getterHTTPGetter,
}

var getterHTTPClient = cleanhttp.DefaultClient()

var getterHTTPGetter = &getter.HttpGetter{
	Client: getterHTTPClient,
	Netrc:  true,
}

// getWithGoGetter retrieves the package referenced in the given address
// into the installation path and then returns the full path to any subdir
// indicated in the address.
//
// The errors returned by this function are those surfaced by the underlying
// go-getter library, which have very inconsistent quality as
// end-user-actionable error messages. At this time we do not have any
// reasonable way to improve these error messages at this layer because
// the underlying errors are not separatelyr recognizable.
func getWithGoGetter(instPath, addr string) (string, error) {
	packageAddr, subDir := splitAddrSubdir(addr)

	log.Printf("[DEBUG] will download %q to %s", packageAddr, instPath)

	realAddr, err := getter.Detect(packageAddr, instPath, getter.Detectors)
	if err != nil {
		return "", err
	}

	var realSubDir string
	realAddr, realSubDir = splitAddrSubdir(realAddr)
	if realSubDir != "" {
		subDir = filepath.Join(realSubDir, subDir)
	}

	if realAddr != packageAddr {
		log.Printf("[TRACE] go-getter detectors rewrote %q to %q", packageAddr, realAddr)
	}

	client := getter.Client{
		Src: realAddr,
		Dst: instPath,
		Pwd: instPath,

		Mode: getter.ClientModeDir,

		Detectors:     goGetterNoDetectors, // we already did detection above
		Decompressors: goGetterDecompressors,
		Getters:       goGetterGetters,
	}
	err = client.Get()
	if err != nil {
		return "", err
	}

	// Our subDir string can contain wildcards until this point, so that
	// e.g. a subDir of * can expand to one top-level directory in a .tar.gz
	// archive. Now that we've expanded the archive successfully we must
	// resolve that into a concrete path.
	var finalDir string
	if subDir != "" {
		finalDir, err = getter.SubdirGlob(instPath, subDir)
		log.Printf("[TRACE] expanded %q to %q", subDir, finalDir)
		if err != nil {
			return "", err
		}
	} else {
		finalDir = instPath
	}

	// If we got this far then we have apparently succeeded in downloading
	// the requested object!
	return filepath.Clean(finalDir), nil
}
