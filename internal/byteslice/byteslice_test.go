package byteslice

import (
	"bytes"
	"reflect"
	"testing"
	"unsafe"
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

// TestSlice_Empty verifies Slice does not panic on a zero-length slice and
// returns an empty byte slice.
func TestSlice_Empty(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Slice([]int32{}) panicked: %v", r)
		}
	}()
	got := Slice([]int32{})
	if len(got) != 0 {
		t.Errorf("Slice([]int32{}) length = %d, want 0", len(got))
	}

	// Also exercise other element types and a nil slice.
	if got := Slice([]uint16(nil)); len(got) != 0 {
		t.Errorf("Slice(nil []uint16) length = %d, want 0", len(got))
	}
	if got := Slice([]uint64{}); len(got) != 0 {
		t.Errorf("Slice([]uint64{}) length = %d, want 0", len(got))
	}
}

func TestStruct_AliasesMemory(t *testing.T) {
	type S struct {
		A uint32
		B uint16
	}
	s := S{A: 0xDEADBEEF, B: 0x1234}
	origA := s.A
	b := Struct(&s)
	if len(b) != int(unsafe.Sizeof(s)) {
		t.Fatalf("len = %d, want %d", len(b), int(unsafe.Sizeof(s)))
	}
	// Mutate first byte of A through the alias and verify the struct sees it.
	orig := b[0]
	b[0] = ^orig
	if s.A == origA {
		t.Errorf("Struct() does not alias memory: A unchanged %#x", s.A)
	}
	// restore
	b[0] = orig
	if s.A != origA {
		t.Errorf("restore failed: %#x vs %#x", s.A, origA)
	}
}

func TestStruct_PointerOffset(t *testing.T) {
	type S struct {
		X uint64
	}
	s := &S{X: 0x0102030405060708}
	b := Struct(s)
	if uintptr(unsafe.Pointer(&b[0])) != uintptr(unsafe.Pointer(s)) {
		t.Errorf("Struct() returned slice not pointing at struct")
	}
}

func TestUint32_AliasesMemory(t *testing.T) {
	s := []uint32{0x01020304, 0x05060708}
	b := Uint32(s)
	if len(b) != 8 {
		t.Fatalf("len = %d, want 8", len(b))
	}
	// Verify alias - mutate a byte and observe in source slice.
	old := s[0]
	b[0] ^= 0xFF
	if s[0] == old {
		t.Errorf("Uint32() does not alias memory")
	}
	b[0] ^= 0xFF
	if s[0] != old {
		t.Errorf("restore failed: %#x vs %#x", s[0], old)
	}
}

func TestUint32_NilEmpty(t *testing.T) {
	if got := Uint32(nil); got != nil {
		t.Errorf("Uint32(nil) = %v, want nil", got)
	}
	if got := Uint32([]uint32{}); got != nil {
		t.Errorf("Uint32([]) = %v, want nil", got)
	}
}

func TestUint32_Length(t *testing.T) {
	cases := []int{1, 2, 3, 16, 100}
	for _, n := range cases {
		s := make([]uint32, n)
		got := Uint32(s)
		want := n * 4
		if len(got) != want {
			t.Errorf("Uint32(len=%d) byte length = %d, want %d", n, len(got), want)
		}
	}
}

func TestSlice_Uint16(t *testing.T) {
	s := []uint16{0x1122, 0x3344}
	b := Slice(s)
	if len(b) != 4 {
		t.Fatalf("len = %d, want 4", len(b))
	}
	if uintptr(unsafe.Pointer(&b[0])) != uintptr(unsafe.Pointer(&s[0])) {
		t.Errorf("Slice() returned non-aliasing pointer")
	}
}

func TestSlice_Float32(t *testing.T) {
	s := []float32{1, 2, 3, 4}
	b := Slice(s)
	want := len(s) * int(unsafe.Sizeof(float32(0)))
	if len(b) != want {
		t.Errorf("len = %d, want %d", len(b), want)
	}
}

func TestSlice_RespectsLenNotCap(t *testing.T) {
	s := make([]uint32, 3, 10)
	b := Slice(s)
	if len(b) != 3*4 {
		t.Errorf("len = %d, want %d", len(b), 3*4)
	}
}

func TestSlice_AliasesMemory(t *testing.T) {
	s := []uint32{0xAABBCCDD}
	b := Slice(s)
	if len(b) != 4 {
		t.Fatalf("len = %d, want 4", len(b))
	}
	old := s[0]
	b[0] ^= 0xFF
	if s[0] == old {
		t.Errorf("Slice() does not alias memory")
	}
}
