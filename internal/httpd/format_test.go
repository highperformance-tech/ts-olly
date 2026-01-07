package httpd

import "testing"

func TestGetFormats(t *testing.T) {
	t.Run("valid httpd.conf file returns valid instance", func(t *testing.T) {
		formats, err := GetFormats("testdata/httpd.conf")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(formats) != 2 {
			t.Errorf("expected 2 formats, got %d", len(formats))
		}
	})
	t.Run("invalid httpd.conf file returns error", func(t *testing.T) {
		_, err := GetFormats("testdata/bad-httpd.conf")
		want := "invalid configuration in testdata/bad-httpd.conf"
		if err.Error() != want {
			t.Errorf("expected %q, got %q", want, err.Error())
		}
	})
}
