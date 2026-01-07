package log4j

import (
	"testing"
)

func TestGetFormats(t *testing.T) {
	t.Run("valid xml configuration returns valid instance", func(t *testing.T) {
		formats, err := GetFormats("testdata/log4j.xml")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(formats) != 1 {
			t.Errorf("expected 1 format, got %d", len(formats))
		}
	})
	t.Run("invalid xml configuration returns error", func(t *testing.T) {
		_, err := GetFormats("testdata/bad-log4j.xml")
		want := "invalid configuration in testdata/bad-log4j.xml"
		if err.Error() != want {
			t.Errorf("expected %q, got %q", want, err.Error())
		}
	})
	t.Run("valid properties configuration returns valid instance", func(t *testing.T) {
		formats, err := GetFormats("testdata/log4j.properties")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(formats) != 3 {
			t.Errorf("expected 3 format, got %d", len(formats))
		}
	})
	t.Run("invalid properties configuration returns error", func(t *testing.T) {
		_, err := GetFormats("testdata/bad-log4j.properties")
		want := "invalid configuration in testdata/bad-log4j.properties"
		if err.Error() != want {
			t.Errorf("expected %q, got %q", want, err.Error())
		}
	})
}
