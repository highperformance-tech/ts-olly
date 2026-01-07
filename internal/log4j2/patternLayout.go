package log4j2

type PatternLayout struct {
	pattern string
}

func NewPatternLayout(pattern string) *PatternLayout {
	return &PatternLayout{pattern: pattern}
}

func (l *PatternLayout) Pattern() string {
	return l.pattern
}
