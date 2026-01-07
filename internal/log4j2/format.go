package log4j2

import (
	"fmt"
	"os"
	"path/filepath"
)

func GetFormats(path string) (map[string]string, error) {
	configXml, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read log4j2 config %s: %w", path, err)
	}
	cfg := NewConfig(configXml)
	if cfg.Empty() {
		return nil, fmt.Errorf("invalid configuration in %s", path)
	}
	formats := make(map[string]string)
	for _, appender := range cfg.Appenders() {
		if appender.PatternLayout() != nil {
			formatName := filepath.Base(appender.Filename())
			if appender.Name() == "standardOut" {
				formatName = "stdout"
			}
			formats[formatName] = Regexp(appender.PatternLayout().Pattern(), make(map[string]string))
		}
	}
	return formats, nil
}
