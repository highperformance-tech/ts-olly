package log4j2

type Appender struct {
	immediateFlush bool
	patternLayout  *PatternLayout
	appenderType   string
	name           string
	filename       string
	filePattern    string
}

func NewAppender(name, appenderType, filename, filePattern string, patternLayout *PatternLayout, immediateFlush bool) Appender {
	return Appender{
		immediateFlush: immediateFlush,
		patternLayout:  patternLayout,
		appenderType:   appenderType,
		name:           name,
		filename:       filename,
		filePattern:    filePattern,
	}
}

func (a Appender) ImmediateFlush() bool {
	return a.immediateFlush
}

func (a Appender) PatternLayout() *PatternLayout {
	return a.patternLayout
}

func (a Appender) Type() string {
	return a.appenderType
}

func (a Appender) Name() string {
	return a.name
}

func (a Appender) Filename() string {
	return a.filename
}

func (a Appender) FilePattern() string {
	return a.filePattern
}
