package logrus_extension

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"sync"
)

import (
	"github.com/sirupsen/logrus"
)

const (
	fieldKeyLogTime = "log_time"
	fieldKeyLevel   = "level"
	fieldKeyHost    = "host"
	fieldKeyMessage = "msg"
	fieldKeyError   = "error"
)

type (
	// kvTextFormatter formats logs into text
	kvTextFormatter struct {
		// Disable timestamp logging. useful when output is redirected to logging
		// system that already adds timestamps.
		DisableTimestamp bool

		// TimestampFormat to use for display when a full timestamp is printed
		TimestampFormat string

		// The fields are sorted by default for a consistent output. For applications
		// that log extremely frequently and don't use the JSON formatter this may not
		// be desired.
		DisableSorting bool

		KVSeparator byte

		sync.Once

		hostName string
	}
)

func NewKvTextFormatter() *kvTextFormatter {
	return &kvTextFormatter{}
}

func (kvFmt *kvTextFormatter) init(entry *logrus.Entry) {
	if kvFmt.TimestampFormat == "" {
		kvFmt.TimestampFormat = "2006-01-02 15:04:05"
	}
	if kvFmt.KVSeparator == byte(0) {
		kvFmt.KVSeparator = '\t'
	}
	if kvFmt.hostName == "" {
		kvFmt.hostName, _ = os.Hostname()
	}
}

// Format renders a single log entry
func (kvFmt *kvTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var buf *bytes.Buffer

	keys := make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}

	if !kvFmt.DisableSorting {
		sort.Strings(keys)
	}
	if entry.Buffer != nil {
		buf = entry.Buffer
	} else {
		buf = &bytes.Buffer{}
	}

	kvFmt.Do(func() { kvFmt.init(entry) })

	if !kvFmt.DisableTimestamp {
		kvFmt.appendKeyValue(buf, fieldKeyLogTime, entry.Time.Format(kvFmt.TimestampFormat))
	}
	if entry.Level == logrus.WarnLevel {
		kvFmt.appendKeyValue(buf, fieldKeyLevel, entry.Level.String()[0:4])
	} else {
		kvFmt.appendKeyValue(buf, fieldKeyLevel, entry.Level.String())
	}
	kvFmt.appendKeyValue(buf, fieldKeyHost, kvFmt.hostName)

	for _, key := range keys {
		if value := entry.Data[key]; value != nil {
			switch v := value.(type) {
			case error:
				kvFmt.appendKeyValue(buf, fieldKeyError, v.Error())
			default:
				kvFmt.appendKeyValue(buf, key, v)
			}
		}
	}

	if entry.Message != "" {
		kvFmt.appendKeyValue(buf, fieldKeyMessage, entry.Message)
	}

	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

func (kvFmt *kvTextFormatter) needsQuoting(text string) bool {
	if len(text) == 0 {
		return false
	}
	for _, ch := range text {
		if ch == '\t' || ch == '\r' || ch == '\n' {
			return true
		}
	}
	return false
}

func (kvFmt *kvTextFormatter) appendKeyValue(buf *bytes.Buffer, key string, value interface{}) {
	if buf.Len() > 0 {
		buf.WriteByte(kvFmt.KVSeparator)
	}

	buf.WriteString(key)
	buf.WriteByte('=')
	if key == fieldKeyLogTime {
		buf.WriteString(fmt.Sprint(value))
	} else {
		kvFmt.appendValue(buf, value)
	}
}

func (kvFmt *kvTextFormatter) appendValue(buf *bytes.Buffer, value interface{}) {
	stringVal, ok := value.(string)
	if !ok {
		stringVal = fmt.Sprint(value)
	}

	if !kvFmt.needsQuoting(stringVal) {
		buf.WriteString(stringVal)
	} else {
		buf.WriteString(fmt.Sprintf("%q", stringVal))
	}
}
