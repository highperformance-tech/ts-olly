package log4j

import "testing"

func TestAppender(t *testing.T) {
	t.Run("NewAppender", func(t *testing.T) {
		a := NewAppender("test", "test", NewLayout("test", "test", "test"), map[string]string{"test": "test"})
		if a.Name() != "test" {
			t.Errorf("expected name %q, got %q", "test", a.Name())
		}
		if a.Class() != "test" {
			t.Errorf("expected class %q, got %q", "test", a.Class())
		}
		if a.Layout().Name() != "test" {
			t.Errorf("expected layout name %q, got %q", "test", a.Layout().Name())
		}
		if a.Layout().Class() != "test" {
			t.Errorf("expected layout class %q, got %q", "test", a.Layout().Class())
		}
		if a.Layout().Pattern() != "test" {
			t.Errorf("expected layout pattern %q, got %q", "test", a.Layout().Pattern())
		}
		if a.Params()["test"] != "test" {
			t.Errorf("expected params %q, got %q", "test", a.Params()["test"])
		}
	})
}
