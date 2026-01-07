package log4j

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// log4jPatterns is a map of functions that will replace a logs pattern with
// a Go regexp pattern.
var log4jPatterns = map[string]func(conversion) conversionToRegex{
	"date": func(conversion conversion) conversionToRegex {
		// If there's no modifier, use the default date format as per
		// https://logging.apache.org/log4j/log4j-2.1/manual/layouts.html
		pattern := `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[\.,]\d{3}`

		// If there's a modifier, use it to format the date
		if conversion.Modifier != "" {
			// We don't care about a timezone modifier (the second parameter), because it just changes the numbers, not their format.
			// Extract the first match of `\{[^}]*\}` from the modifier and use it as the format input.
			re := regexp.MustCompile(`\{([^}]*)}`)
			matches := re.FindStringSubmatch(conversion.Modifier)
			if len(matches) > 1 {
				format := matches[1]
				pattern = convertDateFormat(format)
			}

		}

		return conversionToRegex{
			Name:  `date`,
			Match: fmt.Sprintf(`(?P<date>%s)`, pattern),
		}
	},
	"number": func(conversion conversion) conversionToRegex {
		// Regardless of the precision, we'll match anything that's not a space.
		return conversionToRegex{
			Name:  `number`,
			Match: `(?P<number>\d+)`,
		}
	},
	"level": func(conversion conversion) conversionToRegex {
		// If we're given a minimum width, we need to allow for zero or more spaces
		padding := ``
		if conversion.Min != 0 {
			padding = `\s*`
		}
		// If we're left-justified, we pad on the right. Otherwise, we pad on the left.
		leftPadding := ``
		rightPadding := ``
		if conversion.LeftJustified {
			rightPadding = padding
		} else {
			leftPadding = padding
		}
		// Return the conversion.
		return conversionToRegex{
			Name:  `level`,
			Match: fmt.Sprintf(`%s(?P<level>\w+)%s`, leftPadding, rightPadding),
		}
	},
	"logger": func(conversion conversion) conversionToRegex {
		// Regardless of the precision, we'll match anything that's not a space.
		return conversionToRegex{
			Name:  `logger`,
			Match: `(?P<logger>\S+)`,
		}
	},
	"message": func(conversion conversion) conversionToRegex {
		return conversionToRegex{
			Name:  `message`,
			Match: `(?s)(?P<message>.*)(?-s)`,
		}
	},
	"n": func(conversion conversion) conversionToRegex {
		return conversionToRegex{
			Name:  `newline`,
			Match: `[$\n]`,
		}
	},
	"thread": func(conversion conversion) conversionToRegex {
		return conversionToRegex{
			Name:  `thread`,
			Match: `(?P<thread>\S*)`,
		}
	},
	"X": func(conversion conversion) conversionToRegex {
		// This capture group will be named based on the text in the modifier.
		name := ""
		if len(conversion.Modifier) >= 2 {
			name = strings.ToLower(conversion.Modifier[1 : len(conversion.Modifier)-1])
		}
		return conversionToRegex{
			Name:  name,
			Match: `(?P<` + name + `>.*?)`,
		}
	},
}

// aliases is a map of aliases for logs patterns.
var aliases = map[string]string{
	"c":       "logger",
	"d":       "date",
	"date":    "date",
	"i":       "number",
	"level":   "level",
	"logger":  "logger",
	"m":       "message",
	"mdc":     "message",
	"MDC":     "message",
	"message": "message",
	"msg":     "message",
	"n":       "n",
	"p":       "level",
	"t":       "thread",
	"thread":  "thread",
	"X":       "X",
}

type conversion struct {
	LeftJustified bool
	Min           uint8
	Max           uint64
	Pattern       string
	Modifier      string
}

type conversionToRegex struct {
	Name  string
	Match string
}

var groupChars = map[string]string{
	`(`: `)`,
	`{`: `}`,
	`[`: `]`,
	`>`: `<`,
	`<`: `>`,
	`]`: `[`,
	`}`: `{`,
	`)`: `(`,
}

var dateFormats = map[string]string{
	`yy`:  `\d{2}`,
	`MM`:  `\d{2}`,
	`dd`:  `\d{2}`,
	`HH`:  `\d{2}`,
	`mm`:  `\d{2}`,
	`ss`:  `\d{2}`,
	`SSS`: `\d{3}`,
	`Z`:   `[+-]\d{4}`,
}

func convertDateFormat(format string) string {
	// Escape all the characters that are special to regular expressions.
	format = regexp.QuoteMeta(format)
	// Replace all the date formats with their regex equivalent.
	for k, v := range dateFormats {
		format = strings.Replace(format, k, v, -1)
	}
	return format
}

func Regexp(pattern string, customMatchers map[string]string) string {
	return replacePatterns(pattern, customMatchers)
}

func replacePatterns(message string, customMatchers map[string]string) string {
	patterns := extractPatterns(message)
	re := regexp.MustCompile(`%(?P<lj>-?)(?P<min>\d*)\.?(?P<max>\d*)(?P<pattern>\w+)(?P<modifier>[\[\{].*)?`)
	leftJustifiedIndex := re.SubexpIndex("lj")
	minCharsIndex := re.SubexpIndex("min")
	maxCharsIndex := re.SubexpIndex("max")
	patternIndex := re.SubexpIndex("pattern")
	modifierIndex := re.SubexpIndex("modifier")
	replacements := make(map[string]string)
	for _, p := range patterns {
		matches := re.FindStringSubmatch(p)
		if matches == nil {
			continue
		}
		leftJustified := matches[leftJustifiedIndex] == "-"
		minChars, err := strconv.ParseUint(matches[minCharsIndex], 10, 8)
		if err != nil {
			minChars = uint64(0)
		}
		maxChars, err := strconv.ParseUint(matches[maxCharsIndex], 10, 64)
		if err != nil {
			maxChars = uint64(0)
		}
		pattern := aliases[matches[patternIndex]]
		if pattern == "" {
			continue
		}
		modifier := matches[modifierIndex]
		conversion := conversion{
			LeftJustified: leftJustified,
			Min:           uint8(minChars),
			Max:           maxChars,
			Pattern:       pattern,
			Modifier:      modifier,
		}

		// Skip X patterns with empty names - they should be escaped as literals
		if pattern == "X" {
			name := ""
			if len(modifier) >= 2 {
				name = strings.ToLower(modifier[1 : len(modifier)-1])
			}
			if name == "" {
				// Skip this pattern, it will be escaped as a literal
				continue
			}
		}

		replacementString := "\u0000" + log4jPatterns[pattern](conversion).Name + "\u0000" // this is a hack to make sure the replacement string is unique (e.g. loremipsum's "ipsum" isn't replaced with ipsum's regex
		if customMatchers[p] != "" {
			replacements[replacementString] = customMatchers[p]
		} else {
			replacements[replacementString] = log4jPatterns[pattern](conversion).Match
		}
		message = strings.Replace(message, p, replacementString, -1)
	}
	// Now we'll escape the message.
	message = regexp.QuoteMeta(message)
	// Now we'll replace the replacements.
	for k, v := range replacements {
		message = strings.Replace(message, k, v, -1)
	}
	return message
}

func extractPatterns(pattern string) []string {
	given := []rune(pattern)
	var patterns []string
	var currentPattern string
	inPattern := false
	groupLevel := uint8(0)
	groups := make(map[uint8]string)
	for pos, char := range given {
		if pos == len(given)-1 {
			currentPattern += string(char)
			patterns = append(patterns, currentPattern)
			break
		}
		if !inPattern {
			if char == '%' {
				inPattern = true
				currentPattern = string(char)
			}
			continue
		}

		if char == '%' {
			patterns = append(patterns, currentPattern)
			currentPattern = string(char)
			continue
		}
		if char == '{' || char == '[' || char == '(' || char == '<' {
			groupLevel++
			currentPattern += string(char)
			groups[groupLevel] = string(char)
			continue
		}
		if (char == '}' || char == ']' || char == ')' || char == '>') && groupLevel > 0 && groups[groupLevel] == groupChars[string(char)] {
			delete(groups, groupLevel)
			groupLevel--
			currentPattern += string(char)
			continue
		}
		if groupLevel != 0 {
			currentPattern += string(char)
			continue
		}
		if groupLevel == 0 {
			if regexp.MustCompile(`[\w\d_\.-]`).MatchString(string(char)) {
				currentPattern += string(char)
				continue
			}
			patterns = append(patterns, currentPattern)
			inPattern = false
			currentPattern = ""
		}
	}
	return patterns
}
