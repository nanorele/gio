package byteslice

import (
	"bytes"
	"reflect"
	"testing"
)

func TestStruct(t *testing.T) {
	type TestData struct {
		A uint32
		B uint16
		C uint8
	}
	td := TestData{A: 0x11223344, B: 0x5566, C: 0x77}
	got := Struct(&td)
	
	expectedSize := int(reflect.TypeOf(td).Size())
	if len(got) != expectedSize {
		t.Errorf("Struct() length = %v, want %v", len(got), expectedSize)
	}
	
	// We can't easily check the content without knowing the memory layout,
	// but we can check if it reflects the data.
	// Since it's unsafe.Slice, it should be the same memory.
}

func TestUint32(t *testing.T) {
	s := []uint32{0x11223344, 0x55667788}
	got := Uint32(s)
	
	if len(got) != 8 {
		t.Errorf("Uint32() length = %v, want 8", len(got))
	}
	
	// Empty slice
	if Uint32(nil) != nil {
		t.Errorf("Uint32(nil) should be nil")
	}
	if Uint32([]uint32{}) != nil {
		t.Errorf("Uint32([]) should be nil")
	}
}

func TestSlice(t *testing.T) {
	s := []uint16{0x1122, 0x3344, 0x5566}
	got := Slice(s)
	
	if len(got) != 6 {
		t.Errorf("Slice() length = %v, want 6", len(got))
	}
	
	// Check content (assuming little endian for common systems, but we can just compare)
	// Actually, Slice uses reflect.Pointer which should be the start of the slice data.
	
	s2 := []byte{1, 2, 3}
	got2 := Slice(s2)
	if !bytes.Equal(s2, got2) {
		t.Errorf("Slice([]byte) = %v, want %v", got2, s2)
	}
}
