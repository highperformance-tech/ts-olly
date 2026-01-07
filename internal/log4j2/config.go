package log4j2

import (
	"encoding/xml"
	"regexp"
	"strings"
)

type Config struct {
	appenders  map[string]Appender
	properties map[string]string
}

func (c *Config) Empty() bool {
	return len(c.appenders) == 0 && len(c.properties) == 0
}

func (c *Config) Appenders() map[string]Appender {
	return c.appenders
}

func (c *Config) Properties() map[string]string {
	return c.properties
}

func NewConfig(configXml []byte) Config {
	config := Config{
		appenders:  map[string]Appender{},
		properties: map[string]string{},
	}

	parsedXml := xmlNode{}
	err := xml.Unmarshal(configXml, &parsedXml)
	if err != nil {
		return config
	}
	extractProperties(parsedXml, &config)
	extractAppenders(parsedXml, &config)
	return config
}

func extractProperties(parsedXml xmlNode, config *Config) {
	propertiesNodes := parsedXml.Child("Properties").Nodes
	for _, propertyNode := range propertiesNodes {

		name := propertyNode.Attribute("name")
		value := string(propertyNode.Content)
		config.properties[name] = value
	}
}

func extractAppenders(parsedXml xmlNode, config *Config) {
	child := parsedXml.Child("Appenders")
	appenderNodes := child.Children()
	for _, appenderNode := range appenderNodes {
		var (
			appenderType   string         = appenderNode.XMLName.Local
			name           string         = getValue(appenderNode.Attribute("name"), config.Properties())
			filename       string         = getValue(appenderNode.Attribute("fileName"), config.Properties())
			filePattern    string         = getValue(appenderNode.Attribute("filePattern"), config.Properties())
			patternLayout  *PatternLayout = extractPatternLayout(appenderNode, config.Properties())
			immediateFlush bool           = appenderNode.Attribute("immediateFlush") == "true"
		)

		config.appenders[name] = NewAppender(name, appenderType, filename, filePattern, patternLayout, immediateFlush)
	}
}

func getValue(key string, properties map[string]string) string {
	re := regexp.MustCompile(`(\$\{[\w\.]+\})`)
	for _, match := range re.FindAllString(key, -1) {
		if value, ok := properties[match[2:len(match)-1]]; ok {
			key = strings.Replace(key, match, value, -1)
		}
	}
	return key
}

func extractPatternLayout(appenderNode xmlNode, properties map[string]string) *PatternLayout {
	patternLayoutNode := appenderNode.Child("PatternLayout")
	var pattern string
	if patternLayoutNode.Attribute("pattern") != "" {
		pattern = patternLayoutNode.Attribute("pattern")
	} else if string(patternLayoutNode.Child("Pattern").Content) != "" {
		pattern = string(patternLayoutNode.Child("Pattern").Content)
	} else {
		pattern = "%m%n" // Log4j2 default pattern
	}
	pattern = getValue(pattern, properties)
	patternLayout := NewPatternLayout(pattern)
	return patternLayout
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
