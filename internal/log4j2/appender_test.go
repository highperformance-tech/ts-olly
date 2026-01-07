package log4j2

import "testing"

func TestAppender(t *testing.T) {
	t.Run("NewAppender", func(t *testing.T) {
		a := NewAppender("test", "test", "test", "test", NewPatternLayout("test"), true)
		if a.Name() != "test" {
			t.Errorf("expected name %q, got %q", "test", a.Name())
		}
		if a.Type() != "test" {
			t.Errorf("expected type %q, got %q", "test", a.Type())
		}
		if a.Filename() != "test" {
			t.Errorf("expected filename %q, got %q", "test", a.Filename())
		}
		if a.FilePattern() != "test" {
			t.Errorf("expected filepattern %q, got %q", "test", a.FilePattern())
		}
		if !a.ImmediateFlush() {
			t.Errorf("expected immediate flush")
		}
		if a.PatternLayout().Pattern() != "test" {
			t.Errorf("expected pattern %q, got %q", "test", a.PatternLayout().Pattern())
		}
	})
}
