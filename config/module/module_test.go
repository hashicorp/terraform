package module

import (
	"net/url"
	"path/filepath"
)

const fixtureDir = "./test-fixtures"

func testModule(n string) string {
	p := filepath.Join(fixtureDir, n)
	p, err := filepath.Abs(p)
	if err != nil {
		panic(err)
	}

	var url url.URL
	url.Scheme = "file"
	url.Path = p
	return url.String()
}
