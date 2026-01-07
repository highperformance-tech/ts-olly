package process_test

import (
	"github.com/highperformance-tech/ts-olly/cmd/ts-olly/process"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestFromConfig(t *testing.T) {
	t.Run("valid configuration returns valid instance", func(t *testing.T) {
		p, err := process.FromConfig("testdata/valid/tabadmincontroller_0.20221.22.0712.0324")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		logsConfig := p.Config().GetStringMap("logs.formats")
		if logsConfig == nil {
			t.Errorf("expected logs configuration, got nil")
		}
	})
	t.Run("configs for all processes", func(t *testing.T) {
		de, _ := os.ReadDir("testdata/valid")
		for _, f := range de {
			//if strings.Contains(f.Name(), "vizportal") && f.IsDir() {
			if f.IsDir() {
				t.Run(f.Name(), func(t *testing.T) {
					nameAndId, _, ok := strings.Cut(f.Name(), ".")
					if !ok {
						t.Errorf("expected directory format processName_id.buildNum, got %s", f.Name())
					}
					name, id, ok := strings.Cut(nameAndId, "_")
					if !ok {
						t.Errorf("expected directory format to start with processName_id, got %s", nameAndId)
					}
					var idUint uint8
					if idInt, err := strconv.Atoi(id); err != nil {
						t.Errorf("expected id to be an integer, got %s", id)
					} else {
						idUint = uint8(idInt)
					}
					p, err := process.For(idUint, name, "testdata/valid")
					if err != nil {
						t.Errorf("expected no error, got %v", err)
					}
					logsConfig := p.Config().GetStringMap("logs.formats")
					if logsConfig == nil {
						t.Errorf("expected logs configuration, got nil")
					}
				})
			}
		}
	})
	t.Run("invalid configuration returns error", func(t *testing.T) {
		_, err := process.FromConfig("testdata/invalid")
		if err != process.ErrInvalidConfigFile {
			t.Errorf("expected error %v, got %v", process.ErrInvalidConfigFile, err)
		}
	})
	t.Run("missing configuration directory returns error", func(t *testing.T) {
		_, err := process.FromConfig("testdata/missing")
		if err != process.ErrConfigDirNotFound {
			t.Errorf("expected error %q, got %v\n", process.ErrConfigDirNotFound, err)
		}
	})
	t.Run("missing workgroup file returns error", func(t *testing.T) {
		_, err := process.FromConfig("testdata")
		if err != process.ErrConfigFileNotFound {
			t.Errorf("expected error %q, got %v\n", process.ErrConfigFileNotFound, err)
		}
	})
	t.Run("missing configDir returns error", func(t *testing.T) {
		i, err := process.For(uint8(1), "tabadmincontroller", "testdata/valid")
		if err == nil || i != nil {
			t.Errorf("expected error %v, got %v", process.ErrConfigDirNotFound, err)
		}
	})
}

func TestLog4j2Config(t *testing.T) {
	t.Run("log4j2.xml returns valid configuration", func(t *testing.T) {
		_, err := process.GetLog4j2Config("testdata/valid/tabadmincontroller_0.20221.22.0712.0324/log4j2.xml")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
	t.Run("controlapp.log4j2.xml returns valid configuration", func(t *testing.T) {
		_, err := process.GetLog4j2Config("testdata/valid/tabadmincontroller_0.20221.22.0712.0324/controlapp.log4j2.xml")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
	t.Run("invalid configuration returns error", func(t *testing.T) {
		_, err := process.GetLog4j2Config("testdata/invalid/bad-log4j2.xml")
		if err != process.ErrInvalidConfigFile {
			t.Errorf("expected error %v, got %v", process.ErrInvalidConfigFile, err)
		}
	})
}

func TestLog4jConfig(t *testing.T) {
	t.Run("log4j.xml returns valid configuration", func(t *testing.T) {
		_, err := process.GetLog4jConfig("testdata/valid/interactive_0.20221.22.0712.0324/log4j.xml")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
	t.Run("invalid configuration returns error", func(t *testing.T) {
		_, err := process.GetLog4jConfig("testdata/invalid/bad-log4j.xml")
		if err != process.ErrInvalidConfigFile {
			t.Errorf("expected error %v, got %v", process.ErrInvalidConfigFile, err)
		}
	})
}

func TestHttpdConfig(t *testing.T) {
	t.Run("httpd.conf returns valid configuration", func(t *testing.T) {
		_, err := process.GetHttpdConfig("testdata/valid/gateway_0.20221.22.0712.0324/httpd.conf")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
	t.Run("invalid configuration returns error", func(t *testing.T) {
		_, err := process.GetHttpdConfig("testdata/invalid/bad-httpd.conf")
		if err != process.ErrInvalidConfigFile {
			t.Errorf("expected error %v, got %v", process.ErrInvalidConfigFile, err)
		}
	})
}

func TestInstance_GetLogFormat(t *testing.T) {
	i, err := process.FromConfig("testdata/valid/tabadmincontroller_0.20221.22.0712.0324")
	if err != nil {
		t.Fatalf("failed to create test instance: %v", err)
	}

	t.Run("named log format", func(t *testing.T) {
		logfile := "testdata/logs/tabadmincontroller/tabadmincontroller_node1-0.log"
		logFormat := i.GetLogFormat(logfile)
		want := "(?P<date>\\d{2}\\d{2}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}\\.\\d{3} [+-]\\d{4}) (?P<pid>.*?) (?P<thread>\\S*) : (?P<level>\\w+)\\s* (?P<logger>\\S+) - (?s)(?P<message>.*)(?-s)"
		if logFormat != want {
			t.Errorf("expected log format to be %s, got %s", want, logFormat)
		}
	})

	t.Run("json log format", func(t *testing.T) {
		logfile := "testdata/logs/tabadmincontroller/tabadmincontroller-metrics_node1-0.log"
		logFormat := i.GetLogFormat(logfile)
		want := "json"
		if logFormat != want {
			t.Errorf("expected log format to be %s, got %s", want, logFormat)
		}
	})

	t.Run("edge cases and empty files", func(t *testing.T) {
		tempDir := t.TempDir()

		tests := []struct {
			name        string
			fileContent string
			wantFormat  string
		}{
			{
				name:        "empty_log_file",
				fileContent: "",
				wantFormat:  "",
			},
			{
				name:        "only_newline",
				fileContent: "\n",
				wantFormat:  "",
			},
			{
				name:        "multiple_newlines",
				fileContent: "\n\n\n",
				wantFormat:  "",
			},
			{
				name:        "only_spaces_no_newline",
				fileContent: "   ",
				wantFormat:  "",
			},
			{
				name:        "only_tabs_no_newline",
				fileContent: "\t\t\t",
				wantFormat:  "",
			},
			{
				name:        "whitespace_with_newline",
				fileContent: "   \n",
				wantFormat:  "",
			},
			{
				name:        "tabs_with_newline",
				fileContent: "\t\t\n",
				wantFormat:  "",
			},
			{
				name:        "mixed_whitespace_with_newline",
				fileContent: " \t \n",
				wantFormat:  "",
			},
			{
				name:        "valid_json_log",
				fileContent: `{"timestamp":"2024-01-01","level":"INFO","message":"test"}`,
				wantFormat:  "json",
			},
			{
				name:        "json_with_newline",
				fileContent: "{\"timestamp\":\"2024-01-01\",\"level\":\"INFO\",\"message\":\"test\"}\n",
				wantFormat:  "json",
			},
			{
				name:        "valid_log4j_format",
				fileContent: "2024-01-01 12:00:00.123 INFO [main] com.example.Main - Starting application",
				wantFormat:  "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				testFile := tempDir + "/" + tt.name + ".log"
				if err := os.WriteFile(testFile, []byte(tt.fileContent), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}

				got := i.GetLogFormat(testFile)
				if got != tt.wantFormat {
					t.Errorf("GetLogFormat() = %q, want %q", got, tt.wantFormat)
				}
			})
		}
	})

	//t.Run("generic log format", func(t *testing.T) {
	// There do not seem to be any of these in the wild.
	//})
}

func TestHttpdLogs(t *testing.T) {
	t.Run("httpd logs", func(t *testing.T) {
		i, err := process.FromConfig("testdata/valid/gateway_0.20221.22.0712.0324")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		logfile := "testdata/logs/httpd/access.2022_08_03_00_00_00.log"
		logFormat := i.GetLogFormat(logfile)
		want := `(?P<requested_hostname>\S+) (?P<remote_hostname>\S+) (?P<remote_user>\S+) (?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}) \"(?P<timezone>[\+\-]\d{4})\" (?P<request_port>\d+) \"(?P<request>[^\"]+)\" \"(?P<xff>\S*)\" (?P<status>\d{3}) (?P<bytes>\d+|-) \"(?P<content_length>\d+|-)\" (?P<ms>\d+) (?P<unique_id>\S+) (?P<tableau_error_source>\S+) (?P<tableau_status_code>\S+) (?P<tableau_error_code>\S+) (?P<tableau_service_name>\S+) \"(?P<tableau_trace_id>\S+)\"`
		if logFormat != want {
			t.Errorf("expected log format to be %s, got %s", want, logFormat)
		}
	})
}
