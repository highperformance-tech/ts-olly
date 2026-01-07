package log4j

type Layout struct {
	class   string
	name    string
	pattern string
}

func NewLayout(class, name, pattern string) Layout {
	return Layout{
		class:   class,
		name:    name,
		pattern: pattern,
	}
}

func (l Layout) Class() string {
	return l.class
}

func (l Layout) Name() string {
	return l.name
}

func (l Layout) Pattern() string {
	return l.pattern
}
