package fling

import (
	"math"
	"strconv"
	"strings"
	"time"
)

type Extrapolation struct {
	idx int

	samples   []sample
	lastValue float32

	cache [historySize]sample

	values [historySize]float32
	times  [historySize]float32
}

type sample struct {
	t time.Duration
	v float32
}

type matrix struct {
	rows, cols int
	data       []float32
}

type Estimate struct {
	Velocity float32
	Distance float32
}

type coefficients [degree + 1]float32

const (
	degree       = 2
	historySize  = 20
	maxAge       = 100 * time.Millisecond
	maxSampleGap = 40 * time.Millisecond
)

func (e *Extrapolation) SampleDelta(t time.Duration, delta float32) {
	val := delta + e.lastValue
	e.Sample(t, val)
}

func (e *Extrapolation) Sample(t time.Duration, val float32) {
	e.lastValue = val
	if e.samples == nil {
		e.samples = e.cache[:0]
	}
	s := sample{
		t: t,
		v: val,
	}
	if e.idx == len(e.samples) && e.idx < cap(e.samples) {
		e.samples = append(e.samples, s)
	} else {
		e.samples[e.idx] = s
	}
	e.idx++
	if e.idx == cap(e.samples) {
		e.idx = 0
	}
}

func (e *Extrapolation) Estimate() Estimate {
	if len(e.samples) == 0 {
		return Estimate{}
	}
	values := e.values[:0]
	times := e.times[:0]
	first := e.get(0)
	t := first.t

	for i := range e.samples {
		p := e.get(-i)
		age := first.t - p.t
		if age >= maxAge || t-p.t >= maxSampleGap {

			break
		}
		t = p.t
		values = append(values, first.v-p.v)
		times = append(times, float32((-age).Seconds()))
	}
	coef, ok := polyFit(times, values)
	if !ok {
		return Estimate{}
	}
	dist := values[len(values)-1] - values[0]
	return Estimate{
		Velocity: coef[1],
		Distance: dist,
	}
}

func (e *Extrapolation) get(i int) sample {
	idx := (e.idx + i - 1 + len(e.samples)) % len(e.samples)
	return e.samples[idx]
}

func polyFit(X, Y []float32) (coefficients, bool) {
	if len(X) != len(Y) {
		panic("X and Y lengths differ")
	}
	if len(X) <= degree {

		return coefficients{}, false
	}

	A := newMatrix(degree+1, len(X))
	for i, x := range X {
		A.set(0, i, 1)
		for j := 1; j < A.rows; j++ {
			A.set(j, i, A.get(j-1, i)*x)
		}
	}

	Q, Rt, ok := decomposeQR(A)
	if !ok {
		return coefficients{}, false
	}

	var B coefficients
	for i := Q.rows - 1; i >= 0; i-- {
		B[i] = dot(Q.col(i), Y)
		for j := Q.rows - 1; j > i; j-- {
			B[i] -= Rt.get(i, j) * B[j]
		}
		B[i] /= Rt.get(i, i)
	}
	return B, true
}

func decomposeQR(A *matrix) (*matrix, *matrix, bool) {

	Q := newMatrix(A.rows, A.cols)
	Rt := newMatrix(A.rows, A.rows)
	for i := range Q.rows {

		for j := range Q.cols {
			Q.set(i, j, A.get(i, j))
		}

		for j := range i {
			d := dot(Q.col(j), Q.col(i))
			for k := range Q.cols {
				Q.set(i, k, Q.get(i, k)-d*Q.get(j, k))
			}
		}

		n := norm(Q.col(i))
		if n < 0.000001 {

			return nil, nil, false
		}
		invNorm := 1 / n
		for j := range Q.cols {
			Q.set(i, j, Q.get(i, j)*invNorm)
		}

		for j := i; j < Rt.cols; j++ {
			Rt.set(i, j, dot(Q.col(i), A.col(j)))
		}
	}
	return Q, Rt, true
}

func norm(V []float32) float32 {
	var n float32
	for _, v := range V {
		n += v * v
	}
	return float32(math.Sqrt(float64(n)))
}

func dot(V1, V2 []float32) float32 {
	var d float32
	for i, v1 := range V1 {
		d += v1 * V2[i]
	}
	return d
}

func newMatrix(rows, cols int) *matrix {
	return &matrix{
		rows: rows,
		cols: cols,
		data: make([]float32, rows*cols),
	}
}

func (m *matrix) set(row, col int, v float32) {
	if row < 0 || row >= m.rows {
		panic("row out of range")
	}
	if col < 0 || col >= m.cols {
		panic("col out of range")
	}
	m.data[row*m.cols+col] = v
}

func (m *matrix) get(row, col int) float32 {
	if row < 0 || row >= m.rows {
		panic("row out of range")
	}
	if col < 0 || col >= m.cols {
		panic("col out of range")
	}
	return m.data[row*m.cols+col]
}

func (m *matrix) col(c int) []float32 {
	return m.data[c*m.cols : (c+1)*m.cols]
}

func (m *matrix) approxEqual(m2 *matrix) bool {
	if m.rows != m2.rows || m.cols != m2.cols {
		return false
	}
	const epsilon = 0.00001
	for row := range m.rows {
		for col := range m.cols {
			d := m2.get(row, col) - m.get(row, col)
			if d < -epsilon || d > epsilon {
				return false
			}
		}
	}
	return true
}

func (m *matrix) transpose() *matrix {
	t := &matrix{
		rows: m.cols,
		cols: m.rows,
		data: make([]float32, len(m.data)),
	}
	for i := range m.rows {
		for j := range m.cols {
			t.set(j, i, m.get(i, j))
		}
	}
	return t
}

func (m *matrix) mul(m2 *matrix) *matrix {
	if m.rows != m2.cols {
		panic("mismatched matrices")
	}
	mm := &matrix{
		rows: m.rows,
		cols: m2.cols,
		data: make([]float32, m.rows*m2.cols),
	}
	for i := range mm.rows {
		for j := range mm.cols {
			var v float32
			for k := range m.rows {
				v += m.get(k, j) * m2.get(i, k)
			}
			mm.set(i, j, v)
		}
	}
	return mm
}

func (m *matrix) String() string {
	var b strings.Builder
	for i := range m.rows {
		for j := range m.cols {
			v := m.get(i, j)
			b.WriteString(strconv.FormatFloat(float64(v), 'g', -1, 32))
			b.WriteString(", ")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (c coefficients) approxEqual(c2 coefficients) bool {
	const epsilon = 0.00001
	for i, v := range c {
		d := v - c2[i]
		if d < -epsilon || d > epsilon {
			return false
		}
	}
	return true
}
