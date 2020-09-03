package msgpack

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vmihailenco/msgpack/v4/codes"
)

type queryResult struct {
	query       string
	key         string
	hasAsterisk bool

	values []interface{}
}

func (q *queryResult) nextKey() {
	ind := strings.IndexByte(q.query, '.')
	if ind == -1 {
		q.key = q.query
		q.query = ""
		return
	}
	q.key = q.query[:ind]
	q.query = q.query[ind+1:]
}

// Query extracts data specified by the query from the msgpack stream skipping
// any other data. Query consists of map keys and array indexes separated with dot,
// e.g. key1.0.key2.
func (d *Decoder) Query(query string) ([]interface{}, error) {
	res := queryResult{
		query: query,
	}
	if err := d.query(&res); err != nil {
		return nil, err
	}
	return res.values, nil
}

func (d *Decoder) query(q *queryResult) error {
	q.nextKey()
	if q.key == "" {
		v, err := d.decodeInterfaceCond()
		if err != nil {
			return err
		}
		q.values = append(q.values, v)
		return nil
	}

	code, err := d.PeekCode()
	if err != nil {
		return err
	}

	switch {
	case code == codes.Map16 || code == codes.Map32 || codes.IsFixedMap(code):
		err = d.queryMapKey(q)
	case code == codes.Array16 || code == codes.Array32 || codes.IsFixedArray(code):
		err = d.queryArrayIndex(q)
	default:
		err = fmt.Errorf("msgpack: unsupported code=%x decoding key=%q", code, q.key)
	}
	return err
}

func (d *Decoder) queryMapKey(q *queryResult) error {
	n, err := d.DecodeMapLen()
	if err != nil {
		return err
	}
	if n == -1 {
		return nil
	}

	for i := 0; i < n; i++ {
		k, err := d.bytesNoCopy()
		if err != nil {
			return err
		}

		if string(k) == q.key {
			if err := d.query(q); err != nil {
				return err
			}
			if q.hasAsterisk {
				return d.skipNext((n - i - 1) * 2)
			}
			return nil
		}

		if err := d.Skip(); err != nil {
			return err
		}
	}

	return nil
}

func (d *Decoder) queryArrayIndex(q *queryResult) error {
	n, err := d.DecodeArrayLen()
	if err != nil {
		return err
	}
	if n == -1 {
		return nil
	}

	if q.key == "*" {
		q.hasAsterisk = true

		query := q.query
		for i := 0; i < n; i++ {
			q.query = query
			if err := d.query(q); err != nil {
				return err
			}
		}

		q.hasAsterisk = false
		return nil
	}

	ind, err := strconv.Atoi(q.key)
	if err != nil {
		return err
	}

	for i := 0; i < n; i++ {
		if i == ind {
			if err := d.query(q); err != nil {
				return err
			}
			if q.hasAsterisk {
				return d.skipNext(n - i - 1)
			}
			return nil
		}

		if err := d.Skip(); err != nil {
			return err
		}
	}

	return nil
}

func (d *Decoder) skipNext(n int) error {
	for i := 0; i < n; i++ {
		if err := d.Skip(); err != nil {
			return err
		}
	}
	return nil
}
