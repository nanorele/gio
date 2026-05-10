package widget

import (
	"image"
	"testing"

	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/op/paint"
)

func TestImageScale(t *testing.T) {
	var ops op.Ops
	gtx := layout.Context{
		Ops: &ops,
		Constraints: layout.Constraints{
			Max: image.Pt(50, 50),
		},
	}
	imgSize := image.Pt(10, 10)
	img := image.NewNRGBA(image.Rectangle{Max: imgSize})
	imgOp := paint.NewImageOp(img)

	dims := Image{Src: imgOp}.Layout(gtx)
	expectedSize := imgSize
	expectedSize.X = int(float32(expectedSize.X))
	expectedSize.Y = int(float32(expectedSize.Y))
	if dims.Size != expectedSize {
		t.Fatalf("non-scaled image is wrong size, expected %v, got %v", expectedSize, dims.Size)
	}

	currentScale := float32(0.5)
	dims = Image{Src: imgOp, Scale: float32(currentScale)}.Layout(gtx)
	expectedSize = imgSize
	expectedSize.X = int(float32(expectedSize.X) * currentScale)
	expectedSize.Y = int(float32(expectedSize.Y) * currentScale)
	if dims.Size != expectedSize {
		t.Fatalf(".5 scale image is wrong size, expected %v, got %v", expectedSize, dims.Size)
	}

	currentScale = float32(1)
	gtx.Metric.PxPerDp = 2
	dims = Image{Src: imgOp, Scale: float32(currentScale)}.Layout(gtx)
	expectedSize = imgSize
	expectedSize.X = int(float32(expectedSize.X) * currentScale * gtx.Metric.PxPerDp)
	expectedSize.Y = int(float32(expectedSize.Y) * currentScale * gtx.Metric.PxPerDp)
	if dims.Size != expectedSize {
		t.Fatalf("HiDPI non-scaled image is wrong size, expected %v, got %v", expectedSize, dims.Size)
	}

	currentScale = float32(.5)
	gtx.Metric.PxPerDp = 2
	dims = Image{Src: imgOp, Scale: float32(currentScale)}.Layout(gtx)
	expectedSize = imgSize
	expectedSize.X = int(float32(expectedSize.X) * currentScale * gtx.Metric.PxPerDp)
	expectedSize.Y = int(float32(expectedSize.Y) * currentScale * gtx.Metric.PxPerDp)
	if dims.Size != expectedSize {
		t.Fatalf("HiDPI .5 scale image is wrong size, expected %v, got %v", expectedSize, dims.Size)
	}
}

// makeImageOp builds a paint.ImageOp of the given size for tests.
func makeImageOp(w, h int) paint.ImageOp {
	if w <= 0 || h <= 0 {
		return paint.ImageOp{}
	}
	return paint.NewImageOp(image.NewNRGBA(image.Rect(0, 0, w, h)))
}

// TestImageZeroSizedSrc verifies that an image with no source pixels does not
// panic and produces a non-negative size.
func TestImageZeroSizedSrc(t *testing.T) {
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Constraints{Max: image.Pt(100, 100)},
	}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("zero-sized src panicked: %v", r)
		}
	}()
	dims := Image{Src: paint.ImageOp{}}.Layout(gtx)
	if dims.Size.X < 0 || dims.Size.Y < 0 {
		t.Errorf("negative size: %v", dims.Size)
	}
}

// TestImageFitContainPreservesAspect verifies aspect preservation when
// scaling a wide image into a square.
func TestImageFitContainPreservesAspect(t *testing.T) {
	src := makeImageOp(200, 100) // 2:1
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Constraints{Max: image.Pt(100, 100)},
	}
	dims := Image{Src: src, Fit: Contain}.Layout(gtx)
	if dims.Size.X != 100 {
		t.Errorf("Contain wide image: width %d, want 100", dims.Size.X)
	}
	if dims.Size.Y != 50 {
		t.Errorf("Contain wide image: height %d, want 50 (aspect 2:1)", dims.Size.Y)
	}
}

// TestImageFitCoverFillsAxis verifies Cover scales so the image covers both
// axes, possibly cropping one dimension.
func TestImageFitCoverFillsAxis(t *testing.T) {
	src := makeImageOp(200, 100) // 2:1
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Constraints{Max: image.Pt(100, 100)},
	}
	dims := Image{Src: src, Fit: Cover}.Layout(gtx)
	// Cover scales to fill the larger dimension; final size is constrained
	// to the constraints, so both should equal 100.
	if dims.Size.X != 100 || dims.Size.Y != 100 {
		t.Errorf("Cover: got %v, want 100x100", dims.Size)
	}
}

// TestImageFitFillStretches verifies Fill stretches both axes.
func TestImageFitFillStretches(t *testing.T) {
	src := makeImageOp(10, 100)
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Constraints{Max: image.Pt(100, 100)},
	}
	dims := Image{Src: src, Fit: Fill}.Layout(gtx)
	if dims.Size.X != 100 || dims.Size.Y != 100 {
		t.Errorf("Fill: got %v, want 100x100", dims.Size)
	}
}

// TestImageFitScaleDownNoUpscale verifies ScaleDown leaves a small image
// at its native size when it already fits.
func TestImageFitScaleDownNoUpscale(t *testing.T) {
	src := makeImageOp(20, 20)
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Constraints{Max: image.Pt(100, 100)},
	}
	dims := Image{Src: src, Fit: ScaleDown}.Layout(gtx)
	if dims.Size.X != 20 || dims.Size.Y != 20 {
		t.Errorf("ScaleDown small image: got %v, want 20x20", dims.Size)
	}
}

// TestImageFitScaleDownLargeShrinks verifies that a big image is scaled down.
func TestImageFitScaleDownLargeShrinks(t *testing.T) {
	src := makeImageOp(400, 200) // 2:1
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Constraints{Max: image.Pt(100, 100)},
	}
	dims := Image{Src: src, Fit: ScaleDown}.Layout(gtx)
	if dims.Size.X != 100 || dims.Size.Y != 50 {
		t.Errorf("ScaleDown large image: got %v, want 100x50", dims.Size)
	}
}

// TestImageFitUnscaled verifies the default Fit returns native size, even if
// it overflows the constraints (clamped to Max).
func TestImageFitUnscaled(t *testing.T) {
	src := makeImageOp(200, 200)
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Constraints{Max: image.Pt(100, 100)},
	}
	dims := Image{Src: src, Fit: Unscaled}.Layout(gtx)
	if dims.Size.X != 100 || dims.Size.Y != 100 {
		t.Errorf("Unscaled clamps to Max: got %v, want 100x100", dims.Size)
	}
}

// TestImageFitContainTallSrc covers the orthogonal aspect (tall, not wide).
func TestImageFitContainTallSrc(t *testing.T) {
	src := makeImageOp(100, 400) // 1:4
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Constraints{Max: image.Pt(100, 100)},
	}
	dims := Image{Src: src, Fit: Contain}.Layout(gtx)
	if dims.Size.Y != 100 {
		t.Errorf("Contain tall: height %d, want 100", dims.Size.Y)
	}
	if dims.Size.X != 25 {
		t.Errorf("Contain tall: width %d, want 25 (aspect 1:4)", dims.Size.X)
	}
}

// TestImageScaleZeroDefaults verifies Scale==0 is treated as 1.
func TestImageScaleZeroDefaults(t *testing.T) {
	src := makeImageOp(20, 30)
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Constraints{Max: image.Pt(100, 100)},
	}
	dims := Image{Src: src, Scale: 0}.Layout(gtx)
	if dims.Size.X != 20 || dims.Size.Y != 30 {
		t.Errorf("Scale==0 fallback: got %v, want 20x30", dims.Size)
	}
}

// TestImageHugeSrcSmallConstraints stresses a very large image down to
// minimal pixels.
func TestImageHugeSrcSmallConstraints(t *testing.T) {
	src := makeImageOp(10000, 5000)
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Constraints{Max: image.Pt(10, 10)},
	}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("huge src small constraints panicked: %v", r)
		}
	}()
	dims := Image{Src: src, Fit: Contain}.Layout(gtx)
	if dims.Size.X != 10 {
		t.Errorf("Contain huge: width %d, want 10", dims.Size.X)
	}
	if dims.Size.Y < 1 || dims.Size.Y > 10 {
		t.Errorf("Contain huge: height %d out of [1,10]", dims.Size.Y)
	}
}

// TestImageFitWithAllPositions ensures every Position produces a valid size
// (non-negative, within constraints).
func TestImageFitWithAllPositions(t *testing.T) {
	src := makeImageOp(50, 25)
	positions := []layout.Direction{
		layout.NW, layout.N, layout.NE,
		layout.W, layout.Center, layout.E,
		layout.SW, layout.S, layout.SE,
	}
	for _, fit := range []Fit{Unscaled, Contain, Cover, ScaleDown, Fill} {
		for _, pos := range positions {
			gtx := layout.Context{
				Ops:         new(op.Ops),
				Constraints: layout.Constraints{Max: image.Pt(100, 100)},
			}
			dims := Image{Src: src, Fit: fit, Position: pos}.Layout(gtx)
			if dims.Size.X < 0 || dims.Size.Y < 0 {
				t.Errorf("fit=%v pos=%v produced negative size %v", fit, pos, dims.Size)
			}
			if dims.Size.X > 100 || dims.Size.Y > 100 {
				t.Errorf("fit=%v pos=%v exceeds max: %v", fit, pos, dims.Size)
			}
		}
	}
}
