package log4j2_test

import (
	"github.com/highperformance-tech/ts-olly/internal/log4j2"
	"regexp"
	"testing"
)

func TestRegex(t *testing.T) {
	patternTests := []struct {
		description string
		given       string
		expected    string
	}{
		{`logger (short pattern name)`, `%c`, `(?P<logger>\S+)`},
		{`logger (long pattern name)`, `%logger`, `(?P<logger>\S+)`},
		{`logger (ignores precision modifier)`, `%c{1}`, `(?P<logger>\S+)`},
		{`level (short pattern name)`, `%p`, `(?P<level>\w+)`},
		{`level (long pattern name)`, `%level`, `(?P<level>\w+)`},
		{`level (left-justified padded)`, `%-5p`, `(?P<level>\w+)\s*`},
		{`level (right-justified padded)`, `%5p`, `\s*(?P<level>\w+)`},
		{`X/MDC (key name becomes match group name)`, `%X{lorem}`, `(?P<lorem>.*?)`},
		{`X/MDC (key name becomes match group name)`, `%X{ipsum}`, `(?P<ipsum>.*?)`},
		{`X/MDC (key names should be allowed overlaps)`, `%X{ipsum},%X{loremipsum}`, `(?P<ipsum>.*?),(?P<loremipsum>.*?)`},
		{`thread (short pattern name)`, `%t`, `(?P<thread>\S*)`},
		{`thread (long pattern name)`, `%thread`, `(?P<thread>\S*)`},
		{`message (short pattern name)`, `%m`, `(?s)(?P<message>.*)(?-s)`},
		{`message (medium pattern name)`, `%msg`, `(?s)(?P<message>.*)(?-s)`},
		{`message (long pattern name)`, `%message`, `(?s)(?P<message>.*)(?-s)`},
		{`date (short pattern name, default format)`, `%d`, `(?P<date>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[\.,]\d{3})`},
		{`date (long pattern name, default format)`, `%date`, `(?P<date>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[\.,]\d{3})`},
		{`date (custom format with decimal microseconds)`, `%d{yyyy-MM-dd HH:mm:ss.SSS Z}{UTC}`, `(?P<date>\d{2}\d{2}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3} [+-]\d{4})`},
		{`date (custom format with comma microseconds)`, `%d{yyyy-MM-dd HH:mm:ss,SSS Z}{UTC}`, `(?P<date>\d{2}\d{2}-\d{2}-\d{2} \d{2}:\d{2}:\d{2},\d{3} [+-]\d{4})`},
		{`date (malformed opening brace only)`, `%d{`, `(?P<date>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[\.,]\d{3})`},
		{`date (malformed incomplete format)`, `%d{incomplete`, `(?P<date>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[\.,]\d{3})`},
		{`date (malformed square brackets)`, `%d[test]`, `(?P<date>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[\.,]\d{3})`},
		{`X/MDC (malformed opening brace only)`, `%X{`, `%X\{`},
		{`X/MDC (malformed empty braces)`, `%X{}`, `%X\{\}`},
		{`X/MDC (single character key)`, `%X{a}`, `(?P<a>.*?)`},
		{`invalid pattern (incomplete)`, `%`, `%`},
		{`invalid pattern (non-word char)`, `%!`, `%!`},
		{`invalid pattern (unknown alias)`, `%999`, `%999`},
		{`text with trailing percent`, `[%p] text%`, `\[(?P<level>\w+)\] text%`},
		{`empty string`, ``, ``},
		{`no pattern markers`, `just text`, `just text`},
	}
	for _, test := range patternTests {
		t.Run(test.description, func(t *testing.T) {
			got := log4j2.Regexp(test.given, make(map[string]string))
			if got != test.expected {
				t.Errorf("%s: given %q, expected %q, got %q", test.description, test.given, test.expected, got)

			} else {
				_, err := regexp.Compile(got)
				if err != nil {
					t.Errorf("%s: result %q is not a valid regexp. fix the test", test.description, got)
				}
			}

		})
	}
	t.Run("custom matcher", func(t *testing.T) {
		pattern := `%X{lorem}`
		customMatchers := map[string]string{
			`%X{lorem}`: `(?P<lorem>\N?)`,
		}
		got := log4j2.Regexp(pattern, customMatchers)
		if got != customMatchers[pattern] {
			t.Errorf("given %q, expected %q, got %q", "%X{lorem}", customMatchers["lorem"], got)
		}
	})
}
