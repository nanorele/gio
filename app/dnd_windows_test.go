package app

import (
	"testing"
	"unsafe"

	"github.com/nanorele/gio/app/internal/windows"
)

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
//
// The callback is only ever invoked by OLE with native pointers, so the test
// mimics that with OS-allocated memory. Passing addresses of Go variables
// round-tripped through uintptr would violate unsafe rules — the stack can
// move between taking the address and using it — and checkptr (enabled by
// -race) rightly flags that.
func TestDropQueryInterface(t *testing.T) {
	h, err := windows.GlobalAlloc(64)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.GlobalFree(h)
	mem, err := windows.GlobalLock(h)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.GlobalUnlock(h)

	riid := (*_GUID)(mem)
	outMem := unsafe.Add(mem, 32)
	out := (*uintptr)(outMem)
	riidArg := uintptr(mem)
	outArg := uintptr(outMem)

	var this uintptr = 0xDEAD00
	*riid = iidIDropTarget
	if hr := dropQueryInterface(this, riidArg, outArg); hr != _S_OK || *out != this {
		t.Errorf("QueryInterface(IDropTarget) hr=%#x out=%#x, want S_OK and this", hr, *out)
	}
	*riid = iidIUnknown
	*out = 0
	if hr := dropQueryInterface(this, riidArg, outArg); hr != _S_OK || *out != this {
		t.Errorf("QueryInterface(IUnknown) hr=%#x out=%#x, want S_OK and this", hr, *out)
	}
	*riid = iidIUnknown
	riid.Data1 = 0x12345678
	*out = 0xBEEF
	if hr := dropQueryInterface(this, riidArg, outArg); hr != _E_NOINTERFACE || *out != 0 {
		t.Errorf("QueryInterface(other) hr=%#x out=%#x, want E_NOINTERFACE and null", hr, *out)
	}
}
