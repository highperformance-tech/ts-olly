package httpd

import (
	"fmt"
	"os"
)

func GetFormats(path string) ([]string, error) {
	c, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read httpd config %s: %w", path, err)
	}
	cfg := From(c)
	if cfg.Empty() {
		return nil, fmt.Errorf("invalid configuration in %s", path)
	}
	formats := make([]string, 0)
	for _, format := range cfg.Formats() {
		formats = append(formats, Regexp(format))
	}
	return formats, nil
}
