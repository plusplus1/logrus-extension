package logrus_extension

import (
	"io/ioutil"
)

import (
	"github.com/sirupsen/logrus"
)

const (
	defaultLoggerName = "root"
)

func NewKvTextFormatter() logrus.Formatter {
	kvf := &kvTextFormatter{}
	kvf.Do(kvf.reset)
	return kvf
}

// InitFileHook, reset from config file
func InitFileHook(configYaml string) {
	InitFileHookWithLogger(configYaml, defaultLoggerName)
}

// InitFileHookWithLogger, reset logger
func InitFileHookWithLogger(configYaml string, loggerName string) {
	logger := getLogger(loggerName)
	logger.Out = ioutil.Discard             // avoid to print log to screen
	logger.Formatter = NewKvTextFormatter() // set kv formatter
	configLogger(configYaml, logger)        // add file log hook
}

// GetLogger
func GetLogger(loggerName string) logrus.FieldLogger {
	return getLogger(loggerName)
}
