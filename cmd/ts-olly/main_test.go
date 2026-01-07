package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/nxadm/tail"
	"github.com/rs/zerolog"
	"io"
	"strconv"
	"testing"
	"time"
)

func TestLogOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := zerolog.New(buf)
	fileid := 0xa
	fields := map[string]interface{}{
		"level":     "trace",
		"component": "test-component",
		"filename":  "test-filename.txt",
		"fileid":    "a",
		"process":   "test-process",
		"processid": uint8(0),
		"line":      1,
		"offset":    int64(20),
		"message":   "this is a TRACE line",
	}
	log := func(f map[string]interface{}) {
		outputLine(logger, line{
			Line: &tail.Line{
				Text: f["message"].(string),
				Num:  f["line"].(int),
				SeekInfo: tail.SeekInfo{
					Offset: f["offset"].(int64),
					Whence: io.SeekStart,
				},
				Time: time.Now(),
				Err:  nil,
			},
			filename:    f["filename"].(string),
			fileId:      fileId(uint64(fileid)),
			processName: f["process"].(string),
			processId:   f["processid"].(uint8),
			component:   f["component"].(string),
		})
	}
	t.Run("outputs match log lines", func(t *testing.T) {
		// The logger's output level matches the level of the log line.
		// The logger's output contains the line's node, level, component, fileId, process name, process id, line number, offset, and message.
		// Log lines with formats that aren't parsed have the "message" field set to the raw log line as a string.
		defer buf.Truncate(0)

		log(fields)
		for key, value := range fields {
			t.Run("loggers output contains "+key, func(t *testing.T) {
				if _, err := strconv.Atoi(fmt.Sprintf("%v", value)); err != nil {
					value = fmt.Sprintf("%q", value)
				} else {
					value = fmt.Sprintf("%d", value)
				}
				expected := fmt.Sprintf(`"%s":%v`, key, value)
				if !bytes.Contains(buf.Bytes(), []byte(expected)) {
					t.Errorf("expected %s to be in %s", expected, buf.Bytes())
				}
			})
		}
	})

	t.Run("outputs do not contain level if no level in log line", func(t *testing.T) {
		// If a log line has no level, the logger's output does not contain a level.
		defer buf.Truncate(0)
		fields := fields
		fields["level"] = ""
		fields["message"] = "this is a line without a level"

		log(fields)
		if bytes.Contains(buf.Bytes(), []byte(`"level":`)) {
			t.Errorf("expected level to be absent in %s", buf.Bytes())
		}
	})
	t.Run("output message is json object when log line is json", func(t *testing.T) {
		// JSON log lines are embedded as the "message" field of the logger's output.
		defer buf.Truncate(0)
		fields := fields
		fields["message"] = `{"foo": "bar"}`

		log(fields)
		if !bytes.Contains(buf.Bytes(), []byte(`"message":{"foo": "bar"}`)) {
			t.Errorf(`expected "message":{"foo": "bar"} to be in %s`, buf.Bytes())
		}

	})
	// If a log line is erroneous, the logger's output represents the error.
	t.Run("outputs error when log line is erroneous", func(t *testing.T) {
		defer buf.Truncate(0)
		outputLine(logger, line{
			Line: &tail.Line{
				Err: errors.New("this is an error"),
			},
		})
		if !bytes.Contains(buf.Bytes(), []byte(`"error":"this is an error"`)) {
			t.Errorf("expected error to be in %s", buf.Bytes())
		}
		if !bytes.Contains(buf.Bytes(), []byte(`"level":"error"`)) {
			t.Errorf("expected level to be error in %s", buf.Bytes())
		}
	})

	// If a log line's level is invalid, the logger's output sends an error for that
	t.Run("outputs error when log line's level is not valid", func(t *testing.T) {
		defer buf.Truncate(0)
		fields := fields
		fields["level"] = "not-a-level"
		fields["message"] = `{"sev":"not-a-level"}`

		log(fields)
		invalidLevelError := "Unknown Level String" // string from zerolog/log.go in ParseLevel function
		if !bytes.Contains(buf.Bytes(), []byte(invalidLevelError)) {
			t.Errorf("expected error %q to be in output %s", invalidLevelError, buf.Bytes())
		}
	})

	// If a log line's level is invalid, the logger's output still includes the level.
	t.Run("still outputs the invalid level when the log line's level is not valid", func(t *testing.T) {
		defer buf.Truncate(0)
		fields := fields
		fields["level"] = "not-a-level"
		fields["message"] = `{"sev":"not-a-level"}`

		log(fields)
		expected := `"level":"not-a-level"`
		if !bytes.Contains(buf.Bytes(), []byte(expected)) {
			t.Errorf("expected %q to be in output %s", expected, buf.Bytes())
		}
	})
}
