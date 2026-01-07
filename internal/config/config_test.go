package config_test

import (
	"fmt"
	"github.com/highperformance-tech/ts-olly/internal/config"
	"testing"
)

func TestConfig(t *testing.T) {
	t.Run("test creating child key and setting value", func(t *testing.T) {
		c := config.Config
		c.Key("foo").Set("qux")
		if c.Key("foo").Get() != "qux" {
			t.Errorf("expected key %q to have value %q\n", "foo", "qux")
		}
	})
	t.Run("test creating nested child key and setting value", func(t *testing.T) {
		c := config.Config
		c.Key("foo").Key("bar").Key("baz").Set("qux")
		if c.Key("foo").Key("bar").Key("baz").Get() != "qux" {
			t.Errorf("expected key %q to have value %q\n", "foo.bar.baz", "qux")
		}
	})
	t.Run("test variadic version of creating nested child key and setting value", func(t *testing.T) {
		c := config.Config
		c.Key("foo", "bar", "baz").Set("qux")
		if c.Key("foo", "bar", "baz").Get() != "qux" {
			t.Errorf("expected key %q to have value %q\n", "foo.bar.baz", "qux")
		}
	})
	t.Run("test getting nested path", func(t *testing.T) {
		c := config.Config
		path := c.Key("foo", "bar", "baz").Path()
		expected := "foo.bar.baz"
		if path != expected {
			t.Errorf("expected %q, got %q\n", expected, path)
		}
	})
	t.Run("test stringer", func(t *testing.T) {
		c := config.Config
		baz := c.Key("foo", "bar", "baz")
		baz.Set("qux")
		wanted := "foo.bar.baz: qux"
		got := fmt.Sprintf("%v", baz)
		if got != wanted {
			t.Errorf("wanted %q, got %q\n", wanted, got)
		}
	})
	t.Run("test list as value", func(t *testing.T) {
		c := config.Config
		wanted := []string{
			"item1",
			"item2",
		}
		c.Key("foo").Set(wanted)
		got := c.Key("foo").Get()
		if fmt.Sprint(got) != fmt.Sprint(wanted) {
			t.Errorf("wanted %q, got %q\n", wanted, got)
		}
	})

	t.Run("test int as value", func(t *testing.T) {
		c := config.Config
		wanted := 25
		c.Key("foo").Set(wanted)
		got := c.Key("foo").Get()
		if got != wanted {
			t.Errorf("wanted %q, got %q\n", wanted, got)
		}
	})

	t.Run("test getting children", func(t *testing.T) {
		c := config.Config

		wanted := map[string]string{
			"bar": "qux",
			"baz": "qux",
		}

		for k, v := range wanted {
			c.Key("foo").Key(k).Set(v)
		}
		got := c.Key("foo").Children()
		if len(got) != len(wanted) {
			t.Errorf("expected %d children, got %d\n", len(wanted), len(got))
		}
		for _, v := range got {
			if v.Get() != wanted[v.Name()] {
				t.Errorf("expected %q, got %q\n", wanted[v.Path()], v.Get())
			}
		}
		keyList := c.Key("foo").Children().Keys()
		if len(keyList) != len(wanted) {
			t.Errorf("expected %d keys in the KeyList, got %d\n", len(wanted), len(keyList))
		}
		for _, v := range keyList {
			if _, ok := wanted[v]; !ok {
				t.Errorf("expected %q in wanted, but it wasn't\n", v)
			}
		}
	})
}
