package stroke

import (
	"encoding/binary"
	"math"

	"github.com/nanorele/gio/internal/f32"
	"github.com/nanorele/gio/internal/ops"
	"github.com/nanorele/gio/internal/scene"
)

type StrokeStyle struct {
	Width float32
}

const strokeTolerance = 0.01

type QuadSegment struct {
	From, Ctrl, To f32.Point
}

type StrokeQuad struct {
	Contour uint32
	Quad    QuadSegment
}

type strokeState struct {
	p0, p1 f32.Point
	n0, n1 f32.Point
	r0, r1 float32
	ctl    f32.Point
}

type StrokeQuads []StrokeQuad

func (qs *StrokeQuads) pen() f32.Point {
	return (*qs)[len(*qs)-1].Quad.To
}

func (qs *StrokeQuads) lineTo(pt f32.Point) {
	end := qs.pen()
	*qs = append(*qs, StrokeQuad{
		Quad: QuadSegment{
			From: end,
			Ctrl: end.Add(pt).Mul(0.5),
			To:   pt,
		},
	})
}

func (qs *StrokeQuads) arc(f1, f2 f32.Point, angle float32) {
	pen := qs.pen()
	m, segments := ArcTransform(pen, f1.Add(pen), f2.Add(pen), angle)
	for range segments {
		p0 := qs.pen()
		p1 := m.Transform(p0)
		p2 := m.Transform(p1)
		ctl := p1.Mul(2).Sub(p0.Add(p2).Mul(.5))
		*qs = append(*qs, StrokeQuad{
			Quad: QuadSegment{
				From: p0, Ctrl: ctl, To: p2,
			},
		})
	}
}

func (qs StrokeQuads) split() []StrokeQuads {
	if len(qs) == 0 {
		return nil
	}
	var o []StrokeQuads
	start := 0
	c := qs[0].Contour
	for i, q := range qs {
		if q.Contour != c {
			o = append(o, qs[start:i])
			c = q.Contour
			start = i
		}
	}
	o = append(o, qs[start:])
	return o
}

func (qs StrokeQuads) stroke(stroke StrokeStyle) StrokeQuads {
	var (
		o  StrokeQuads
		hw = 0.5 * stroke.Width
	)

	for _, ps := range qs.split() {
		rhs, lhs := ps.offset(hw, stroke)
		switch lhs {
		case nil:
			o = o.append(rhs)
		default:

			switch {
			case ps.ccw():
				lhs = lhs.reverse()
				o = o.append(rhs)
				o = o.append(lhs)
			default:
				rhs = rhs.reverse()
				o = o.append(lhs)
				o = o.append(rhs)
			}
		}
	}

	return o
}

func (qs StrokeQuads) offset(hw float32, stroke StrokeStyle) (rhs, lhs StrokeQuads) {
	var (
		states []strokeState
		beg    = qs[0].Quad.From
		end    = qs[len(qs)-1].Quad.To
		closed = beg == end
	)
	for i := range qs {
		q := qs[i].Quad

		var (
			n0 = strokePathNorm(q.From, q.Ctrl, q.To, 0, hw)
			n1 = strokePathNorm(q.From, q.Ctrl, q.To, 1, hw)
			r0 = strokePathCurv(q.From, q.Ctrl, q.To, 0)
			r1 = strokePathCurv(q.From, q.Ctrl, q.To, 1)
		)
		states = append(states, strokeState{
			p0:  q.From,
			p1:  q.To,
			n0:  n0,
			n1:  n1,
			r0:  r0,
			r1:  r1,
			ctl: q.Ctrl,
		})
	}

	for i, state := range states {
		rhs = rhs.append(strokeQuadBezier(state, +hw, strokeTolerance))
		lhs = lhs.append(strokeQuadBezier(state, -hw, strokeTolerance))

		if hasNext := i+1 < len(states); hasNext || closed {
			var next strokeState
			switch {
			case hasNext:
				next = states[i+1]
			case closed:
				next = states[0]
			}
			if state.n1 != next.n0 {
				strokePathRoundJoin(&rhs, &lhs, hw, state.p1, state.n1, next.n0, state.r1, next.r0)
			}
		}
	}

	if closed {
		rhs.close()
		lhs.close()
		return rhs, lhs
	}

	qbeg := &states[0]
	qend := &states[len(states)-1]

	lhs = lhs.reverse()
	strokePathCap(stroke, &rhs, hw, qend.p1, qend.n1)

	rhs = rhs.append(lhs)
	strokePathCap(stroke, &rhs, hw, qbeg.p0, qbeg.n0.Mul(-1))

	rhs.close()

	return rhs, nil
}

func (qs *StrokeQuads) close() {
	p0 := (*qs)[len(*qs)-1].Quad.To
	p1 := (*qs)[0].Quad.From

	if p1 == p0 {
		return
	}

	*qs = append(*qs, StrokeQuad{
		Quad: QuadSegment{
			From: p0,
			Ctrl: p0.Add(p1).Mul(0.5),
			To:   p1,
		},
	})
}

func (qs StrokeQuads) ccw() bool {

	var area float32
	for _, ps := range qs.split() {
		for i := 1; i < len(ps); i++ {
			pi := ps[i].Quad.To
			pj := ps[i-1].Quad.To
			area += (pi.X - pj.X) * (pi.Y + pj.Y)
		}
	}
	return area <= 0.0
}

func (qs StrokeQuads) reverse() StrokeQuads {
	if len(qs) == 0 {
		return nil
	}

	ps := make(StrokeQuads, 0, len(qs))
	for i := range qs {
		q := qs[len(qs)-1-i]
		q.Quad.To, q.Quad.From = q.Quad.From, q.Quad.To
		ps = append(ps, q)
	}

	return ps
}

func (qs StrokeQuads) append(ps StrokeQuads) StrokeQuads {
	switch {
	case len(ps) == 0:
		return qs
	case len(qs) == 0:
		return ps
	}

	p0 := qs[len(qs)-1].Quad.To
	p1 := ps[0].Quad.From
	if p0 != p1 && lenPt(p0.Sub(p1)) < strokeTolerance {
		qs = append(qs, StrokeQuad{
			Quad: QuadSegment{
				From: p0,
				Ctrl: p0.Add(p1).Mul(0.5),
				To:   p1,
			},
		})
	}
	return append(qs, ps...)
}

func (q QuadSegment) Transform(t f32.Affine2D) QuadSegment {
	q.From = t.Transform(q.From)
	q.Ctrl = t.Transform(q.Ctrl)
	q.To = t.Transform(q.To)
	return q
}

func strokePathNorm(p0, p1, p2 f32.Point, t, d float32) f32.Point {
	switch t {
	case 0:
		n := p1.Sub(p0)
		if n.X == 0 && n.Y == 0 {
			return f32.Point{}
		}
		n = rot90CW(n)
		return normPt(n, d)
	case 1:
		n := p2.Sub(p1)
		if n.X == 0 && n.Y == 0 {
			return f32.Point{}
		}
		n = rot90CW(n)
		return normPt(n, d)
	}
	panic("impossible")
}

func rot90CW(p f32.Point) f32.Point { return f32.Pt(+p.Y, -p.X) }

func normPt(p f32.Point, l float32) f32.Point {
	if (p.X == 0 && p.Y == 0) || l == 0 {
		return f32.Point{}
	}
	isVerticalUnit := p.X == 0 && (p.Y == l || p.Y == -l)
	isHorizontalUnit := p.Y == 0 && (p.X == l || p.X == -l)
	if isVerticalUnit || isHorizontalUnit {
		if math.Signbit(float64(l)) {
			return f32.Point{X: -p.X, Y: -p.Y}
		} else {
			return f32.Point{X: p.X, Y: p.Y}
		}
	}
	d := math.Hypot(float64(p.X), float64(p.Y))
	l64 := float64(l)
	if math.Abs(d-l64) < 1e-10 {
		if math.Signbit(float64(l)) {
			return f32.Point{X: -p.X, Y: -p.Y}
		} else {
			return f32.Point{X: p.X, Y: p.Y}
		}
	}
	n := float32(l64 / d)
	return f32.Point{X: p.X * n, Y: p.Y * n}
}

func lenPt(p f32.Point) float32 {
	return float32(math.Hypot(float64(p.X), float64(p.Y)))
}

func perpDot(p, q f32.Point) float32 {
	return p.X*q.Y - p.Y*q.X
}

func angleBetween(n0, n1 f32.Point) float64 {
	return math.Atan2(float64(n1.Y), float64(n1.X)) -
		math.Atan2(float64(n0.Y), float64(n0.X))
}

func strokePathCurv(beg, ctl, end f32.Point, t float32) float32 {
	var (
		d1p = quadBezierD1(beg, ctl, end, t)
		d2p = quadBezierD2(beg, ctl, end, t)

		a = float64(perpDot(d1p, d2p))
	)

	if math.Abs(a) < 1e-10 {
		return float32(math.NaN())
	}
	return float32(math.Pow(float64(d1p.X*d1p.X+d1p.Y*d1p.Y), 1.5) / a)
}

func quadBezierSample(p0, p1, p2 f32.Point, t float32) f32.Point {
	t1 := 1 - t
	c0 := t1 * t1
	c1 := 2 * t1 * t
	c2 := t * t

	o := p0.Mul(c0)
	o = o.Add(p1.Mul(c1))
	o = o.Add(p2.Mul(c2))
	return o
}

func quadBezierD1(p0, p1, p2 f32.Point, t float32) f32.Point {
	p10 := p1.Sub(p0).Mul(2 * (1 - t))
	p21 := p2.Sub(p1).Mul(2 * t)

	return p10.Add(p21)
}

func quadBezierD2(p0, p1, p2 f32.Point, t float32) f32.Point {
	p := p2.Sub(p1.Mul(2)).Add(p0)
	return p.Mul(2)
}

func strokeQuadBezier(state strokeState, d, flatness float32) StrokeQuads {

	var qs StrokeQuads
	return flattenQuadBezier(qs, state.p0, state.ctl, state.p1, d, flatness)
}

func flattenQuadBezier(qs StrokeQuads, p0, p1, p2 f32.Point, d, flatness float32) StrokeQuads {
	var (
		t      float32
		flat64 = float64(flatness)
	)
	for t < 1 {
		s2 := float64((p2.X-p0.X)*(p1.Y-p0.Y) - (p2.Y-p0.Y)*(p1.X-p0.X))
		den := math.Hypot(float64(p1.X-p0.X), float64(p1.Y-p0.Y))
		if s2*den == 0.0 {
			break
		}

		s2 /= den
		t = 2.0 * float32(math.Sqrt(flat64/3.0/math.Abs(s2)))
		if t >= 1.0 {
			break
		}
		var q0, q1, q2 f32.Point
		q0, q1, q2, p0, p1, p2 = quadBezierSplit(p0, p1, p2, t)
		qs.addLine(q0, q1, q2, 0, d)
	}
	qs.addLine(p0, p1, p2, 1, d)
	return qs
}

func (qs *StrokeQuads) addLine(p0, ctrl, p1 f32.Point, t, d float32) {
	switch i := len(*qs); i {
	case 0:
		p0 = p0.Add(strokePathNorm(p0, ctrl, p1, 0, d))
	default:

		p0 = (*qs)[i-1].Quad.To
	}

	p1 = p1.Add(strokePathNorm(p0, ctrl, p1, 1, d))

	*qs = append(*qs,
		StrokeQuad{
			Quad: QuadSegment{
				From: p0,
				Ctrl: p0.Add(p1).Mul(0.5),
				To:   p1,
			},
		},
	)
}

func quadInterp(p, q f32.Point, t float32) f32.Point {
	return f32.Pt(
		(1-t)*p.X+t*q.X,
		(1-t)*p.Y+t*q.Y,
	)
}

func quadBezierSplit(p0, p1, p2 f32.Point, t float32) (f32.Point, f32.Point, f32.Point, f32.Point, f32.Point, f32.Point) {
	var (
		b0 = p0
		b1 = quadInterp(p0, p1, t)
		b2 = quadBezierSample(p0, p1, p2, t)

		a0 = b2
		a1 = quadInterp(p1, p2, t)
		a2 = p2
	)

	return b0, b1, b2, a0, a1, a2
}

func strokePathRoundJoin(rhs, lhs *StrokeQuads, hw float32, pivot, n0, n1 f32.Point, r0, r1 float32) {
	rp := pivot.Add(n1)
	lp := pivot.Sub(n1)
	angle := angleBetween(n0, n1)
	switch {
	case angle <= 0:

		c := pivot.Sub(lhs.pen())
		lhs.arc(c, c, float32(angle))
		lhs.lineTo(lp)
		rhs.lineTo(rp)
	default:

		c := pivot.Sub(rhs.pen())
		rhs.arc(c, c, float32(angle))
		rhs.lineTo(rp)
		lhs.lineTo(lp)
	}
}

func strokePathCap(stroke StrokeStyle, qs *StrokeQuads, hw float32, pivot, n0 f32.Point) {
	strokePathRoundCap(qs, hw, pivot, n0)
}

func strokePathRoundCap(qs *StrokeQuads, hw float32, pivot, n0 f32.Point) {
	c := pivot.Sub(qs.pen())
	qs.arc(c, c, math.Pi)
}

func ArcTransform(p, f1, f2 f32.Point, angle float32) (transform f32.Affine2D, segments int) {
	const segmentsPerCircle = 16
	const anglePerSegment = 2 * math.Pi / segmentsPerCircle

	s := angle / anglePerSegment
	if s < 0 {
		s = -s
	}
	segments = int(math.Ceil(float64(s)))
	if segments <= 0 {
		segments = 1
	}

	var rx, ry, alpha float64
	if f1 == f2 {

		rx = dist(f1, p)
		ry = rx
	} else {

		a := 0.5 * (dist(f1, p) + dist(f2, p))

		c := dist(f1, f2) * 0.5
		b := math.Sqrt(a*a - c*c)
		switch {
		case a > b:
			rx = a
			ry = b
		default:
			rx = b
			ry = a
		}
		if f1.X == f2.X {

			alpha = math.Pi / 2
			if f1.Y < f2.Y {
				alpha = -alpha
			}
		} else {
			x := float64(f1.X-f2.X) * 0.5
			if x < 0 {
				x = -x
			}
			alpha = math.Acos(x / c)
		}
	}

	θ := angle / float32(segments)
	ref := f32.AffineId()
	rot := f32.AffineId()
	inv := f32.AffineId()
	center := f32.Point{
		X: 0.5 * (f1.X + f2.X),
		Y: 0.5 * (f1.Y + f2.Y),
	}
	ref = ref.Offset(f32.Point{}.Sub(center))
	ref = ref.Rotate(f32.Point{}, float32(-alpha))
	ref = ref.Scale(f32.Point{}, f32.Point{
		X: float32(1 / rx),
		Y: float32(1 / ry),
	})
	inv = ref.Invert()
	rot = rot.Rotate(f32.Point{}, 0.5*θ)

	return inv.Mul(rot).Mul(ref), segments
}

func dist(p1, p2 f32.Point) float64 {
	var (
		x1 = float64(p1.X)
		y1 = float64(p1.Y)
		x2 = float64(p2.X)
		y2 = float64(p2.Y)
		dx = x2 - x1
		dy = y2 - y1
	)
	return math.Hypot(dx, dy)
}

func StrokePathCommands(style StrokeStyle, scene []byte) StrokeQuads {
	quads := decodeToStrokeQuads(scene)
	return quads.stroke(style)
}

func decodeToStrokeQuads(pathData []byte) StrokeQuads {
	quads := make(StrokeQuads, 0, 2*len(pathData)/(scene.CommandSize+4))
	scratch := make([]QuadSegment, 0, 10)
	for len(pathData) >= scene.CommandSize+4 {
		contour := binary.LittleEndian.Uint32(pathData)
		cmd := ops.DecodeCommand(pathData[4:])
		switch cmd.Op() {
		case scene.OpLine:
			var q QuadSegment
			q.From, q.To = scene.DecodeLine(cmd)
			q.Ctrl = q.From.Add(q.To).Mul(.5)
			quad := StrokeQuad{
				Contour: contour,
				Quad:    q,
			}
			quads = append(quads, quad)
		case scene.OpGap:

		case scene.OpQuad:
			var q QuadSegment
			q.From, q.Ctrl, q.To = scene.DecodeQuad(cmd)
			quad := StrokeQuad{
				Contour: contour,
				Quad:    q,
			}
			quads = append(quads, quad)
		case scene.OpCubic:
			from, ctrl0, ctrl1, to := scene.DecodeCubic(cmd)
			scratch = SplitCubic(from, ctrl0, ctrl1, to, scratch[:0])
			for _, q := range scratch {
				quad := StrokeQuad{
					Contour: contour,
					Quad:    q,
				}
				quads = append(quads, quad)
			}
		default:
			panic("unsupported scene command")
		}
		pathData = pathData[scene.CommandSize+4:]
	}
	return quads
}

func SplitCubic(from, ctrl0, ctrl1, to f32.Point, quads []QuadSegment) []QuadSegment {

	hull := f32.Rectangle{
		Min: from,
		Max: ctrl0,
	}.Canon().Union(f32.Rectangle{
		Min: ctrl1,
		Max: to,
	}.Canon())
	l := hull.Dx()
	if h := hull.Dy(); h > l {
		l = h
	}
	maxDist := l * 0.001
	approxCubeTo(&quads, 0, maxDist*maxDist, from, ctrl0, ctrl1, to)
	return quads
}

func approxCubeTo(quads *[]QuadSegment, splits int, maxDistSq float32, from, ctrl0, ctrl1, to f32.Point) int {

	q0 := ctrl0.Mul(3).Sub(from)
	q1 := ctrl1.Mul(3).Sub(to)
	c := q0.Add(q1).Mul(1.0 / 4.0)
	const maxSplits = 32
	if splits >= maxSplits {
		*quads = append(*quads, QuadSegment{From: from, Ctrl: c, To: to})
		return splits
	}

	v := q0.Sub(q1)
	d2 := (v.X*v.X + v.Y*v.Y) * 3 / (36 * 36)
	if d2 <= maxDistSq {
		*quads = append(*quads, QuadSegment{From: from, Ctrl: c, To: to})
		return splits
	}

	t := float32(0.5)
	c0 := from.Add(ctrl0.Sub(from).Mul(t))
	c1 := ctrl0.Add(ctrl1.Sub(ctrl0).Mul(t))
	c2 := ctrl1.Add(to.Sub(ctrl1).Mul(t))
	c01 := c0.Add(c1.Sub(c0).Mul(t))
	c12 := c1.Add(c2.Sub(c1).Mul(t))
	c0112 := c01.Add(c12.Sub(c01).Mul(t))
	splits++
	splits = approxCubeTo(quads, splits, maxDistSq, from, c0, c01, c0112)
	splits = approxCubeTo(quads, splits, maxDistSq, c0112, c12, c2, to)
	return splits
}
