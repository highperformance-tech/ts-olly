package httpd

import (
	"os"
	"testing"
)

func TestConfig_Definitions(t *testing.T) {
	c, err := os.ReadFile("testdata/httpd.conf")
	if err != nil {
		t.Fatal(err)
	}
	config := From(c)
	numDefsWanted := 2
	if len(config.Definitions()) != numDefsWanted {
		t.Errorf("Expected %d definitions, got %d", numDefsWanted, len(config.Definitions()))
	}
	numFormatsWanted := 2
	if len(config.Formats()) != numFormatsWanted {
		t.Errorf("Expected %d formats, got %d", numFormatsWanted, len(config.Formats()))
	}
}

func TestConfig_From(t *testing.T) {
	tests := []struct {
		description string
		config      string
		expectEmpty bool
	}{
		{"empty format value", `LogFormat "" common`, false},
		{"single quote format", `LogFormat " common`, true},
		{"malformed config", `LogFormat`, true},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			config := From([]byte(test.config))
			if config.Empty() != test.expectEmpty {
				t.Errorf("expected Empty()=%v, got %v", test.expectEmpty, config.Empty())
			}
		})
	}
}

func TestConfig_GetValue(t *testing.T) {
	tests := []struct {
		description string
		original    string
		definitions map[string]string
		expected    string
	}{
		{"short variable reference", "${}", map[string]string{}, "${}"},
		{"variable with value", "${a}", map[string]string{"a": "value"}, "value"},
		{"empty string", "", map[string]string{}, ""},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			got := getValue(test.original, test.definitions)
			if got != test.expected {
				t.Errorf("expected %q, got %q", test.expected, got)
			}
		})
	}
}
