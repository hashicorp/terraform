package internal

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"io/ioutil"
)

func compress(b []byte) ([]byte, error) {
	buf := bytes.Buffer{}
	w := zlib.NewWriter(&buf)
	_, err := w.Write(b)
	w.Close()

	if nil != err {
		return nil, err
	}

	return buf.Bytes(), nil
}

func uncompress(b []byte) ([]byte, error) {
	buf := bytes.NewBuffer(b)
	r, err := zlib.NewReader(buf)
	if nil != err {
		return nil, err
	}
	defer r.Close()

	return ioutil.ReadAll(r)
}

func compressEncode(b []byte) (string, error) {
	compressed, err := compress(b)

	if nil != err {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(compressed), nil
}

func uncompressDecode(s string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if nil != err {
		return nil, err
	}

	return uncompress(decoded)
}
