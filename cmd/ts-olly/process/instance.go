package process

import (
	"bufio"
	"fmt"
	"github.com/highperformance-tech/ts-olly/internal/httpd"
	"github.com/highperformance-tech/ts-olly/internal/log4j"
	"github.com/highperformance-tech/ts-olly/internal/log4j2"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Instance interface {
	ID() int
	Process() *Process
	Config() *viper.Viper
}

type instance struct {
	id      int
	process *Process
	config  viper.Viper
}

// ID returns the ID of the instance.
func (i instance) ID() int {
	return i.id
}

// Process returns a pointer to the process of the instance.
func (i instance) Process() *Process {
	return i.process
}

// Config returns a pointer to the configuration of the instance.
func (i instance) Config() *viper.Viper {
	return &i.config
}

// GetLogFormat returns the log format for the given file.
func (i instance) GetLogFormat(file string) string {
	line, err := firstLine(file)
	if err != nil {
		return ""
	}

	// Is this a JSON format? We test this first because some log4j2-based processes log the message in json
	// with a log4j2 format of just the message.
	if line[0] == '{' {
		return "json"
	}

	// Does this file have a dedicated format?
	namedLogFormats := i.Config().GetStringMapString("logs.formats.named")
	for name, format := range namedLogFormats {
		if strings.Contains(file, name) {
			return format
		}
	}

	// Does this file match a generic format?
	patterns := []string{
		`^\[(?P<date>\w{3} \w{3} \d{2} \d{2}:\d{2}:\d{2}[\.,]\d{6} \d{4})\] \[(?P<module>\S*):(?P<level>\w+)\] \[pid (?P<pid>\d+):tid (?P<tid>\d+)\] (?s)(?P<message>.*)(?-s)[$\n]?`,
		`(?P<level>\w+)\s* (?P<date>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}[\.,]\d{3} [+-]\d{4}) (?P<thread>\S*) : (?P<class>\S+) - (?s)(?P<message>.*)(?-s)[$\n]?`,
		`^(?P<date>\d{2}-\w{3}-\d{4} \d{2}:\d{2}:\d{2}[\.,]\d{3}) (?P<level>\w+) \[(?P<thread>.*)\] (?P<class>\S+) (?s)(?P<message>.*)(?-s)[$\n]?`,
		`^\[(?P<pid>\d+)\] \[(?P<level>\w+)\] (?P<date>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}[\.,]\d{3} [+-]\d{4}) : (?s)(?P<message>.*)(?-s)[$\n]?`,
		`(?P<pid>\d+):(?P<role>\w) (?P<date>\d{2} \w{3} \d{4} \d{2}:\d{2}:\d{2}[\.,]\d{3}) (?P<level>\S) (?s)(?P<message>.*)(?-s)[$\n]?`,
		`^(?P<date>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}[\.,]\d{3} [-+]\d{4}) (?P<thread>\S*) : (?P<level>\w+)\s* (?P<class>\S+) - (?s)(?P<message>.*)(?-s)[$\n]?`,
		`^\[(?P<date>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[\.,]\d{3})\]\[(?P<level>\w+)\s*\]\[(?P<logger>\S*)\s*\] \[(?P<node>\S*)\s*\](?s)(?P<message>.*)(?-s)[$\n]?`,
	}
	patterns = append(i.Config().GetStringSlice("logs.formats.generic"), patterns...)
	for _, format := range patterns {
		if regexp.MustCompile(format).MatchString(line) {
			return format
		}
	}

	// Otherwise, we don't know what the format is, so return an empty string.
	return ""
}

func firstLine(file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", fmt.Errorf("open file %s: %w", file, err)
	}
	defer f.Close()

	buf := bufio.NewReader(f)
	line, err := buf.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", fmt.Errorf("read first line from %s: %w", file, err)
	}
	return line, nil
}

func For(id uint8, name, configDir string) (*instance, error) {
	de, err := os.ReadDir(configDir)
	if err != nil {
		return nil, ErrConfigDirNotFound
	}
	for _, entry := range de {
		if strings.HasPrefix(entry.Name(), fmt.Sprintf("%s_%d", name, id)) {
			return FromConfig(filepath.Join(configDir, entry.Name()))
		}
	}
	return nil, ErrConfigDirNotFound
}

// FromConfig instantiates a new instance of the process from the given configuration directory.
func FromConfig(directory string) (*instance, error) {
	if fi, err := os.Stat(directory); err != nil || !fi.IsDir() {
		if os.IsNotExist(err) {
			return nil, ErrConfigDirNotFound
		}
		return nil, fmt.Errorf("stat config directory %s: %w", directory, err)
	}
	if fi, err := os.Stat(filepath.Join(directory, "workgroup.yml")); err != nil || !fi.Mode().IsRegular() {
		if os.IsNotExist(err) {
			return nil, ErrConfigFileNotFound
		}
		return nil, fmt.Errorf("stat workgroup.yml in %s: %w", directory, err)
	}

	cfg := viper.New()
	cfg.SetConfigName("workgroup")
	cfg.SetConfigFile(filepath.Join(directory, "workgroup.yml"))
	if err := cfg.ReadInConfig(); err != nil {
		if strings.Contains(err.Error(), "While parsing config") {
			return nil, ErrInvalidConfigFile
		}
		return nil, fmt.Errorf("read config %s: %w", filepath.Join(directory, "workgroup.yml"), err)
	}
	namedFormats := make(map[string]string)
	dirEntries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read directory %s: %w", directory, err)
	}
	for _, entry := range dirEntries { // Sometimes we'll have more than one with this pattern
		if strings.Contains(entry.Name(), "log4j2.xml") {
			if _, err := os.Stat(filepath.Join(directory, entry.Name())); err == nil {
				formats, err := GetLog4j2Config(filepath.Join(directory, entry.Name()))
				if err != nil {
					return nil, fmt.Errorf("get log4j2 config from %s: %w", filepath.Join(directory, entry.Name()), err)
				}
				for name, format := range formats {
					namedFormats[name] = format
				}
			}
		}
	}
	if len(namedFormats) != 0 {
		cfg.Set("logs.formats.named", namedFormats)
	}
	var formats []string
	if _, err := os.Stat(filepath.Join(directory, "log4j.xml")); err == nil {
		formats, err = GetLog4jConfig(filepath.Join(directory, "log4j.xml"))
		if err != nil {
			return nil, fmt.Errorf("get log4j config from %s: %w", filepath.Join(directory, "log4j.xml"), err)
		}
	}
	if _, err := os.Stat(filepath.Join(directory, "httpd.conf")); err == nil {
		formats, err = GetHttpdConfig(filepath.Join(directory, "httpd.conf"))
		if err != nil {
			return nil, fmt.Errorf("get httpd config from %s: %w", filepath.Join(directory, "httpd.conf"), err)
		}
	}
	if len(formats) != 0 {
		cfg.Set("logs.formats.generic", formats)
	}

	i := instance{
		config: *cfg,
	}
	return &i, nil
}

func GetLog4j2Config(path string) (map[string]string, error) {
	formats, err := log4j2.GetFormats(path)
	if err != nil && strings.Contains(err.Error(), "invalid configuration") {
		return nil, ErrInvalidConfigFile
	}
	if err != nil {
		return nil, fmt.Errorf("parse log4j2 formats from %s: %w", path, err)
	}
	return formats, nil
}

func GetLog4jConfig(path string) ([]string, error) {
	formats, err := log4j.GetFormats(path)
	if err != nil && strings.Contains(err.Error(), "invalid configuration") {
		return nil, ErrInvalidConfigFile
	}
	if err != nil {
		return nil, fmt.Errorf("parse log4j formats from %s: %w", path, err)
	}
	return formats, nil
}

func GetHttpdConfig(path string) ([]string, error) {
	formats, err := httpd.GetFormats(path)
	if err != nil && strings.Contains(err.Error(), "invalid configuration") {
		return nil, ErrInvalidConfigFile
	}
	if err != nil {
		return nil, fmt.Errorf("parse httpd formats from %s: %w", path, err)
	}
	return formats, nil
}
