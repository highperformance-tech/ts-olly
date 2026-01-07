package httpd

import "strings"

// patterns: "%V %h %u %{%Y-%m-%dT%X}t.%{msec_frac}t \"%{%z}t\" %p \"%r\" \"%{X-Forwarded-For}i\" %>s %b \"%{Content-Length}i\" %D %{UNIQUE_ID}e ${TAB_ERR_ANNOTATIONS} \"%{X-Tableau-Trace-Id}i\""
// httpdPatterns is a map of functions that will replace a logs pattern with
// a Go regexp pattern.
var httpdPatterns = map[string]string{
	`%V`:                            `(?P<requested_hostname>\S+)`,
	`%h`:                            `(?P<remote_hostname>\S+)`,
	`%u`:                            `(?P<remote_user>\S+)`,
	`%{%Y-%m-%dT%X}t.%{msec_frac}t`: `(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3})`,
	`%{%z}t`:                        `(?P<timezone>[\+\-]\d{4})`,
	`%p`:                            `(?P<request_port>\d+)`,
	`%r`:                            `(?P<request>[^\"]+)`,
	`%{X-Forwarded-For}i`:           `(?P<xff>\S*)`,
	`%>s`:                           `(?P<status>\d{3})`,
	`%b`:                            `(?P<bytes>\d+|-)`,
	`%{Content-Length}i`:            `(?P<content_length>\d+|-)`,
	`%D`:                            `(?P<ms>\d+)`,
	`%{UNIQUE_ID}e`:                 `(?P<unique_id>\S+)`,
	`%{tableau_error_source}o`:      `(?P<tableau_error_source>\S+)`,
	`%{tableau_status_code}o`:       `(?P<tableau_status_code>\S+)`,
	`%{tableau_error_code}o`:        `(?P<tableau_error_code>\S+)`,
	`%{tableau_service_name}o`:      `(?P<tableau_service_name>\S+)`,
	`%{X-Tableau-Trace-Id}i`:        `(?P<tableau_trace_id>\S+)`,
}

func Regexp(pattern string) string {
	return replacePatterns(pattern)
}

func replacePatterns(message string) string {
	for k, v := range httpdPatterns {
		message = strings.Replace(message, k, v, -1)
	}
	return message
}
