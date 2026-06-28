// Shell drag-and-drop via OLE IDropTarget.
//
// gio's Windows backend uses RegisterDragDrop (rather than the simpler
// DragAcceptFiles/WM_DROPFILES path) so the application receives drag-enter,
// drag-over and drag-leave notifications — each carrying the cursor position —
// in addition to the final drop. That lets apps highlight the drop zone under
// the cursor while a drag is in progress, like a modern editor, and route the
// drop to whatever zone it landed on. See DragFilesEvent / DropFilesEvent.
//
//nolint:errcheck
package app

import (
	stdsyscall "syscall"
	"unsafe"

	syscall "golang.org/x/sys/windows"

	"github.com/nanorele/gio/app/internal/windows"
	"github.com/nanorele/gio/f32"
)

var (
	ole32                = syscall.NewLazySystemDLL("ole32.dll")
	procOleInitialize    = ole32.NewProc("OleInitialize")
	procOleUninitialize  = ole32.NewProc("OleUninitialize")
	procRegisterDragDrop = ole32.NewProc("RegisterDragDrop")
	procRevokeDragDrop   = ole32.NewProc("RevokeDragDrop")
	procReleaseStgMedium = ole32.NewProc("ReleaseStgMedium")

	shell32drop        = syscall.NewLazySystemDLL("shell32.dll")
	procDragQueryFileW = shell32drop.NewProc("DragQueryFileW")
)

const (
	_S_OK             = 0
	_E_NOINTERFACE    = 0x80004002
	_CF_HDROP         = 15
	_DVASPECT_CONTENT = 1
	_TYMED_HGLOBAL    = 1
	_DROPEFFECT_NONE  = 0
	_DROPEFFECT_COPY  = 1
)

type _GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var (
	iidIUnknown    = _GUID{0x00000000, 0x0000, 0x0000, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidIDropTarget = _GUID{0x00000122, 0x0000, 0x0000, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
)

type _FORMATETC struct {
	cfFormat uint16
	ptd      uintptr
	dwAspect uint32
	lindex   int32
	tymed    uint32
}

type _STGMEDIUM struct {
	tymed          uint32
	_              uint32
	unionData      uintptr // hGlobal (an HDROP for CF_HDROP)
	pUnkForRelease uintptr
}

// iDropTargetVtbl mirrors the COM IDropTarget vtable: three IUnknown methods
// followed by the four IDropTarget methods, all as raw function pointers.
type iDropTargetVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	DragEnter      uintptr
	DragOver       uintptr
	DragLeave      uintptr
	Drop           uintptr
}

// dropTarget is our IDropTarget COM object. The first field MUST be a pointer
// to the vtable so a *dropTarget can be handed to COM as an IDropTarget*; the
// callbacks recover the Go object by casting the `this` pointer back.
type dropTarget struct {
	vtbl *iDropTargetVtbl
	w    *window
}

// The vtable is shared by every window (the callbacks recover per-window state
// from the `this` pointer), so the syscall.NewCallback trampolines — a scarce
// process-wide resource — are created exactly once.
var dropTargetVtbl = &iDropTargetVtbl{
	QueryInterface: syscall.NewCallback(dropQueryInterface),
	AddRef:         syscall.NewCallback(dropAddRef),
	Release:        syscall.NewCallback(dropRelease),
	DragEnter:      syscall.NewCallback(dropDragEnter),
	DragOver:       syscall.NewCallback(dropDragOver),
	DragLeave:      syscall.NewCallback(dropDragLeave),
	Drop:           syscall.NewCallback(dropDrop),
}

// registerDropTarget initialises OLE on the (locked) window thread and registers
// the window as a drag-drop target. Safe to call once per window after the HWND
// exists; failures are non-fatal (the app simply receives no drop events).
func (w *window) registerDropTarget() {
	procOleInitialize.Call(0)
	w.dnd = &dropTarget{vtbl: dropTargetVtbl, w: w}
	procRegisterDragDrop.Call(uintptr(w.hwnd), uintptr(unsafe.Pointer(w.dnd)))
}

// revokeDropTarget unregisters the target and tears OLE down. Called from
// WM_DESTROY, on the same thread that registered it.
func (w *window) revokeDropTarget() {
	if w.dnd == nil {
		return
	}
	procRevokeDragDrop.Call(uintptr(w.hwnd))
	w.dnd = nil
	procOleUninitialize.Call()
}

func guidEqual(a *_GUID, b *_GUID) bool {
	if a.Data1 != b.Data1 || a.Data2 != b.Data2 || a.Data3 != b.Data3 {
		return false
	}
	for i := 0; i < 8; i++ {
		if a.Data4[i] != b.Data4[i] {
			return false
		}
	}
	return true
}

func dropQueryInterface(this, riid, ppv uintptr) uintptr {
	id := (*_GUID)(unsafe.Pointer(riid))
	if guidEqual(id, &iidIUnknown) || guidEqual(id, &iidIDropTarget) {
		*(*uintptr)(unsafe.Pointer(ppv)) = this
		return _S_OK
	}
	if ppv != 0 {
		*(*uintptr)(unsafe.Pointer(ppv)) = 0
	}
	return _E_NOINTERFACE
}

// The Go object's lifetime is owned by the window (w.dnd keeps it alive until
// revoke), so reference counting is a formality: report a steady single ref.
func dropAddRef(this uintptr) uintptr  { return 1 }
func dropRelease(this uintptr) uintptr { return 1 }

// decodePOINTL splits a by-value POINTL (passed as a single 8-byte register on
// amd64) into screen x/y.
func decodePOINTL(pt uintptr) (int32, int32) {
	return int32(uint32(pt)), int32(uint32(pt >> 32))
}

// clientPoint converts a screen-space POINTL to window-client coordinates.
func (w *window) clientPoint(pt uintptr) f32.Point {
	sx, sy := decodePOINTL(pt)
	p := windows.Point{X: sx, Y: sy}
	windows.ScreenToClient(w.hwnd, &p)
	return f32.Point{X: float32(p.X), Y: float32(p.Y)}
}

// dataHasFiles reports whether the drag payload offers the CF_HDROP (file list)
// format, via IDataObject::QueryGetData (vtable slot 5).
func dataHasFiles(pDataObj uintptr) bool {
	if pDataObj == 0 {
		return false
	}
	fmtetc := _FORMATETC{cfFormat: _CF_HDROP, dwAspect: _DVASPECT_CONTENT, lindex: -1, tymed: _TYMED_HGLOBAL}
	vtbl := *(*uintptr)(unsafe.Pointer(pDataObj))
	queryGetData := *(*uintptr)(unsafe.Pointer(vtbl + 5*unsafe.Sizeof(uintptr(0))))
	r, _, _ := stdsyscall.SyscallN(queryGetData, pDataObj, uintptr(unsafe.Pointer(&fmtetc)))
	return r == _S_OK
}

func dropDragEnter(this, pDataObj, grfKeyState, pt, pdwEffect uintptr) uintptr {
	dt := (*dropTarget)(unsafe.Pointer(this))
	files := dataHasFiles(pDataObj)
	setEffect(pdwEffect, files)
	if files {
		dt.w.ProcessEvent(DragFilesEvent{Position: dt.w.clientPoint(pt), Active: true})
	}
	return _S_OK
}

func dropDragOver(this, grfKeyState, pt, pdwEffect uintptr) uintptr {
	dt := (*dropTarget)(unsafe.Pointer(this))
	// pdwEffect is reset by the OS before each DragOver; honour file drags only.
	files := *(*uint32)(unsafe.Pointer(pdwEffect)) != 0
	setEffect(pdwEffect, files)
	if files {
		dt.w.ProcessEvent(DragFilesEvent{Position: dt.w.clientPoint(pt), Active: true})
	}
	return _S_OK
}

func dropDragLeave(this uintptr) uintptr {
	dt := (*dropTarget)(unsafe.Pointer(this))
	dt.w.ProcessEvent(DragFilesEvent{Active: false})
	return _S_OK
}

func dropDrop(this, pDataObj, grfKeyState, pt, pdwEffect uintptr) uintptr {
	dt := (*dropTarget)(unsafe.Pointer(this))
	pos := dt.w.clientPoint(pt)
	// A drag-leave is not delivered before a drop, so clear the highlight first.
	dt.w.ProcessEvent(DragFilesEvent{Active: false})
	paths := extractDropPaths(pDataObj)
	setEffect(pdwEffect, len(paths) > 0)
	if len(paths) > 0 {
		dt.w.ProcessEvent(DropFilesEvent{Paths: paths, Position: pos})
	}
	return _S_OK
}

func setEffect(pdwEffect uintptr, accept bool) {
	if pdwEffect == 0 {
		return
	}
	if accept {
		*(*uint32)(unsafe.Pointer(pdwEffect)) = _DROPEFFECT_COPY
	} else {
		*(*uint32)(unsafe.Pointer(pdwEffect)) = _DROPEFFECT_NONE
	}
}

// extractDropPaths pulls the CF_HDROP file list out of the drag's IDataObject
// (GetData → HGLOBAL HDROP → DragQueryFileW), then releases the medium.
func extractDropPaths(pDataObj uintptr) []string {
	if pDataObj == 0 {
		return nil
	}
	fmtetc := _FORMATETC{cfFormat: _CF_HDROP, dwAspect: _DVASPECT_CONTENT, lindex: -1, tymed: _TYMED_HGLOBAL}
	var medium _STGMEDIUM
	vtbl := *(*uintptr)(unsafe.Pointer(pDataObj))
	getData := *(*uintptr)(unsafe.Pointer(vtbl + 3*unsafe.Sizeof(uintptr(0))))
	r, _, _ := stdsyscall.SyscallN(getData, pDataObj, uintptr(unsafe.Pointer(&fmtetc)), uintptr(unsafe.Pointer(&medium)))
	if r != _S_OK || medium.unionData == 0 {
		return nil
	}
	defer procReleaseStgMedium.Call(uintptr(unsafe.Pointer(&medium)))
	return queryDropFiles(medium.unionData)
}

// queryDropFiles reads every path out of an HDROP handle.
func queryDropFiles(hDrop uintptr) []string {
	n, _, _ := procDragQueryFileW.Call(hDrop, 0xFFFFFFFF, 0, 0)
	count := int(n)
	if count <= 0 {
		return nil
	}
	paths := make([]string, 0, count)
	for i := 0; i < count; i++ {
		l, _, _ := procDragQueryFileW.Call(hDrop, uintptr(i), 0, 0)
		size := int(l) + 1
		if size <= 1 {
			continue
		}
		buf := make([]uint16, size)
		procDragQueryFileW.Call(hDrop, uintptr(i), uintptr(unsafe.Pointer(&buf[0])), uintptr(size))
		if p := syscall.UTF16ToString(buf); p != "" {
			paths = append(paths, p)
		}
	}
	return paths
}
