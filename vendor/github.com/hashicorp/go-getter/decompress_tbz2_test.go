package getter

import (
	"path/filepath"
	"testing"
)

func TestTarBzip2Decompressor(t *testing.T) {
	cases := []TestDecompressCase{
		{
			"empty.tar.bz2",
			false,
			true,
			nil,
			"",
		},

		{
			"single.tar.bz2",
			false,
			false,
			nil,
			"d3b07384d113edec49eaa6238ad5ff00",
		},

		{
			"single.tar.bz2",
			true,
			false,
			[]string{"file"},
			"",
		},

		{
			"multiple.tar.bz2",
			true,
			false,
			[]string{"file1", "file2"},
			"",
		},

		{
			"multiple.tar.bz2",
			false,
			true,
			nil,
			"",
		},
	}

	for i, tc := range cases {
		cases[i].Input = filepath.Join("./test-fixtures", "decompress-tbz2", tc.Input)
	}

	TestDecompressor(t, new(TarBzip2Decompressor), cases)
}
