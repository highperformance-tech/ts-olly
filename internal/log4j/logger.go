package log4j

type Logger struct {
	name     string
	level    string
	appender *Appender
}

func (l Logger) Name() string {
	return l.name
}

func (l Logger) Level() string {
	return l.level
}

func (l Logger) Appender() *Appender {
	return l.appender
}

func NewLogger(name, level string, appender *Appender) Logger {
	return Logger{
		name:     name,
		level:    level,
		appender: appender,
	}
}
