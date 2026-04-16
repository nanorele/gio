package byteslice

import (
	"reflect"
	"unsafe"
)

func Struct(s any) []byte {
	v := reflect.ValueOf(s)
	sz := int(v.Elem().Type().Size())
	return unsafe.Slice((*byte)(unsafe.Pointer(v.Pointer())), sz)
}

func Uint32(s []uint32) []byte {
	n := len(s)
	if n == 0 {
		return nil
	}
	blen := n * int(unsafe.Sizeof(s[0]))
	return unsafe.Slice((*byte)(unsafe.Pointer(&s[0])), blen)
}

func Slice(s any) []byte {
	v := reflect.ValueOf(s)
	first := v.Index(0)
	sz := int(first.Type().Size())
	res := unsafe.Slice((*byte)(unsafe.Pointer(v.Pointer())), sz*v.Cap())
	return res[:sz*v.Len()]
}
