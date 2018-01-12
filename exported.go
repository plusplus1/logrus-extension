package logrus_extension

import (
	"io/ioutil"
)

import (
	"github.com/plusplus1/logrus-extension/format"
	"github.com/plusplus1/logrus-extension/hook/lvldf"
)

import (
	"github.com/sirupsen/logrus"
)

var (
	loggerMap = make(map[string]*logrus.Logger)
)

// InitFileHook, init from config file
func InitFileHook(configYaml string) {
	InitFileHookWithLogger(configYaml, "")
}

// InitFileHookWithLogger, init logger
func InitFileHookWithLogger(configYaml string, loggerName string) {

	logger := GetLogger(loggerName)
	logger.Out = ioutil.Discard                            // avoid to print log to screen
	logger.Formatter = format.NewKvTextFormatter()         // set kv formatter
	lvldf.InitLevelDividedFileLoggerV2(configYaml, logger) // add file log hook

}

func GetLogger(loggerName string) *logrus.Logger {
	if loggerName == "" || loggerName == "root" || loggerName == "ROOT" {
		return logrus.StandardLogger()
	}
	if logger, ok := loggerMap[loggerName]; ok {
		return logger
	}

	logger := logrus.New()
	loggerMap[loggerName] = logger
	return logger
}
