package widget

import (
	"image"
	"image/color"
	"testing"

	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/unit"

	"golang.org/x/exp/shiny/materialdesign/icons"
)

func TestNewIcon_Error(t *testing.T) {
	_, err := NewIcon([]byte("invalid"))
	if err == nil {
		t.Error("expected error for invalid icon data")
	}
}

func TestIcon_Alpha(t *testing.T) {
	icon, err := NewIcon(icons.ToggleCheckBox)
	if err != nil {
		t.Fatal(err)
	}

	col := color.NRGBA{B: 0xff, A: 0x40}

	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(100, 100)),
	}

	_ = icon.Layout(gtx, col)
}

func TestWidgetConstraints(t *testing.T) {
	_cs := func(v ...layout.Constraints) []layout.Constraints { return v }
	for _, tc := range []struct {
		label       string
		widget      layout.Widget
		constraints []layout.Constraints
	}{
		{
			label: "Icon",
			widget: func(gtx layout.Context) layout.Dimensions {
				ic, _ := NewIcon(icons.ToggleCheckBox)
				return ic.Layout(gtx, color.NRGBA{A: 0xff})
			},
			constraints: _cs(
				layout.Constraints{
					Min: image.Pt(20, 0),
					Max: image.Pt(100, 100),
				},
				layout.Constraints{
					Max: image.Pt(100, 100),
				},
			),
		},
	} {
		t.Run(tc.label, func(t *testing.T) {
			for _, cs := range tc.constraints {
				gtx := layout.Context{
					Constraints: cs,
					Ops:         new(op.Ops),
				}
				dims := tc.widget(gtx)
				csr := image.Rectangle{
					Min: cs.Min,
					Max: cs.Max,
				}
				if !dims.Size.In(csr) {
					t.Errorf("dims size %v not within constraints %v", dims.Size, csr)
				}
			}
		})
	}
}

// TestNewIconNil ensures NewIcon returns an error (not panic) for nil input.
func TestNewIconNil(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NewIcon(nil) panicked: %v", r)
		}
	}()
	if _, err := NewIcon(nil); err == nil {
		t.Error("NewIcon(nil) returned no error")
	}
}

// TestNewIconEmpty ensures NewIcon returns an error for empty input.
func TestNewIconEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NewIcon([]byte{}) panicked: %v", r)
		}
	}()
	if _, err := NewIcon([]byte{}); err == nil {
		t.Error("NewIcon([]byte{}) returned no error")
	}
}

// TestIconDefaultSize ensures that with Min=0 the icon falls back to
// defaultIconSize (24dp).
func TestIconDefaultSize(t *testing.T) {
	icon, err := NewIcon(icons.ToggleCheckBox)
	if err != nil {
		t.Fatal(err)
	}
	gtx := layout.Context{
		Ops:    new(op.Ops),
		Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{
			Min: image.Pt(0, 0),
			Max: image.Pt(200, 200),
		},
	}
	dims := icon.Layout(gtx, color.NRGBA{R: 0xff, A: 0xff})
	want := gtx.Dp(unit.Dp(24))
	if dims.Size.X != want {
		t.Errorf("default size: got X=%d, want %d", dims.Size.X, want)
	}
}

// TestIconRespectsMin verifies that a non-zero Min size is honored.
func TestIconRespectsMin(t *testing.T) {
	icon, err := NewIcon(icons.ToggleCheckBox)
	if err != nil {
		t.Fatal(err)
	}
	gtx := layout.Context{
		Ops:    new(op.Ops),
		Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{
			Min: image.Pt(40, 40),
			Max: image.Pt(200, 200),
		},
	}
	dims := icon.Layout(gtx, color.NRGBA{A: 0xff})
	if dims.Size.X < 40 {
		t.Errorf("icon ignored Min: got %v", dims.Size)
	}
}

// TestIconClampedToMax verifies that the icon size is constrained to Max.
func TestIconClampedToMax(t *testing.T) {
	icon, err := NewIcon(icons.ToggleCheckBox)
	if err != nil {
		t.Fatal(err)
	}
	gtx := layout.Context{
		Ops:    new(op.Ops),
		Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{
			Min: image.Pt(500, 500),
			Max: image.Pt(50, 50),
		},
	}
	dims := icon.Layout(gtx, color.NRGBA{A: 0xff})
	if dims.Size.X > 50 || dims.Size.Y > 50 {
		t.Errorf("icon exceeded Max: got %v, max=50x50", dims.Size)
	}
}

// TestIconColorCacheReuse verifies that successive calls with the same size
// and color reuse the cached image (no panic, returns valid op).
func TestIconColorCacheReuse(t *testing.T) {
	icon, err := NewIcon(icons.ToggleCheckBox)
	if err != nil {
		t.Fatal(err)
	}
	gtx := layout.Context{
		Ops:    new(op.Ops),
		Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{
			Min: image.Pt(32, 32),
			Max: image.Pt(32, 32),
		},
	}
	col := color.NRGBA{R: 0x80, G: 0x00, B: 0x00, A: 0xff}
	d1 := icon.Layout(gtx, col)
	gtx.Ops.Reset()
	d2 := icon.Layout(gtx, col)
	if d1.Size != d2.Size {
		t.Errorf("cached layout differs: %v vs %v", d1.Size, d2.Size)
	}
}

// TestIconColorChangeInvalidatesCache exercises the cache miss path when
// only the color changes.
func TestIconColorChangeInvalidatesCache(t *testing.T) {
	icon, err := NewIcon(icons.ToggleCheckBox)
	if err != nil {
		t.Fatal(err)
	}
	gtx := layout.Context{
		Ops:    new(op.Ops),
		Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{
			Min: image.Pt(32, 32),
			Max: image.Pt(32, 32),
		},
	}
	icon.Layout(gtx, color.NRGBA{R: 0xff, A: 0xff})
	gtx.Ops.Reset()
	icon.Layout(gtx, color.NRGBA{B: 0xff, A: 0xff})
	if icon.imgColor.B != 0xff {
		t.Errorf("color cache not refreshed on color change: imgColor=%v", icon.imgColor)
	}
}

// TestIconTransparentColor verifies that a fully-transparent color renders
// without error (alpha == 0 path).
func TestIconTransparentColor(t *testing.T) {
	icon, err := NewIcon(icons.ToggleCheckBox)
	if err != nil {
		t.Fatal(err)
	}
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(20, 20)),
	}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("transparent icon layout panicked: %v", r)
		}
	}()
	_ = icon.Layout(gtx, color.NRGBA{})
}

// TestIconTinyConstraint covers the smallest non-degenerate constraint.
func TestIconTinyConstraint(t *testing.T) {
	icon, err := NewIcon(icons.ToggleCheckBox)
	if err != nil {
		t.Fatal(err)
	}
	gtx := layout.Context{
		Ops:    new(op.Ops),
		Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{
			Min: image.Pt(1, 1),
			Max: image.Pt(1, 1),
		},
	}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("1x1 icon layout panicked: %v", r)
		}
	}()
	dims := icon.Layout(gtx, color.NRGBA{A: 0xff})
	if dims.Size.X != 1 || dims.Size.Y != 1 {
		t.Errorf("1x1 icon: got %v, want 1x1", dims.Size)
	}
}

// TestIconHiDPIDefaultSize verifies default size scales with PxPerDp.
func TestIconHiDPIDefaultSize(t *testing.T) {
	icon, err := NewIcon(icons.ToggleCheckBox)
	if err != nil {
		t.Fatal(err)
	}
	gtx := layout.Context{
		Ops:    new(op.Ops),
		Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2},
		Constraints: layout.Constraints{
			Min: image.Pt(0, 0),
			Max: image.Pt(500, 500),
		},
	}
	dims := icon.Layout(gtx, color.NRGBA{A: 0xff})
	want := gtx.Dp(unit.Dp(24))
	if dims.Size.X != want {
		t.Errorf("HiDPI default: got X=%d, want %d", dims.Size.X, want)
	}
}
