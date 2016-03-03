package mssql

import (
	"testing"
)

func TestToString(t *testing.T) {
	values := []struct {
		dec Decimal
		s   string
	}{
		{Decimal{positive: true, prec: 10, scale: 0, integer: [4]uint32{1, 0, 0, 0}}, "1"},
		{Decimal{positive: false, prec: 10, scale: 0, integer: [4]uint32{1, 0, 0, 0}}, "-1"},
		{Decimal{positive: true, prec: 10, scale: 1, integer: [4]uint32{1, 0, 0, 0}}, "0.1"},
		{Decimal{positive: false, prec: 10, scale: 1, integer: [4]uint32{1, 0, 0, 0}}, "-0.1"},
		{Decimal{positive: true, prec: 10, scale: 2, integer: [4]uint32{100, 0, 0, 0}}, "1.00"},
		{Decimal{positive: false, prec: 10, scale: 2, integer: [4]uint32{100, 0, 0, 0}}, "-1.00"},
		{Decimal{positive: true, prec: 30, scale: 0, integer: [4]uint32{0, 1, 0, 0}}, "4294967296"},           // 2^32
		{Decimal{positive: true, prec: 30, scale: 0, integer: [4]uint32{0, 0, 1, 0}}, "18446744073709551616"}, // 2^64
		{Decimal{positive: true, prec: 30, scale: 0, integer: [4]uint32{0, 1, 1, 0}}, "18446744078004518912"}, // 2^64+2^32
	}
	for _, v := range values {
		if v.dec.String() != v.s {
			t.Error("String values don't match ", v.dec.String(), v.s)
		}
	}
}

func TestToFloat64(t *testing.T) {
	values := []struct {
		dec Decimal
		flt float64
	}{
		{Decimal{positive: true, prec: 1},
			0.0},
		{Decimal{positive: true, prec: 1, integer: [4]uint32{1}},
			1.0},
		{Decimal{positive: false, prec: 1, integer: [4]uint32{1}},
			-1.0},
		{Decimal{positive: true, prec: 1, scale: 1, integer: [4]uint32{5}},
			0.5},
		{Decimal{positive: true, prec: 38, integer: [4]uint32{0xffffffff, 0xffffffff, 0xffffffff, 0xffffffff}},
			3.402823669209385e+38},
		{Decimal{positive: true, prec: 38, scale: 3, integer: [4]uint32{0xffffffff, 0xffffffff, 0xffffffff, 0xffffffff}},
			3.402823669209385e+35},
	}
	for _, v := range values {
		if v.dec.ToFloat64() != v.flt {
			t.Error("ToFloat values don't match ", v.dec.ToFloat64(), v.flt)
		}
	}
}

func TestFromFloat64(t *testing.T) {
	values := []struct {
		dec Decimal
		flt float64
	}{
		{Decimal{positive: true, prec: 20},
			0.0},
		{Decimal{positive: true, prec: 20, integer: [4]uint32{1}},
			1.0},
		{Decimal{positive: false, prec: 20, integer: [4]uint32{1}},
			-1.0},
		{Decimal{positive: true, prec: 20, scale: 1, integer: [4]uint32{5}},
			0.5},
		{Decimal{positive: true, prec: 20, integer: [4]uint32{0, 0, 0xfffff000, 0xffffffff}},
			3.402823669209384e+38},
		//{Decimal{positive: true, prec: 20, scale: 3, integer: [4]uint32{0, 0, 0xfffff000, 0xffffffff}},
		// 3.402823669209385e+35},
	}
	for _, v := range values {
		decfromflt, err := Float64ToDecimal(v.flt)
		if err == nil {
			if decfromflt != v.dec {
				t.Error("FromFloat values don't match ", decfromflt, v.dec)
			}
		} else {
			t.Error("Float64ToDecimal failed with error:", err.Error())
		}
	}
}
