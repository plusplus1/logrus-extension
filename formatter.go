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

// kvTextFormatter formats logs into text
type kvTextFormatter struct {
	sync.Once

	enableLogTime bool
	sortKeys      bool
	kvSeparator   byte
	hostName      []byte
	logTimeFormat string
}

func (kvf *kvTextFormatter) reset() {
	kvf.sortKeys = true
	kvf.enableLogTime = true
	kvf.logTimeFormat = "2006-01-02 15:04:05.000"
	kvf.kvSeparator = '\t'
	host, _ := os.Hostname()
	kvf.hostName = []byte(host)
}

// Format renders a single log entry
func (kvf *kvTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	keys := make([]string, len(entry.Data))
	var i = 0
	for k := range entry.Data {
		keys[i] = k
		i++
	}

	if kvf.sortKeys {
		sort.Strings(keys)
	}

	var buf = entry.Buffer
	if buf == nil {
		buf = &bytes.Buffer{}
	}
	if kvf.enableLogTime {
		kvf.appendKeyValue(buf, fieldKeyLogTime, entry.Time.Format(kvf.logTimeFormat))
	}

	if name, ok := levelNameDict[entry.Level]; ok {
		kvf.appendKeyValue(buf, fieldKeyLevel, name)
	} else {
		kvf.appendKeyValue(buf, fieldKeyLevel, entry.Level.String())
	}
	kvf.appendKeyValue(buf, fieldKeyHost, kvf.hostName)

	for _, key := range keys {
		if value := entry.Data[key]; value != nil {
			switch v := value.(type) {
			case error:
				kvf.appendKeyValue(buf, fieldKeyError, v.Error())
			default:
				kvf.appendKeyValue(buf, key, v)
			}
		}
	}

	if entry.Message != "" {
		kvf.appendKeyValue(buf, fieldKeyMessage, entry.Message)
	}

	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

func (kvf *kvTextFormatter) appendKeyValue(buf *bytes.Buffer, key string, value interface{}) {
	if buf.Len() > 0 {
		buf.WriteByte(kvf.kvSeparator)
	}

	buf.WriteString(key)
	buf.WriteByte('=')

	if b, ok := value.([]byte); ok {
		buf.Write(b)
		return
	}

	if s, ok := value.(string); ok {
		buf.WriteString(s)
		return
	}

	if o, ok := value.(interface{ Bytes() []byte }); ok && o != nil {
		buf.Write(o.Bytes())
		return
	}

	if o, ok := value.(interface{ String() string }); ok && o != nil {
		buf.WriteString(o.String())
		return
	}

	buf.WriteString(fmt.Sprint(value))
	return
}
