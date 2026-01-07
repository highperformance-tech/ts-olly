package log4j

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/go-viper/encoding/javaproperties"
	"github.com/spf13/viper"
	"path/filepath"
)

type Config struct {
	appenders map[string]Appender
	loggers   map[string]Logger
}

func (c *Config) Empty() bool {
	return len(c.appenders) == 0 && len(c.loggers) == 0
}

func (c *Config) Appenders() map[string]Appender {
	return c.appenders
}

func (c *Config) Loggers() map[string]Logger {
	return c.loggers
}

func FromProperties(properties []byte) Config {
	// Register Java properties codec for Viper v1.20+
	codecRegistry := viper.NewCodecRegistry()
	codec := &javaproperties.Codec{}
	codecRegistry.RegisterCodec("properties", codec)

	v := viper.NewWithOptions(
		viper.WithCodecRegistry(codecRegistry),
	)
	v.SetConfigType("properties")
	err := v.ReadConfig(bytes.NewReader(properties))
	if err != nil {
		return Config{}
	}
	appenders := make(map[string]Appender)
	for k := range v.GetStringMap("log4j.appender") {
		name := k
		file := v.GetString(fmt.Sprintf("log4j.appender.%s.file", name))
		file = filepath.Base(file)
		if file == "." {
			file = "stdout"
		}
		appenderClass := v.GetString(fmt.Sprintf("log4j.appender.%s", name))
		layoutPattern := v.GetString(fmt.Sprintf("log4j.appender.%s.layout.conversionpattern", name))
		layoutClass := v.GetString(fmt.Sprintf("log4j.appender.%s.layout", name))
		layout := NewLayout(layoutClass, file, layoutPattern)
		appenders[name] = NewAppender(name, appenderClass, layout, nil)
	}
	return Config{
		appenders: appenders,
	}
}

func FromXML(configXml []byte) Config {
	config := Config{
		appenders: make(map[string]Appender),
		loggers:   make(map[string]Logger),
	}
	parsedXml := xmlNode{}
	err := xml.Unmarshal(configXml, &parsedXml)
	if err != nil {
		return config
	}
	for _, a := range parsedXml.Children() {
		switch a.XMLName.Local {
		case "appender":
			config = withAppenderXML(config, a)
		case "logger":
			config = withLoggerXML(config, a)
		case "root":
			config = withRootLoggerXML(config, a)
		}
	}
	return config
}

func withLoggerXML(config Config, loggerXml xmlNode) Config {
	name := loggerXml.Attribute("name")
	levelNode := loggerXml.Child("level")
	level := levelNode.Attribute("value")
	appender := &Appender{}
	config.loggers[name] = NewLogger(name, level, appender)
	return config
}

func withRootLoggerXML(config Config, rootXml xmlNode) Config {
	name := "root"
	levelNode := rootXml.Child("priority")
	level := levelNode.Attribute("value")
	appender := &Appender{}
	if ref := rootXml.Child("appender-ref"); ref.XMLName.Local == "appender-ref" {
		refAppender := config.Appenders()[ref.Attribute("ref")]
		appender = &refAppender
	}
	config.loggers[name] = NewLogger(name, level, appender)
	return config
}

func withAppenderXML(config Config, appenderXml xmlNode) Config {
	var (
		name   string            = appenderXml.Attribute("name")
		class  string            = appenderXml.Attribute("class")
		params map[string]string = make(map[string]string)
		layout Layout
	)
	appender := NewAppender(name, class, layout, params)
	if ref := appenderXml.Child("appender-ref"); ref.XMLName.Local == "appender-ref" {
		refAppender := config.Appenders()[ref.Attribute("ref")]
		appender.layout = refAppender.Layout()
		appender.params = refAppender.Params()
	}
	for _, child := range appenderXml.Children() {
		if child.XMLName.Local == "param" {
			appender.params[child.Attribute("name")] = child.Attribute("value")
		}
		if child.XMLName.Local == "layout" {
			appender.layout = extractLayoutXML(child)
		}
	}
	config.appenders[name] = appender
	return config
}

func extractLayoutXML(layout xmlNode) Layout {
	class := layout.Attribute("class")
	childParam := layout.Child("param")
	name := childParam.Attribute("name")
	pattern := childParam.Attribute("value")
	return NewLayout(class, name, pattern)
}

type xmlNode struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:"-"`
	Content []byte     `xml:",innerxml"`
	Nodes   []xmlNode  `xml:",any"`
}

func (n *xmlNode) Children() []xmlNode {
	return n.Nodes
}

func (n *xmlNode) Child(name string) xmlNode {
	for _, child := range n.Nodes {
		if child.XMLName.Local == name {
			return child
		}
	}
	return xmlNode{}
}

func (n *xmlNode) Attribute(name string) string {
	for _, attr := range n.Attrs {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

func (n *xmlNode) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	n.Attrs = start.Attr
	type node xmlNode

	return d.DecodeElement((*node)(n), &start)
}
