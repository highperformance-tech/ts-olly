package httpd

import (
	"regexp"
	"testing"
)

var testLog string = "tableau.example.com 127.0.0.6 - 2022-08-02T15:16:44.042 \"+0000\" 8080 \"POST /vizportal/api/web/v1/getSessionInfo HTTP/1.1\" \"192.168.90.27\" 401 35 \"39\" 5795 Yuk_3LxNY7cqNJiackGbNAAAAKY Client 16 0x285DFA77 vizportal \"-\""
var testFormat string = "%V %h %u %{%Y-%m-%dT%X}t.%{msec_frac}t \"%{%z}t\" %p \"%r\" \"%{X-Forwarded-For}i\" %>s %b \"%{Content-Length}i\" %D %{UNIQUE_ID}e %{tableau_error_source}o %{tableau_status_code}o %{tableau_error_code}o %{tableau_service_name}o \"%{X-Tableau-Trace-Id}i\""

func TestRegex(t *testing.T) {
	result := Regexp(testFormat)
	re, err := regexp.Compile(result)
	if err != nil {
		t.Errorf("Regexp failed to compile: %s", err)
	}
	matched := re.MatchString(testLog)
	if !matched {
		t.Errorf("Regexp failed to match: %s", testLog)
	}
}

func TestMatch(t *testing.T) {
	pattern := `(?P<requested_hostname>\S+) (?P<remote_hostname>\S+) (?P<remote_user>\S+) (?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}) \"(?P<timezone>[\+\-]\d{4})\" (?P<request_port>\d+) \"(?P<request>[^\"]+)\" \"(?P<xff>\S*)\" (?P<status>\d{3}) (?P<bytes>\d+|-) \"(?P<content_length>\d+|-)\" (?P<ms>\d+) (?P<unique_id>\S+) (?P<tableau_error_source>\S+) (?P<tableau_status_code>\S+) (?P<tableau_error_code>\S+) (?P<tableau_service_name>\S+) \"(?P<tableau_trace_id>\S+)\"`
	line := `localhost 127.0.0.1 - 2022-08-03T00:00:02.638 "+0000" 8080 "HEAD /favicon.ico HTTP/1.1" "-" 200 - "-" 335 Yum6grxNY7cqNJiackGy6wAAAJs - - - - "-"`
	matched, err := regexp.MatchString(pattern, line)
	if err != nil {
		t.Errorf("Regexp failed to compile: %s", err)
	}
	if !matched {
		t.Errorf("Regexp failed to match: %s", line)
	}
}
