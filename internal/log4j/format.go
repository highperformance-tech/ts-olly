package log4j

import (
	"fmt"
	"os"
	"strings"
)

func GetFormats(path string) ([]string, error) {
	configFile, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read log4j config %s: %w", path, err)
	}
	var cfg Config
	if strings.HasSuffix(path, "xml") {
		cfg = FromXML(configFile)
	} else if strings.HasSuffix(path, "properties") {
		cfg = FromProperties(configFile)
	}
	if cfg.Empty() {
		return nil, fmt.Errorf("invalid configuration in %s", path)
	}
	formats := make([]string, 0)
	for _, appender := range cfg.Appenders() {
		if pattern := appender.Layout().Pattern(); pattern != "" {
			pattern = Regexp(pattern, make(map[string]string))
			if !in(pattern, formats) {
				formats = append(formats, pattern)
			}
		}
	}
	return formats, nil
}

func in(pattern string, patterns []string) bool {
	for _, p := range patterns {
		if pattern == p {
			return true
		}
	}
	return false
}
