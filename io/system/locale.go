package system

type Locale struct {
	Language string

	Direction TextDirection
}

const (
	axisShift = iota
	progressionShift
)

type TextDirection byte

const (
	LTR TextDirection = TextDirection(Horizontal<<axisShift) | TextDirection(FromOrigin<<progressionShift)

	RTL TextDirection = TextDirection(Horizontal<<axisShift) | TextDirection(TowardOrigin<<progressionShift)
)

func (d TextDirection) Axis() TextAxis {
	return TextAxis((d & (1 << axisShift)) >> axisShift)
}

func (d TextDirection) Progression() TextProgression {
	return TextProgression((d & (1 << progressionShift)) >> progressionShift)
}

func (d TextDirection) String() string {
	switch d {
	case RTL:
		return "RTL"
	default:
		return "LTR"
	}
}

type TextAxis byte

const (
	Horizontal TextAxis = iota

	Vertical
)

type TextProgression byte

const (
	FromOrigin TextProgression = iota

	TowardOrigin
)
