package log4j2

import (
	"encoding/xml"
	"os"
	"testing"
)

func getTestXmlConfiguration() []byte {
	xmlFile := "testdata/log4j2.xml"
	config, err := os.ReadFile(xmlFile)
	if err != nil {
		panic(err)
	}
	return config
}

func TestXmlParser(t *testing.T) {
	t.Run("ensure extractProperties extracts properties", func(t *testing.T) {
		configXml := getTestXmlConfiguration()
		parsedXml := xmlNode{}
		err := xml.Unmarshal(configXml, &parsedXml)
		if err != nil {
			t.Errorf("unable to parse xml: %v", err)
		}
		config := Config{
			appenders:  map[string]Appender{},
			properties: map[string]string{},
		}
		extractProperties(parsedXml, &config)
		if len(config.Properties()) == 0 {
			t.Errorf("expected properties to be extracted")
		}

		if path, ok := config.Properties()["discovery.log.file.path"]; !ok || path != "/var/opt/tableau/tableau_server/data/tabsvc/logs/activationservice/activationservice-discovery_node1-0.log" {
			t.Errorf("expected property to be extracted")
		}
	})
	t.Run("ensure extractAppenders extracts appenders", func(t *testing.T) {
		configXml := getTestXmlConfiguration()
		parsedXml := xmlNode{}
		err := xml.Unmarshal(configXml, &parsedXml)
		if err != nil {
			t.Errorf("unable to parse xml: %v", err)
		}
		config := Config{
			appenders:  map[string]Appender{},
			properties: map[string]string{},
		}
		extractAppenders(parsedXml, &config)
		if len(config.Appenders()) == 0 {
			t.Errorf("expected appenders to be extracted")
		}

		_, ok := config.Appenders()["dailyFileDiscoveryService"]
		if !ok {
			t.Errorf("expected appender to be extracted")
		}
	})
	t.Run("ensure NewConfig provides a new Config that automatically replaces property keys with their respective values in appender field getters", func(t *testing.T) {
		configXml := getTestXmlConfiguration()
		parsedXml := xmlNode{}
		err := xml.Unmarshal(configXml, &parsedXml)
		if err != nil {
			t.Errorf("unable to parse xml: %v", err)
		}
		config := NewConfig(getTestXmlConfiguration())
		appenderInstance, ok := config.Appenders()["dailyFileDiscoveryService"]
		if !ok {
			t.Errorf("expected appender to be extracted")
		}
		if appenderInstance.Filename() != "/var/opt/tableau/tableau_server/data/tabsvc/logs/activationservice/activationservice-discovery_node1-0.log" {
			t.Errorf("expected property key to be automatically replaced with its value")
		}

	})
}
