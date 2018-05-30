package logrus_extension

import (
	"io/ioutil"
)

import (
	"github.com/sirupsen/logrus"
)

var (
	loggerMap = make(map[string]*logrus.Logger)
)

// InitFileHook, init from config file
func InitFileHook(configYaml string) {
	InitFileHookWithLogger(configYaml, "root")
}

// InitFileHookWithLogger, init logger
func InitFileHookWithLogger(configYaml string, loggerName string) {

	logger := initLogger(loggerName)
	logger.Out = ioutil.Discard                   // avoid to print log to screen
	logger.Formatter = NewKvTextFormatter()       // set kv formatter
	registerLvlHookWithLogger(configYaml, logger) // add file log hook
}

func GetLogger(loggerName string) (logger *logrus.Logger) {
	if loggerName == "" || loggerName == "root" || loggerName == "ROOT" {
		loggerName = "root"
	}
	if lg, ok := loggerMap[loggerName]; ok && lg != nil {
		return lg
	}
	return logrus.StandardLogger()
}

func initLogger(loggerName string) (logger *logrus.Logger) {
	if loggerName == "" || loggerName == "root" || loggerName == "ROOT" {
		loggerName = "root"
	}

	for i := 0; i < 1; i++ {
		if lg, ok := loggerMap[loggerName]; ok && lg != nil {
			logger = lg
			break
		}
		if loggerName == "root" {
			logger = logrus.StandardLogger()
			break
		}
		logger = logrus.New()
		break
	}

	loggerMap[loggerName] = logger

	return
}
