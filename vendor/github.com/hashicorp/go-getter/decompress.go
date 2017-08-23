package getter

// Decompressor defines the interface that must be implemented to add
// support for decompressing a type.
type Decompressor interface {
	// Decompress should decompress src to dst. dir specifies whether dst
	// is a directory or single file. src is guaranteed to be a single file
	// that exists. dst is not guaranteed to exist already.
	Decompress(dst, src string, dir bool) error
}

// Decompressors is the mapping of extension to the Decompressor implementation
// that will decompress that extension/type.
var Decompressors map[string]Decompressor

func init() {
	tbzDecompressor := new(TarBzip2Decompressor)
	tgzDecompressor := new(TarGzipDecompressor)
	txzDecompressor := new(TarXzDecompressor)

	Decompressors = map[string]Decompressor{
		"bz2":     new(Bzip2Decompressor),
		"gz":      new(GzipDecompressor),
		"xz":      new(XzDecompressor),
		"tar.bz2": tbzDecompressor,
		"tar.gz":  tgzDecompressor,
		"tar.xz":  txzDecompressor,
		"tbz2":    tbzDecompressor,
		"tgz":     tgzDecompressor,
		"txz":     txzDecompressor,
		"zip":     new(ZipDecompressor),
	}
}
