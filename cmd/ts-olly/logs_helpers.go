package main

import (
	"github.com/highperformance-tech/ts-olly/internal/fileid"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func getFileId(path string) (fileId, error) {
	fid, err := fileid.Query(path)
	return fileId(fid), err
}

func getProcessName(path, logsDir string) string {
	path = strings.Replace(path, logsDir+string(filepath.Separator), "", 1)
	path, _, ok := strings.Cut(path, string(filepath.Separator))
	if !ok {
		return ""
	}
	if path == "httpd" {
		return "gateway"
	}
	return path
}

func getProcessId(filename string) uint8 {
	// Matches consecutive groups of numbers like dates/times/counts, as well as node IDs. If we strip these, the only numbers left are the process IDs.
	pattern := `(\d{4}[_-]\d{2}[_-]\d{2}[_-]\d{2}[_-]\d{2}[_-]\d{2}[_-]\d{2})|(\d{4}[_-]\d{2}[_-]\d{2}[_-]\d{2}[_-]\d{2}[_-]\d{2})|(\d{4}[_-]\d{2}[_-]\d{2}[_-]?\d*)|(node\d+)|(\d+$)|(_\d+-)`
	re := regexp.MustCompile(pattern)
	filename = re.ReplaceAllString(filename, "")
	if idInt, err := strconv.ParseUint(regexp.MustCompile(`\d+`).FindString(filename), 10, 8); err == nil {
		return uint8(idInt)
	}
	return 0
}

func getComponent(filename string) string {
	var component string
	switch {
	case strings.HasPrefix(filename, "tomcat_"):
		component = "tomcat"
	case strings.HasPrefix(filename, "stdout_"):
		component = "stdout"
	case strings.HasPrefix(filename, "control_"):
		component = "control"
	case strings.HasPrefix(filename, "nativeapi_"):
		component = "nativeapi"
	case strings.HasPrefix(filename, "tabprotosrv_"):
		component = "tabprotosrv"
	case strings.Contains(filename, "instrumentation-metrics_"):
		component = "instrumentation-metrics"
	case strings.Contains(filename, "metrics_"):
		component = "metrics"
	case strings.Contains(filename, "discovery_"):
		component = "discovery"
	case strings.Contains(filename, "oauth-service"):
		component = "oauth-service"
	case strings.Contains(filename, "audit-history_"):
		component = "audit-history"
	case strings.Contains(filename, "vizql-client"):
		component = "vizql-client"
	case strings.Contains(filename, "checklicense"):
		component = "checklicense"
	default:
		component = ""
	}
	return component
}

func getLevel(message string) string {
	var level string
	if strings.HasPrefix(message, "{") && strings.HasSuffix(message, "}") {
		messageFields := strings.FieldsFunc(message, func(r rune) bool {
			//return r == '-' || r == '_' || r == '.'
			return r == '{' || r == ',' || r == '}'
		})
		for _, field := range messageFields {
			if strings.HasPrefix(field, `"sev":"`) {
				if len(field) >= 8 {
					level = field[7 : len(field)-1]
				}
				break
			}
		}
	}
	pos := -1

	levels := []string{"TRACE", "DEBUG", "INFO", "WARN", "WARNING", "ERROR", "FATAL"}
	if level == "" {
		for _, l := range levels {
			if levelPos := strings.Index(message, l); levelPos != -1 && (levelPos < pos || pos == -1) {
				pos = levelPos
				level = strings.ToLower(l)
			}
		}
	}
	if level == "warning" {
		level = "warn"
	}
	return level
}
