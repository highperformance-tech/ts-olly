package httpd

import (
	lex "github.com/timtadh/lexmachine"
	"regexp"
	"strings"
)

type Config struct {
	definitions map[string]string
	formats     []string
}

func (c Config) Definitions() map[string]string {
	return c.definitions
}

func (c Config) Formats() []string {
	return c.formats
}

func (c Config) Empty() bool {
	return len(c.definitions) == 0 && len(c.formats) == 0
}

func From(config []byte) Config {
	defs := make(map[string]string)
	var formats []string
	l, err := newLexer()
	if err != nil {
		return Config{}
	}
	s, err := l.Scanner(config)
	if err != nil {
		return Config{}
	}
	tokens := make([]*lex.Token, 0)
	for tok, err, eof := s.Next(); !eof; tok, err, eof = s.Next() {
		if err != nil {
			return Config{}
		}
		token, ok := tok.(*lex.Token)
		if !ok {
			continue
		}
		tokens = append(tokens, token)
	}
	for i := 0; i < len(tokens)-2; i++ {
		if len(tokens[i:]) < 3 {
			break
		}
		tok := tokens[i : i+3]
		if l.Tokens[tok[0].Type] == "Define" && l.Tokens[tok[1].Type] == "ID" && l.Tokens[tok[2].Type] == "VALUE" {
			key, ok := tok[1].Value.(string)
			if !ok {
				continue
			}
			value, ok := tok[2].Value.(string)
			if !ok {
				continue
			}
			defs[key] = strings.Trim(
				strings.Replace(value, "\\\n", "", -1),
				`"`)
			i += 2
		}
		if l.Tokens[tok[0].Type] == "LogFormat" && l.Tokens[tok[1].Type] == "VALUE" && l.Tokens[tok[2].Type] == "ID" {
			format, ok := tok[1].Value.(string)
			if !ok {
				continue
			}
			if len(format) == 0 {
				continue
			}
			if format[0] == '"' {
				format = format[1:]
			}
			if len(format) > 0 && format[len(format)-1] == '"' {
				format = format[:len(format)-1]
			}
			formats = append(formats, format)
			i += 2
		}
	}
	for i, format := range formats {
		formats[i] = getValue(format, defs)
	}
	return Config{
		definitions: defs,
		formats:     formats,
	}
}

func getValue(original string, definitions map[string]string) string {
	replacement := original
	re := regexp.MustCompile(`(\$\{[\w\.]+\})`)
	for _, match := range re.FindAllString(original, -1) {
		if value, ok := definitions[match[2:len(match)-1]]; ok {
			replacement = strings.Replace(original, match, value, -1)
		}
	}
	replacement = strings.Replace(replacement, "${ANNOTATED_HTTP_CODES}", "", -1)
	return replacement
}
