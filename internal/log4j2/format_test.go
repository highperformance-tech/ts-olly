package log4j2

import "testing"

func TestGetFormats(t *testing.T) {
	t.Run("valid log4j2.xml configuration returns valid instance", func(t *testing.T) {
		formats, err := GetFormats("testdata/log4j2.xml")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(formats) != 2 {
			t.Errorf("expected 2 formats, got %d", len(formats))
		}
		if formats["activationservice-metrics_node1-0.log"] != "(?s)(?P<message>.*)(?-s)" {
			t.Errorf("expected format %q, got %q", "(?s)(?P<message>.*)(?-s)", formats["activationservice-metrics_node1-0.log"])
		}
	})
	t.Run("valid controlapp.log4j2.xml configuration returns valid instance", func(t *testing.T) {
		formats, err := GetFormats("testdata/log4j2.xml")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(formats) != 2 {
			t.Errorf("expected 2 formats, got %d", len(formats))
		}
		if formats["activationservice-metrics_node1-0.log"] != "(?s)(?P<message>.*)(?-s)" {
			t.Errorf("expected format %q, got %q", "(?s)(?P<message>.*)(?-s)", formats["activationservice-metrics_node1-0.log"])
		}
	})
	t.Run("invalid configuration returns error", func(t *testing.T) {
		_, err := GetFormats("testdata/bad-log4j2.xml")
		want := "invalid configuration in testdata/bad-log4j2.xml"
		if err.Error() != want {
			t.Errorf("expected %q, got %q", want, err.Error())
		}
	})
}
