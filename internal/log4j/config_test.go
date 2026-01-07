package log4j

import (
	"os"
	"testing"
)

func TestXmlParser(t *testing.T) {
	t.Run("valid log4j.properties", func(t *testing.T) {
		propertiesFile, err := os.ReadFile("testdata/log4j.properties")
		if err != nil {
			t.Errorf("unable to read properties file: %v", err)
		}
		cfg := FromProperties(propertiesFile)
		if len(cfg.Appenders()) != 3 {
			t.Errorf("expected 3 appenders, got %d", len(cfg.Appenders()))
		}
	})
	t.Run("valid log4j.xml", func(t *testing.T) {
		xmlFile := "testdata/log4j.xml"
		configXml, err := os.ReadFile(xmlFile)
		if err != nil {
			panic(err)
		}
		config := FromXML(configXml)
		if config.Empty() {
			t.Errorf("expected non-empty config, got empty config")
		}
		if len(config.Appenders()) != 4 {
			t.Errorf("expected 4 appenders, got %d", len(config.Appenders()))
		}
		if len(config.Loggers()) != 12 {
			t.Errorf("expected 12 loggers, got %d", len(config.Loggers()))
		}
		if config.Loggers()["root"].Appender().Name() != "dailyFile" {
			t.Errorf("expected appender name 'dailyFile', got %q", config.Loggers()["root"].Appender().Name())
		}
		if config.Loggers()["root"].Appender().Layout().Pattern() != "%d{yyyy-MM-dd HH:mm:ss.SSS Z}{UTC} (%X{siteName},%X{userName},%X{wgsessionid},%X{requestId},%X{localRequestId}) %t %X{serviceName}: %-5p %c - %m%n" {
			t.Error("expected appender layout pattern '%d{yyyy-MM-dd HH:mm:ss.SSS Z}{UTC} (%X{siteName},%X{userName},%X{wgsessionid},%X{requestId},%X{localRequestId}) %t %X{serviceName}: %-5p %c - %m%n', got '" + config.Loggers()["root"].Appender().Layout().Pattern() + "'")
		}
		if config.Loggers()["root"].Name() != "root" {
			t.Errorf("expected logger name 'root', got %q", config.Loggers()["root"].Name())
		}
		if config.Loggers()["root"].Level() != "warn" {
			t.Errorf("expected logger level 'warn', got %q", config.Loggers()["root"].Level())
		}
	})
	t.Run("invalid log4j.xml", func(t *testing.T) {
		xmlFile := "testdata/bad-log4j.xml"
		configXml, err := os.ReadFile(xmlFile)
		if err != nil {
			panic(err)
		}
		config := FromXML(configXml)
		if !config.Empty() {
			t.Errorf("expected empty config, got non-empty config")
		}
	})
	t.Run("regex parses correctly", func(t *testing.T) {
		xmlFile := "testdata/log4j.xml"
		configXml, err := os.ReadFile(xmlFile)
		if err != nil {
			panic(err)
		}
		config := FromXML(configXml)
		for _, appender := range config.Appenders() {
			pattern := appender.Layout().Pattern()
			regexPattern := Regexp(pattern, make(map[string]string))
			if regexPattern == "" {
				t.Errorf("expected non-empty regex pattern, got empty regex pattern")
			}
		}
	})
}
