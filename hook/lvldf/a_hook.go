package lvldf

import (
	"log"
	"os"
)
import (
	"github.com/sirupsen/logrus"
)

const (
	minLevel    = logrus.ErrorLevel
	rotateByDay = "D"
)

var (
	levelFileSuffixMap = map[logrus.Level]string{
		logrus.DebugLevel: logrus.DebugLevel.String(),
		logrus.InfoLevel:  logrus.InfoLevel.String(),
		logrus.WarnLevel:  logrus.WarnLevel.String()[0:4],
		logrus.ErrorLevel: logrus.ErrorLevel.String(),
	}
)

// levelDividedFileLogger: every single file for each level of log message
type levelDividedFileLogger struct {
	logName      string // log name , with level suffix
	logDirectory string // log directory

	level     logrus.Level                  // min level
	allLevels []logrus.Level                // all supported levels
	writers   map[logrus.Level]*mutexWriter // writers, do real work

	// log split related fields
	rotateEnable bool
	rotateBy     string

	// buf enable
	bufferEnable bool
}

// InitLevelDividedFileLogger, init
func InitLevelDividedFileLogger(configYaml string) {
	InitLevelDividedFileLoggerV2(configYaml, nil)
}

// InitLevelDividedFileLoggerV2, init
func InitLevelDividedFileLoggerV2(configYaml string, lg *logrus.Logger) {

	logger := &levelDividedFileLogger{writers: make(map[logrus.Level]*mutexWriter)}

	if configYaml != "" {
		cfg := newConfig(configYaml)
		logger.logName = cfg.Filename
		logger.logDirectory = cfg.Directory

		os.MkdirAll(logger.logDirectory, 0775)
		if info, err := os.Stat(logger.logDirectory); err != nil {
			log.Panicf("make log directory fail, error=%v", err)
		} else if !info.IsDir() {
			log.Panicf("%s is not a directory", logger.logDirectory)
		}

		if lvl, e := logrus.ParseLevel(cfg.Level); e != nil {
			log.Panicf("parse config level fail, error=%v", e.Error())
		} else {
			logrus.SetLevel(lvl)
			logger.level = lvl
		}
		if len(logger.Levels()) < 1 {
			log.Panicf("no valid levels for hook")
		}

		if cfg.Daily {
			logger.rotateEnable = true
			logger.rotateBy = rotateByDay
		}

		if cfg.Buffer {
			logger.bufferEnable = true
		}
	}

	for _, lvl := range logger.Levels() {
		logger.writers[lvl] = newMutexWriter(lvl, logger)
	}

	if lg == nil {
		logrus.AddHook(logger)
	} else {
		lg.AddHook(logger)
	}
	return
}

// Levels, get all supported log levels
func (ldf *levelDividedFileLogger) Levels() []logrus.Level {
	if (len(ldf.allLevels)) < 1 {
		for _, lvl := range logrus.AllLevels {
			if lvl <= ldf.level && lvl >= minLevel {
				ldf.allLevels = append(ldf.allLevels, lvl)
			}
		}
	}
	return ldf.allLevels
}

// Fire, start to write log
func (ldf *levelDividedFileLogger) Fire(entry *logrus.Entry) (err error) {
	if w, ok := ldf.writers[entry.Level]; ok {
		var msg string
		if msg, err = entry.String(); err != nil {
			return err
		}
		err = w.writeMessage(entry.Level, entry.Time, msg)
	}
	return
}
