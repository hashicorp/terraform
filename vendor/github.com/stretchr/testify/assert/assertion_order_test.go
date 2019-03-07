package assert

import (
	"reflect"
	"testing"
)

func TestCompare(t *testing.T) {
	for _, currCase := range []struct {
		less    interface{}
		greater interface{}
		cType   string
	}{
		{less: "a", greater: "b", cType: "string"},
		{less: int(1), greater: int(2), cType: "int"},
		{less: int8(1), greater: int8(2), cType: "int8"},
		{less: int16(1), greater: int16(2), cType: "int16"},
		{less: int32(1), greater: int32(2), cType: "int32"},
		{less: int64(1), greater: int64(2), cType: "int64"},
		{less: uint8(1), greater: uint8(2), cType: "uint8"},
		{less: uint16(1), greater: uint16(2), cType: "uint16"},
		{less: uint32(1), greater: uint32(2), cType: "uint32"},
		{less: uint64(1), greater: uint64(2), cType: "uint64"},
		{less: float32(1), greater: float32(2), cType: "float32"},
		{less: float64(1), greater: float64(2), cType: "float64"},
	} {
		resLess, isComparable := compare(currCase.less, currCase.greater, reflect.ValueOf(currCase.less).Kind())
		if !isComparable {
			t.Error("object should be comparable for type " + currCase.cType)
		}

		if resLess != 1 {
			t.Errorf("object less should be less than greater for type " + currCase.cType)
		}

		resGreater, isComparable := compare(currCase.greater, currCase.less, reflect.ValueOf(currCase.less).Kind())
		if !isComparable {
			t.Error("object are comparable for type " + currCase.cType)
		}

		if resGreater != -1 {
			t.Errorf("object greater should be greater than less for type " + currCase.cType)
		}

		resEqual, isComparable := compare(currCase.less, currCase.less, reflect.ValueOf(currCase.less).Kind())
		if !isComparable {
			t.Error("object are comparable for type " + currCase.cType)
		}

		if resEqual != 0 {
			t.Errorf("objects should be equal for type " + currCase.cType)
		}
	}
}

func TestGreater(t *testing.T) {
	mockT := new(testing.T)

	if !Greater(mockT, 2, 1) {
		t.Error("Greater should return true")
	}

	if Greater(mockT, 1, 1) {
		t.Error("Greater should return false")
	}

	if Greater(mockT, 1, 2) {
		t.Error("Greater should return false")
	}
}

func TestGreaterOrEqual(t *testing.T) {
	mockT := new(testing.T)

	if !GreaterOrEqual(mockT, 2, 1) {
		t.Error("Greater should return true")
	}

	if !GreaterOrEqual(mockT, 1, 1) {
		t.Error("Greater should return true")
	}

	if GreaterOrEqual(mockT, 1, 2) {
		t.Error("Greater should return false")
	}
}

func TestLess(t *testing.T) {
	mockT := new(testing.T)

	if !Less(mockT, 1, 2) {
		t.Error("Less should return true")
	}

	if Less(mockT, 1, 1) {
		t.Error("Less should return false")
	}

	if Less(mockT, 2, 1) {
		t.Error("Less should return false")
	}
}

func TestLessOrEqual(t *testing.T) {
	mockT := new(testing.T)

	if !LessOrEqual(mockT, 1, 2) {
		t.Error("Greater should return true")
	}

	if !LessOrEqual(mockT, 1, 1) {
		t.Error("Greater should return true")
	}

	if LessOrEqual(mockT, 2, 1) {
		t.Error("Greater should return false")
	}
}
