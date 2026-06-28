package app

import (
	"testing"
	"unsafe"
)

func ptrOf[T any](p *T) uintptr { return uintptr(unsafe.Pointer(p)) }

func TestDecodePOINTL(t *testing.T) {
	cases := []struct{ x, y int32 }{
		{0, 0},
		{10, 20},
		{1920, 1080},
		{-5, -10}, // multi-monitor setups can place the cursor at negative coords
		{-1, 2147483647},
	}
	for _, c := range cases {
		pt := uintptr(uint32(c.x)) | uintptr(uint32(c.y))<<32
		gotX, gotY := decodePOINTL(pt)
		if gotX != c.x || gotY != c.y {
			t.Errorf("decodePOINTL(%#x) = (%d,%d), want (%d,%d)", pt, gotX, gotY, c.x, c.y)
		}
	}
}

func TestGUIDEqual(t *testing.T) {
	if !guidEqual(&iidIDropTarget, &iidIDropTarget) {
		t.Error("a GUID must equal itself")
	}
	if guidEqual(&iidIDropTarget, &iidIUnknown) {
		t.Error("IDropTarget and IUnknown GUIDs must differ")
	}
	// Differ only in the final Data4 byte.
	a := iidIUnknown
	b := iidIUnknown
	b.Data4[7] ^= 0xFF
	if guidEqual(&a, &b) {
		t.Error("GUIDs differing in Data4 must not be equal")
	}
}

// dropQueryInterface must hand back the object for IUnknown / IDropTarget and
// reject anything else with E_NOINTERFACE.
func TestDropQueryInterface(t *testing.T) {
	var this uintptr = 0xDEAD00
	var out uintptr
	if hr := dropQueryInterface(this, ptrOf(&iidIDropTarget), ptrOf(&out)); hr != _S_OK || out != this {
		t.Errorf("QueryInterface(IDropTarget) hr=%#x out=%#x, want S_OK and this", hr, out)
	}
	out = 0
	if hr := dropQueryInterface(this, ptrOf(&iidIUnknown), ptrOf(&out)); hr != _S_OK || out != this {
		t.Errorf("QueryInterface(IUnknown) hr=%#x out=%#x, want S_OK and this", hr, out)
	}
	out = 0xBEEF
	other := iidIUnknown
	other.Data1 = 0x12345678
	if hr := dropQueryInterface(this, ptrOf(&other), ptrOf(&out)); hr != _E_NOINTERFACE || out != 0 {
		t.Errorf("QueryInterface(other) hr=%#x out=%#x, want E_NOINTERFACE and null", hr, out)
	}
}
