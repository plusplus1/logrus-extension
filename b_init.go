package logrus_extension

import (
	"strings"
)

import (
	"github.com/sirupsen/logrus"
)

var (
	loggerMap     = make(map[string]*logrus.Logger)
	levelNameDict = map[logrus.Level]string{
		logrus.DebugLevel: logrus.DebugLevel.String(),
		logrus.InfoLevel:  logrus.InfoLevel.String(),
		logrus.WarnLevel:  logrus.WarnLevel.String()[0:4],
		logrus.ErrorLevel: logrus.ErrorLevel.String(),
	}
)

func getLogger(loggerName string) (logger *logrus.Logger) {
	if loggerName = strings.ToLower(loggerName); loggerName == "" {
		loggerName = "root"
	}

	for {
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

func configLogger(configFile string, logger *logrus.Logger) {
	if conf, err := newLvlConf(configFile); err != nil {
		log.Panicf("configLogger failed, %v", err)
	} else {
		hook := newLevelFilteredTimeRotatedHook(conf)
		level, _ := logrus.ParseLevel(conf.Level)
		logger.SetLevel(level)
		logger.AddHook(hook)
	}
}
