package lua

import (
	"reflect"
	"unsafe"
)

// iface is an internal representation of the go-interface.
type iface struct {
	itab unsafe.Pointer
	word unsafe.Pointer
}

const preloadLimit LNumber = 128

var _fv float64
var _uv uintptr

// allocator is a fast bulk memory allocator for the LValue.
type allocator struct {
	top         int
	size        int
	nptrs       []LValue
	nheader     *reflect.SliceHeader
	fptrs       []float64
	fheader     *reflect.SliceHeader
	itabLNumber unsafe.Pointer
	preloads    [int(preloadLimit)]LValue
}

func newAllocator(size int) *allocator {
	al := &allocator{
		top:         0,
		size:        size,
		nptrs:       make([]LValue, size),
		nheader:     nil,
		fptrs:       make([]float64, size),
		fheader:     nil,
		itabLNumber: unsafe.Pointer(nil),
	}
	al.nheader = (*reflect.SliceHeader)(unsafe.Pointer(&al.nptrs))
	al.fheader = (*reflect.SliceHeader)(unsafe.Pointer(&al.fptrs))

	var v LValue = LNumber(0)
	vp := (*iface)(unsafe.Pointer(&v))
	al.itabLNumber = vp.itab
	for i := 0; i < int(preloadLimit); i++ {
		al.preloads[i] = LNumber(i)
	}
	return al
}

func (al *allocator) LNumber2I(v LNumber) LValue {
	if v >= 0 && v < preloadLimit && float64(v) == float64(int64(v)) {
		return al.preloads[int(v)]
	}
	if al.top == len(al.nptrs)-1 {
		al.top = 0
		al.nptrs = make([]LValue, al.size)
		al.nheader = (*reflect.SliceHeader)(unsafe.Pointer(&al.nptrs))
		al.fptrs = make([]float64, al.size)
		al.fheader = (*reflect.SliceHeader)(unsafe.Pointer(&al.fptrs))
	}
	fptr := (*float64)(unsafe.Pointer(al.fheader.Data + uintptr(al.top)*unsafe.Sizeof(_fv)))
	e := *(*LValue)(unsafe.Pointer(al.nheader.Data + uintptr(al.top)*unsafe.Sizeof(_uv)))
	al.top++

	ep := (*iface)(unsafe.Pointer(&e))
	ep.itab = al.itabLNumber
	*fptr = float64(v)
	ep.word = unsafe.Pointer(fptr)
	return e
}
