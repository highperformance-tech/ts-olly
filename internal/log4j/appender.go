package log4j

type Appender struct {
	layout Layout
	class  string
	name   string
	params map[string]string
}

func NewAppender(name, class string, layout Layout, params map[string]string) Appender {
	return Appender{
		name:   name,
		class:  class,
		layout: layout,
		params: params,
	}
}

func (a Appender) Layout() Layout {
	return a.layout
}

func (a Appender) Class() string {
	return a.class
}

func (a Appender) Name() string {
	return a.name
}

func (a Appender) Params() map[string]string {
	return a.params
}
